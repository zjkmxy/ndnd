package sec

import (
	"fmt"
	"io"
	"os"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security"
	"github.com/spf13/cobra"
)

// YYYYMMDDhhmmss
const TIME_LAYOUT = "20060102150405"

var signCertFlags = struct {
	Start  string
	End    string
	Info   string
	Issuer string
}{}

func signCert(_ *cobra.Command, args []string) {
	flags := signCertFlags

	keysFile, err := os.Open(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open key file: %s\n", err)
		os.Exit(1)
	}
	defer keysFile.Close()

	keysBytes, err := io.ReadAll(keysFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read key file: %s\n", err)
		os.Exit(1)
	}

	csrsBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read CSR: %s\n", err)
		os.Exit(1)
	}

	signers, _, _ := security.DecodeFile(keysBytes)
	if len(signers) != 1 {
		fmt.Fprintf(os.Stderr, "Expected exactly one key, got %d\n", len(signers))
		os.Exit(1)
	}
	signer := signers[0]

	csrDatasBytes := security.PemDecode(csrsBytes)
	if len(csrDatasBytes) != 1 {
		fmt.Fprintf(os.Stderr, "Expected exactly one CSR, got %d\n", len(csrDatasBytes))
		os.Exit(1)
	}
	csrDataBytes := csrDatasBytes[0]

	csr, _, err := spec.Spec{}.ReadData(enc.NewBufferReader(csrDataBytes))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read CSR: %s\n", err)
		os.Exit(1)
	}

	notBefore := time.Now()
	notAfter := time.Now().AddDate(1, 0, 0)

	if flags.Start != "now" {
		notBefore, err = time.Parse(TIME_LAYOUT, flags.Start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse start time: %s\n", err)
			os.Exit(1)
		}
	}
	if flags.End != "now + 1 year" {
		notAfter, err = time.Parse(TIME_LAYOUT, flags.End)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse end time: %s\n", err)
			os.Exit(1)
		}
	}

	// TODO: set description
	certWire, err := security.SignCert(security.SignCertArgs{
		Signer:    signer,
		Data:      csr,
		IssuerId:  enc.NewStringComponent(enc.TypeGenericNameComponent, flags.Issuer),
		NotBefore: notBefore,
		NotAfter:  notAfter,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sign certificate: %s\n", err)
		os.Exit(1)
	}

	pem, err := security.PemEncode(certWire.Join())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to PEM encode certificate: %s\n", err)
		os.Exit(1)
	}

	os.Stdout.Write(pem)
}
