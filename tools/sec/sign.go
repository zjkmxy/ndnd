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

	const start_default = "now"
	const end_default = "now + 1 year"
	const issuer_default = "NA"

	flagset := flag.NewFlagSet("sign-cert", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <key-file>\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Expects CSR input on stdin\n")
		fmt.Fprintf(os.Stderr, "Signer key and CSR can be TLV or PEM encoded\n")
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

	panic("not implemented")
}
