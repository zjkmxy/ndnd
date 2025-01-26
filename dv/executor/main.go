package executor

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/named-data/ndnd/dv/config"
	"github.com/named-data/ndnd/std/utils/toolutils"
)

func Main(args []string) {
	flagset := flag.NewFlagSet("ndn-dv", flag.ExitOnError)
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", args[0])
		flagset.PrintDefaults()
	}
	flagset.Parse(args[1:])

	configfile := flagset.Arg(0)
	if flagset.NArg() != 1 || configfile == "" {
		flagset.Usage()
		os.Exit(3)
	}

	config := struct {
		Config *config.Config `json:"dv"`
	}{
		Config: config.DefaultConfig(),
	}
	toolutils.ReadYaml(&config, configfile)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	dve, err := NewDvExecutor(config.Config)
	if err != nil {
		panic(err)
	}

	go func() {
		dve.Start()
		sigchan <- syscall.SIGTERM
	}()

	// wait for interrupt
	<-sigchan
	dve.Stop()
	<-sigchan
}
