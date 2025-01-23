package dvc

import (
	"flag"
	"fmt"
	"os"
	"time"

	spec_dv "github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/engine"
	"github.com/named-data/ndnd/std/ndn"
	"github.com/named-data/ndnd/std/utils"
)

func dvGetStatus() *spec_dv.Status {
	app := engine.NewBasicEngine(engine.NewDefaultFace())
	err := app.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start engine: %v\n", err)
		os.Exit(1)
	}
	defer app.Stop()

	name, _ := enc.NameFromStr("/localhost/nlsr/status")
	cfg := &ndn.InterestConfig{
		MustBeFresh: true,
		Lifetime:    utils.IdPtr(time.Second),
		Nonce:       utils.ConvertNonce(app.Timer().Nonce()),
	}

	interest, err := app.Spec().MakeInterest(name, cfg, nil, nil)
	if err != nil {
		panic(err)
	}

	ch := make(chan ndn.ExpressCallbackArgs)
	err = app.Express(interest, func(args ndn.ExpressCallbackArgs) { ch <- args })
	if err != nil {
		panic(err)
	}
	args := <-ch

	if args.Result != ndn.InterestResultData {
		fmt.Fprintf(os.Stderr, "Failed to get router state. Is DV running?\n")
		os.Exit(1)
	}

	status, err := spec_dv.ParseStatus(enc.NewWireReader(args.Data.Content()), false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse router state: %v\n", err)
		os.Exit(1)
	}

	return status
}

func RunDvStatus(args []string) {
	flagset := flag.NewFlagSet("dv-status", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	status := dvGetStatus()
	fmt.Printf("version = %s\n", status.Version)
	fmt.Printf("routerName = %s\n", status.RouterName.Name)
	fmt.Printf("networkName = %s\n", status.NetworkName.Name)
	fmt.Printf("nRibEntries = %d\n", status.NRibEntries)
	fmt.Printf("nNeighbors = %d\n", status.NNeighbors)
	fmt.Printf("nFibEntries = %d\n", status.NFibEntries)
}

func RunDvLinkCreate(nfdcTree *utils.CmdTree) func([]string) {
	return func(args []string) {
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

		// Get router status to get network name
		status := dvGetStatus() // will panic if fail

		// /localhop/<network>/32=DV/32=ADS/32=ACT
		name := enc.LOCALHOP.Append(status.NetworkName.Name.Append(
			enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
			enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
			enc.NewStringComponent(enc.TypeKeywordNameComponent, "ACT"),
		)...)

		nfdcTree.Execute([]string{
			"nfdc", "route", "add",
			"persistency=permanent",
			fmt.Sprintf("face=%s", argUri),
			fmt.Sprintf("prefix=%s", name),
		})
	}
}

func RunDvLinkDestroy(nfdcTree *utils.CmdTree) func([]string) {
	return func(args []string) {
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
		nfdcTree.Execute([]string{
			"nfdc", "face", "destroy",
			fmt.Sprintf("face=%s", argUri),
		})
	}
}
