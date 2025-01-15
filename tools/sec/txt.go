package sec

import (
	"fmt"
	"io"
	"os"

	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/security"
)

func txtFrom(args []string) {
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Usage: Provide raw NDN data as input to stdin")
		os.Exit(1)
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(nil, "Failed to read input", "err", err)
		return
	}

	out, err := security.TxtFrom(input)
	if err != nil {
		log.Fatal(nil, "Failed to convert to text", "err", err)
		return
	}

	os.Stdout.Write(out)
}

func txtParse(args []string) {
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Usage: Provide text format as input to stdin")
		os.Exit(1)
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(nil, "Failed to read input", "err", err)
		return
	}

	out := security.TxtParse(input)
	if len(out) == 0 {
		log.Fatal(nil, "No valid NDN data found")
		return
	}
	if len(out) > 1 {
		log.Fatal(nil, "Multiple NDN data found")
		return
	}
	os.Stdout.Write(out[0])
}
