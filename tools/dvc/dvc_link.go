package dvc

import (
	"flag"
	"fmt"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/tools/nfdc"
)

func (t *Tool) RunDvLinkCreate(args []string) {
	flagset := flag.NewFlagSet("dv-link-create", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <neighbor-uri>\n", args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	argUri := flagset.Arg(0)
	if argUri == "" {
		flagset.Usage()
		os.Exit(2)
	}

	t.Start()
	defer t.Stop()

	// Get router status to get network name
	status, err := t.DvStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get router status: %v\n", err)
		os.Exit(1)
	}

	// /localhop/<network>/32=DV/32=ADS/32=ACT
	name := enc.LOCALHOP.Append(status.NetworkName.Name.Append(
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ACT"),
	)...)

	nfdc.Tree().Execute([]string{
		"nfdc", "route", "add",
		"persistency=permanent",
		fmt.Sprintf("face=%s", argUri),
		fmt.Sprintf("prefix=%s", name),
	})
}

func (t *Tool) RunDvLinkDestroy(args []string) {
	flagset := flag.NewFlagSet("dv-link-create", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <neighbor-uri>\n", args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	argUri := flagset.Arg(0)
	if argUri == "" {
		flagset.Usage()
		os.Exit(2)
	}

	// just destroy the face assuming we created it
	nfdc.Tree().Execute([]string{
		"nfdc", "face", "destroy",
		fmt.Sprintf("face=%s", argUri),
	})
}
