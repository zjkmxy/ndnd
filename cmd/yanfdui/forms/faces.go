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
	yindex  int
}

func NewFacesForm() *FacesForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	tab := widgets.NewTable()
	tab.Title = "Faces"
	tab.Rows = [][]string{}
	tab.TextAlignment = ui.AlignCenter
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)

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
		yindex:  0,
	}
}

func (f *FacesForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *FacesForm) Render() {
	dx, _ := ui.TerminalDimensions()
	f.tab.SetRect(0, 5, dx, 24)

	// Face list table
	f.tab.Rows = [][]string{}
	faces := face.FaceTable.GetAll()

	// Sort by face ID
	sort.Slice(faces, func(i, j int) bool {
		return faces[i].FaceID() < faces[j].FaceID()
	})

	if f.yindex >= len(faces) {
		f.yindex = len(faces) - 1
	}

	for i, face := range faces {
		if i >= f.yindex && i < f.yindex+8 {
			f.tab.Rows = append(f.tab.Rows,
				[]string{fmt.Sprint(face.FaceID()), face.LocalURI().String(), face.RemoteURI().String()},
			)
		}
	}

	// Add header row
	f.tab.Rows = append([][]string{{"ID", "Local", "Remote"}}, f.tab.Rows...)

	ui.Render(f.tab)
}

func (f *FacesForm) KeyboardEvent(e ui.Event) {
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
