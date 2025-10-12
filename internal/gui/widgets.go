package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type MinSized struct {
	widget.BaseWidget
	inner fyne.CanvasObject
	min   fyne.Size
}

func NewMinSized(inner fyne.CanvasObject, min fyne.Size) *MinSized {
	m := &MinSized{inner: inner, min: min}
	m.ExtendBaseWidget(m)
	return m
}

func (m *MinSized) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.inner)
}

func (m *MinSized) MinSize() fyne.Size {
	min := m.inner.MinSize()
	if min.Width < m.min.Width {
		min.Width = m.min.Width
	}
	if min.Height < m.min.Height {
		min.Height = m.min.Height
	}
	return min
}

