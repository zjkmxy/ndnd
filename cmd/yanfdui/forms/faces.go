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
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face"
)

type FacesForm struct {
	refresh chan uint
	ticker  *time.Ticker
	tab     *widgets.Table
	yindex  int
	input   *widgets.Paragraph
	state   uint
}

const (
	FacesStateNone uint = iota
	FacesStateAdd
	FacesStateRemove
)

func NewFacesForm() *FacesForm {
	ticker := time.NewTicker(1 * time.Second)
	refresh := make(chan uint)

	tab := widgets.NewTable()
	tab.Title = "Faces"
	tab.Rows = [][]string{}
	tab.TextAlignment = ui.AlignCenter
	tab.TextStyle = ui.NewStyle(ui.ColorWhite)

	input := widgets.NewParagraph()
	input.Title = "Input"
	input.Text = ""
	input.SetRect(0, 1, 70, 4)

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
		input:   input,
		state:   FacesStateNone,
	}
}

func (f *FacesForm) GetName() string {
	return "Faces"
}

func (f *FacesForm) Reset() {
	f.state = FacesStateNone
	f.yindex = 0
}

func (f *FacesForm) IsPopup() bool {
	return f.state != FacesStateNone
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
	if f.IsPopup() {
		ui.Render(f.input)
	}
}

func (f *FacesForm) KeyboardEvent(e ui.Event) {
	switch f.state {
	case FacesStateNone:
		switch e.ID {
		case "<Down>":
			f.yindex += 1
			f.Render()
		case "<Up>":
			if f.yindex > 0 {
				f.yindex -= 1
				f.Render()
			}
		case "d":
			f.state = FacesStateRemove
			f.input.Title = "Face ID to delete"
			f.input.Text = ""
		case "a":
			f.state = FacesStateAdd
			f.input.Text = ""
		}
	case FacesStateAdd, FacesStateRemove:
		switch e.ID {
		case "<Enter>":
			if f.input.Text != "" {
				switch f.state {
				case FacesStateRemove:
					faceId, err := strconv.ParseUint(f.input.Text, 10, 64)
					if err != nil {
						core.LogError(f, "Failed to parse input {", faceId, "} to face ID: ", err.Error())
					} else {
						if face.FaceTable.Get(faceId) != nil {
							// Does not really destroy the face, just removes it from the table
							face.FaceTable.Remove(faceId)
							core.LogInfo(f, "Destroyed face with FaceID=", faceId)
						} else {
							core.LogInfo(f, "Ignoring attempt to delete non-existent face with FaceID=", faceId)
						}
					}
				}
			}
			f.state = FacesStateNone
		case "<Escape>":
			f.state = FacesStateNone
		case "<Backspace>":
			if f.input.Text != "" {
				f.input.Text = f.input.Text[:len(f.input.Text)-1]
			}
		default:
			if e.ID != "" && e.ID[0] != '<' {
				f.input.Text += e.ID
			}
		}
	}
}
