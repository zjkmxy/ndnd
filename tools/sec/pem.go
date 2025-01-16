package sec

import (
	"fmt"
	"io"
	"os"

	"github.com/named-data/ndnd/std/security"
)

func pemEncode(args []string) {
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "    provide raw data as input to stdin\n")
		os.Exit(2)
		return
	}

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
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "    provide PEM input to stdin")
		os.Exit(2)
		return
	}

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
