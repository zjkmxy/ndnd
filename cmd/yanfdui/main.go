/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"flag"
	"fmt"
	"log"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/named-data/YaNFD/executor"
)

// Version of YaNFD.
var Version string

func main() {
	// Parse command line options
	var shouldPrintVersion bool
	flag.BoolVar(&shouldPrintVersion, "version", false, "Print version and exit")
	var configFileName string
	flag.StringVar(&configFileName, "config", "/usr/local/etc/ndn/yanfd.toml", "Configuration file location")
	var disableEthernet bool
	flag.BoolVar(&disableEthernet, "disable-ethernet", false, "Disable Ethernet transports")
	var disableUnix bool
	flag.BoolVar(&disableUnix, "disable-unix", false, "Disable Unix stream transports")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version " + Version)
		fmt.Println("Copyright (C) 2020-2021 Eric Newberry")
		fmt.Println("Released under the terms of the MIT License")
		return
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Text = "YaNFD is starting."
	p.SetRect(0, 0, 25, 5)

	ui.Render(p)

	config := executor.YaNFDConfig{
		Version:           Version,
		ConfigFileName:    configFileName,
		DisableEthernet:   disableEthernet,
		DisableUnix:       disableUnix,
		LogFile:           "/var/log/YaNFD.log",
		CpuProfile:        "",
		MemProfile:        "",
		BlockProfile:      "",
		MemoryBallastSize: 0,
	}
	yanfd := executor.NewYaNFD(&config)
	yanfd.Start()

	p.Text = "YaNFD is running."
	ui.Render(p)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}

	yanfd.Stop()
}
