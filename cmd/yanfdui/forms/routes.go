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
	"github.com/named-data/YaNFD/table"
)

type RoutesForm struct {
	refresh chan uint
	ticker  *time.Ticker
	tab     *widgets.Table
	yindex  int
}

func NewRoutesForm() *RoutesForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	tab := widgets.NewTable()
	tab.Title = "Routes"
	tab.Rows = [][]string{}
	tab.TextAlignment = ui.AlignCenter
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)

	go func() {
		for {
			<-ticker.C
			refresh <- 1
		}
	}()

	return &RoutesForm{
		refresh: refresh,
		ticker:  ticker,
		tab:     tab,
		yindex:  0,
	}
}

func (f *RoutesForm) GetName() string {
	return "Routes"
}

func (f *RoutesForm) Reset() {
	f.yindex = 0
}

func (f *RoutesForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *RoutesForm) Render() {
	dx, _ := ui.TerminalDimensions()
	f.tab.SetRect(0, 5, dx, 24)

	// Face list table
	f.tab.Rows = [][]string{}
	rib := table.Rib.GetAllEntries()

	i := 0
	for _, entry := range rib {
		for _, route := range entry.GetRoutes() {
			if i < f.yindex {
				i++
				continue
			}

			if i >= f.yindex+8 {
				i = -1
				break
			}

			f.tab.Rows = append(f.tab.Rows,
				[]string{
					entry.Name.String(),
					fmt.Sprint(route.FaceID),
					fmt.Sprint(route.Origin),
					fmt.Sprint(route.Cost),
					fmt.Sprint(route.Flags),
					fmt.Sprint(route.ExpirationPeriod)},
			)
			i++
		}

		if i < 0 {
			break
		}
	}

	// Add header row
	f.tab.Rows = append([][]string{{
		"Prefix", "Face ID", "Origin", "Cost", "Flags", "ExpiryPeriod",
	}}, f.tab.Rows...)

	ui.Render(f.tab)
}

func (f *RoutesForm) KeyboardEvent(e ui.Event) {
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
