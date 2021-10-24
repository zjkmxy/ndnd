/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package forms

import (
	"fmt"
	"sort"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/named-data/YaNFD/table"
)

type FibForm struct {
	refresh chan uint
	ticker  *time.Ticker
	tab     *widgets.Table
	yindex  int
}

func NewFibForm() *FibForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	tab := widgets.NewTable()
	tab.Title = "FIB Entries"
	tab.Rows = [][]string{}
	tab.TextAlignment = ui.AlignCenter
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)

	go func() {
		for {
			<-ticker.C
			refresh <- 1
		}
	}()

	return &FibForm{
		refresh: refresh,
		ticker:  ticker,
		tab:     tab,
		yindex:  0,
	}
}

func (f *FibForm) GetName() string {
	return "FIB"
}

func (f *FibForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *FibForm) Render() {
	dx, _ := ui.TerminalDimensions()
	f.tab.SetRect(0, 5, dx, 24)

	// FIB list table
	f.tab.Rows = [][]string{}
	entries := table.FibStrategyTable.GetAllFIBEntries()

	// Sort by face ID
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name.Compare(entries[j].Name) < 0
	})

	if f.yindex >= len(entries) {
		f.yindex = len(entries) - 1
	}

	for i, entry := range entries {
		if i >= f.yindex && i < f.yindex+8 {
			nexthops := ""
			for _, nh := range entry.GetNexthops() {
				nexthops += fmt.Sprint(nh.Nexthop, " ")
			}
			f.tab.Rows = append(f.tab.Rows,
				[]string{entry.Name.String(), nexthops},
			)
		}
	}

	// Add header row
	f.tab.Rows = append([][]string{{"Name", "Next Hop IDs"}}, f.tab.Rows...)

	ui.Render(f.tab)
}

func (f *FibForm) KeyboardEvent(e ui.Event) {
	switch e.ID {
	case "<Down>":
		f.yindex += 1
		f.Render()
	case "<Up>":
		if f.yindex > 0 {
			f.yindex -= 1
			f.Render()
		}
	}
}
