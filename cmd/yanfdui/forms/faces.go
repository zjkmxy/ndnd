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
	"github.com/named-data/YaNFD/face"
)

type FacesForm struct {
	refresh chan uint
	ticker  *time.Ticker
	tab     *widgets.Table
}

func NewFacesForm() *FacesForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	tab := widgets.NewTable()
	tab.Title = "Faces"
	tab.Rows = [][]string{}
	tab.TextAlignment = ui.AlignCenter
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)
	tab.SetRect(0, 5, 70, 30)

	go func() {
		for {
			<-ticker.C
			refresh <- 1
		}
	}()

	return &FacesForm{
		refresh: refresh,
		ticker:  ticker,
		tab:     tab,
	}
}

func (f *FacesForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *FacesForm) Render() {
	// Face list table
	f.tab.Rows = [][]string{}
	for _, face := range face.FaceTable.GetAll() {
		f.tab.Rows = append(f.tab.Rows,
			[]string{fmt.Sprint(face.FaceID()), face.LocalURI().String()},
		)
	}

	// Sort by face ID
	sort.Slice(f.tab.Rows, func(i, j int) bool {
		return f.tab.Rows[i][0] < f.tab.Rows[j][0]
	})

	// Add header row
	f.tab.Rows = append([][]string{{"ID", "Local URI"}}, f.tab.Rows...)

	ui.Render(f.tab)
}
