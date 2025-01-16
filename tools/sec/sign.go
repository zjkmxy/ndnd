package sec

import (
	"flag"
	"fmt"
	"os"
)

func signCert(args []string) {
	flags := struct {
		Start  string
		End    string
		Info   string
		Issuer string
	}{}

	flagset := flag.NewFlagSet("sign-cert", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <key-file>\n", args[0])
		fmt.Fprintf(os.Stderr, "    Expects CSR input on stdin\n")
		fmt.Fprintf(os.Stderr, "    Signer key and CSR can be TLV or PEM encoded\n")
		fmt.Fprintf(os.Stderr, "\n")
		flagset.PrintDefaults()
	}

	flagset.StringVar(&flags.Start, "start", "now", "Validity start time in YYYYMMDDhhmmss format")
	flagset.StringVar(&flags.End, "end", "now+1y", "Validity end time in YYYYMMDDhhmmss format")
	flagset.StringVar(&flags.Info, "info", "", "Additional info to be included in the certificate")
	flagset.StringVar(&flags.Issuer, "issuer", "NA", "Issuer ID to be included in the certificate name")

	flagset.Parse(args[1:])

	if flagset.NArg() != 1 {
		flagset.Usage()
		os.Exit(2)
	}

	signerKey := flagset.Arg(0)
	fmt.Println(signerKey)
	panic("not implemented")
}
