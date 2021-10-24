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
	"github.com/named-data/YaNFD/cmd/yanfdui/forms"
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

	config := executor.YaNFDConfig{
		Version:           Version,
		ConfigFileName:    configFileName,
		DisableEthernet:   disableEthernet,
		DisableUnix:       disableUnix,
		LogFile:           "./YaNFD.log",
		CpuProfile:        "",
		MemProfile:        "",
		BlockProfile:      "",
		MemoryBallastSize: 0,
	}
	yanfd := executor.NewYaNFD(&config)
	yanfd.Start()

	// Create UI
	dx, _ := ui.TerminalDimensions()
	header := widgets.NewParagraph()
	header.Text = "Press <C-c> to quit, Press <F1> or <F2> to switch tabs"
	header.SetRect(0, 0, dx, 1)

	tabpane := widgets.NewTabPane("status", "faces", "log")
	tabpane.SetRect(0, 1, dx, 4)

	status := forms.NewStatusForm()
	faces := forms.NewFacesForm()
	notImplemented := forms.NewNotImplementedForm()

	var current forms.Form
	var refresh <-chan uint

	switchTab := func() {
		switch tabpane.ActiveTabIndex {
		case 0:
			current = status
		case 1:
			current = faces
		case 2:
			current = notImplemented
		}
		refresh = current.RefreshSignal()
	}

	switchTab()
	ui.Render(header, tabpane)
	current.Render()

	uiEvents := ui.PollEvents()
	running := true
	for running {
		select {
		case e := <-uiEvents:
			if e.Type == ui.KeyboardEvent {
				switch e.ID {
				case "<C-c>":
					running = false
				case "<F1>":
					tabpane.FocusLeft()
					switchTab()
				case "<F2>":
					tabpane.FocusRight()
					switchTab()
				}

				current.KeyboardEvent(e)
			}
		case <-refresh:
		}
		ui.Clear()
		ui.Render(header, tabpane)
		current.Render()
	}

	yanfd.Stop()
}
