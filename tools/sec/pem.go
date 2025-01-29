package sec

import (
	"fmt"
	"io"
	"os"

	"github.com/named-data/ndnd/std/security"
	"github.com/spf13/cobra"
)

func pemEncode(_ *cobra.Command, args []string) {
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

func pemDecode(_ *cobra.Command, args []string) {
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
