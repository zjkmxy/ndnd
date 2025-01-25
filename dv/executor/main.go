package executor

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	if configfile == "" {
		flagset.Usage()
		os.Exit(3)
	}

	config := DefaultConfig()
	toolutils.ReadYaml(&config, configfile)

	dve, err := NewDvExecutor(config)
	if err != nil {
		panic(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	quitchan := make(chan bool, 1)
	go func() {
		if err = dve.Start(); err != nil {
			panic(err)
		}
		quitchan <- true
	}()

	for {
		select {
		case <-sigchan:
			dve.Stop()
		case <-quitchan:
			return
		}
	}
}
