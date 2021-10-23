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
	"github.com/named-data/YaNFD/table"
)

type StatusForm struct {
	refresh chan uint
	timer   *time.Timer
	table   *widgets.Table
}

func NewStatusForm() *StatusForm {
	timer := time.NewTimer(5 * time.Second)
	refresh := make(chan uint)
	tab := widgets.NewTable()

	tab.Rows = [][]string{
		{"version", ""},
		{"startTime", ""},
		{"currentTime", ""},
		{"upTime", ""},
	}
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)
	tab.SetRect(0, 5, 70, 20)

	go func() {
		<-timer.C
		refresh <- 1
	}()
	return &StatusForm{
		refresh: refresh,
		timer:   timer,
		table:   tab,
	}
}

func (f *StatusForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *StatusForm) Render() {
	// Don't set NNameTreeEntries because we don't use a NameTree
	nFibEntries := uint64(len(table.FibStrategyTable.GetAllFIBEntries()))

	f.table.Rows[0][1] = core.Version
	f.table.Rows[1][1] = core.StartTimestamp.Local().String()
	f.table.Rows[2][1] = time.Now().Local().String()
	f.table.Rows[3][1] = fmt.Sprint(nFibEntries)

	ui.Render(f.table)
}
