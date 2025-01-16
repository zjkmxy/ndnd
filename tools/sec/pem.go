package sec

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/named-data/ndnd/std/security"
)

func pemEncode(args []string) {
	flagset := flag.NewFlagSet("pem-encode", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Encodes a TLV NDN Key or Certificate to PEM.\n")
		fmt.Fprintf(os.Stderr, "Provide TLV data as input to stdin.\n")
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %+v\n", err)
		os.Exit(1)
		return
	}

	out, err := security.PemEncode(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to convert to text: %+v\n", err)
		os.Exit(1)
		return
	}

	os.Stdout.Write(out)
}

func pemDecode(args []string) {
	flagset := flag.NewFlagSet("pem-encode", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Decodes a PEM file with a single NDN TLV.\n")
		fmt.Fprintf(os.Stderr, "Provide PEM data as input to stdin.\n")
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input from stdin: %+v\n", err)
		os.Exit(1)
		return
	}

	out := security.PemDecode(input)
	if len(out) == 0 {
		fmt.Fprintf(os.Stderr, "No valid NDN data found in stdin input\n")
		os.Exit(1)
		return
	}
	if len(out) > 1 {
		fmt.Fprintf(os.Stderr, "Multiple NDN data found in stdin input\n")
		os.Exit(1)
		return
	}

	os.Stdout.Write(out[0])
}
