package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	dv_config "github.com/named-data/ndnd/dv/config"
	dv_executor "github.com/named-data/ndnd/dv/executor"
	fw_core "github.com/named-data/ndnd/fw/core"
	fw_executor "github.com/named-data/ndnd/fw/executor"
	"github.com/named-data/ndnd/std/utils/toolutils"
)

// Daemon runs the combined forwarder-router daemon.
func Daemon(args []string) {
	flagset := flag.NewFlagSet("ndnd-daemon", flag.ExitOnError)
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
		Fw *fw_core.Config   `json:"fw"`
		Dv *dv_config.Config `json:"dv"`
	}{
		Fw: fw_core.DefaultConfig(),
		Dv: dv_config.DefaultConfig(),
	}
	toolutils.ReadYaml(&config, configfile)
	config.Fw.Core.BaseDir = filepath.Dir(configfile)

	// validate unix socket is enabled
	if !config.Fw.Faces.Unix.Enabled {
		panic("Unix socket must be enabled for the combined daemon")
	}

	// point dv to the correct socket
	sock := config.Fw.Faces.Unix.SocketPath
	if runtime.GOOS != "windows" { // wrong slashes
		sock = config.Fw.ResolveRelPath(sock)
	}
	os.Setenv("NDN_CLIENT_TRANSPORT", fmt.Sprintf("unix://%s", sock))

	// setup signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	// yanfd does not block on start
	yanfd := fw_executor.NewYaNFD(config.Fw)
	yanfd.Start()

	// dve blocks on start, so run it in a goroutine
	dve, err := dv_executor.NewDvExecutor(config.Dv)
	if err != nil {
		panic(err)
	}

	go func() {
		dve.Start()
		sigchan <- syscall.SIGTERM
	}()

	// wait for signal
	<-sigchan
	dve.Stop()
	<-sigchan
	yanfd.Stop()
}
