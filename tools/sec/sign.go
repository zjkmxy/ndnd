package sec

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	spec "github.com/named-data/ndnd/std/ndn/spec_2022"
	"github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/keychain"
)

// YYYYMMDDhhmmss
const TIME_LAYOUT = "20060102150405"

func signCert(args []string) {
	flags := struct {
		Start  string
		End    string
		Info   string
		Issuer string
	}{}

	const start_default = "now"
	const end_default = "now + 1 year"
	const issuer_default = "NA"

	flagset := flag.NewFlagSet("sign-cert", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <key-file>\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Expects CSR input on stdin.\n")
		fmt.Fprintf(os.Stderr, "Signer key and CSR can be TLV or PEM encoded.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "CSR can be either a self-signed certificate or a secret key.\n")
		fmt.Fprintf(os.Stderr, "To generate a self-signed certificate, provide the same key\n")
		fmt.Fprintf(os.Stderr, "file as both the signer key and the CSR.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flagset.PrintDefaults()
	}
	flagset.StringVar(&flags.Start, "start", start_default, "Validity start time in YYYYMMDDhhmmss format")
	flagset.StringVar(&flags.End, "end", end_default, "Validity end time in YYYYMMDDhhmmss format")
	flagset.StringVar(&flags.Info, "info", "", "Additional info to be included in the certificate")
	flagset.StringVar(&flags.Issuer, "issuer", issuer_default, "Issuer ID to be included in the certificate name")
	flagset.Parse(args[1:])

	argSigner := flagset.Arg(0)
	if argSigner == "" {
		flagset.Usage()
		os.Exit(2)
	}

	keysFile, err := os.Open(argSigner)
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

	signers, _, _ := keychain.DecodeFile(keysBytes)
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

	if flags.Start != start_default {
		notBefore, err = time.Parse(TIME_LAYOUT, flags.Start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse start time: %s\n", err)
			os.Exit(1)
		}
	}
	if flags.End != end_default {
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
