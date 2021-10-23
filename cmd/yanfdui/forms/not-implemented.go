/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package forms

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type NotImplementedForm struct {
	refresh chan uint
	msg     *widgets.Paragraph
}

func NewNotImplementedForm() *NotImplementedForm {
	msg := widgets.NewParagraph()
	msg.Text = "Not implemented yet"
	msg.SetRect(5, 5, 45, 15)
	return &NotImplementedForm{
		refresh: make(chan uint),
		msg:     msg,
	}
}

func (f *NotImplementedForm) RefreshSignal() <-chan uint {
	return f.refresh
}

func (f *NotImplementedForm) Render() {
	ui.Render(f.msg)
}

func (f *NotImplementedForm) KeyboardEvent(ui.Event) {

}
