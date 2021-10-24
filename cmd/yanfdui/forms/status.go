/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package forms

import (
	"fmt"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/table"
)

type StatusForm struct {
	refresh chan uint
	ticker  *time.Ticker
	tab     *widgets.Table
}

func NewStatusForm() *StatusForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	tab := widgets.NewTable()
	tab.Rows = [][]string{
		{"Version", ""},
		{"Start Time", ""},
		{"Current Time", ""},
		{"Entries FIB PIT CS", ""},
		{"Counter IN", ""},
		{"Counter OUT", ""},
		{"INT Sat/Unsat", ""},
	}
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)

	go func() {
		for {
			<-ticker.C
			refresh <- 1
		}
	}()
	return &StatusForm{
		refresh: refresh,
		ticker:  ticker,
		tab:     tab,
	}
}

func (f *StatusForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *StatusForm) Render() {
	dx, _ := ui.TerminalDimensions()
	f.tab.SetRect(0, 5, dx, 20)

	// Don't set NNameTreeEntries because we don't use a NameTree
	nFibEntries := uint64(len(table.FibStrategyTable.GetAllFIBEntries()))

	f.tab.Rows[0][1] = core.Version
	f.tab.Rows[1][1] = core.StartTimestamp.Local().String()
	f.tab.Rows[2][1] = time.Now().Local().String()

	var nPitEntries, nCsEntries, nInInterests, nInData, nOutInterests uint64
	var nOutData, nSatisfiedInterests, nUnsatisfiedInterests uint64
	for threadID := 0; threadID < fw.NumFwThreads; threadID++ {
		thread := dispatch.GetFWThread(threadID)
		nPitEntries += uint64(thread.GetNumPitEntries())
		nCsEntries += uint64(thread.GetNumCsEntries())
		nInInterests += thread.(*fw.Thread).NInInterests
		nInData += thread.(*fw.Thread).NInData
		nOutInterests += thread.(*fw.Thread).NOutInterests
		nOutData += thread.(*fw.Thread).NOutData
		nSatisfiedInterests += thread.(*fw.Thread).NSatisfiedInterests
		nUnsatisfiedInterests += thread.(*fw.Thread).NUnsatisfiedInterests
	}

	f.tab.Rows[3][1] = fmt.Sprint("fib=", nFibEntries, " pit=", nPitEntries, " cs=", nCsEntries)
	f.tab.Rows[4][1] = fmt.Sprint(nInInterests, "i ", nInData, "d")
	f.tab.Rows[5][1] = fmt.Sprint(nOutInterests, "i ", nOutData, "d")
	f.tab.Rows[6][1] = fmt.Sprint(nSatisfiedInterests, " / ", nUnsatisfiedInterests)

	ui.Render(f.tab)
}

func (f *StatusForm) KeyboardEvent(ui.Event) {

}
