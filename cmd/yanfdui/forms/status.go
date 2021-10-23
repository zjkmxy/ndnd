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
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/table"
)

type StatusForm struct {
	refresh chan uint
	ticker  *time.Ticker
	general *widgets.Table
	faces   *widgets.Table
}

func NewStatusForm() *StatusForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	general := widgets.NewTable()
	general.Title = "General"
	general.Rows = [][]string{
		{"version", ""},
		{"startTime", ""},
		{"currentTime", ""},
		{"nFibEntries", ""},
	}
	general.TextStyle = ui.NewStyle(ui.ColorWhite)
	general.SetRect(0, 5, 70, 14)

	faces := widgets.NewTable()
	faces.Title = "Faces"
	faces.Rows = [][]string{}
	faces.TextAlignment = ui.AlignCenter
	faces.TextStyle = ui.NewStyle(ui.ColorWhite)
	faces.SetRect(0, 15, 70, 30)

	go func() {
		for {
			<-ticker.C
			refresh <- 1
		}
	}()
	return &StatusForm{
		refresh: refresh,
		ticker:  ticker,
		general: general,
		faces:   faces,
	}
}

func (f *StatusForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *StatusForm) Render() {
	// Don't set NNameTreeEntries because we don't use a NameTree
	nFibEntries := uint64(len(table.FibStrategyTable.GetAllFIBEntries()))

	// General status table
	f.general.Rows[0][1] = core.Version
	f.general.Rows[1][1] = core.StartTimestamp.Local().String()
	f.general.Rows[2][1] = time.Now().Local().String()
	f.general.Rows[3][1] = fmt.Sprint(nFibEntries)

	// Face list table
	f.faces.Rows = [][]string{}
	for _, face := range face.FaceTable.GetAll() {
		f.faces.Rows = append(f.faces.Rows,
			[]string{fmt.Sprint(face.FaceID()), face.LocalURI().String()},
		)
	}

	// Sort by face ID
	sort.Slice(f.faces.Rows, func(i, j int) bool {
		return f.faces.Rows[i][0] < f.faces.Rows[j][0]
	})

	// Add header row
	f.faces.Rows = append([][]string{{"ID", "Local URI"}}, f.faces.Rows...)

	ui.Render(f.general)
	ui.Render(f.faces)
}
