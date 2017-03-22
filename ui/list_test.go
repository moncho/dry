package ui

import (
	"testing"

	"github.com/gizak/termui"
)

func TestNewList(t *testing.T) {
	theme := &ColorTheme{Bg: 1, Fg: 2}
	l := NewList(theme)

	if l == nil {
		t.Error("List is nil")
	}

	if l.Bg != termui.Attribute(theme.Bg) {
		t.Errorf("List Bg does not have expected value. Expected:%v, got: %v ", theme.Bg, l.Bg)
	}
	if l.ItemBgColor != termui.Attribute(theme.Bg) {
		t.Errorf("List ItemBgColor does not have expected value. Expected:%v, got: %v ", theme.Bg, l.ItemBgColor)
	}
	if l.ItemFgColor != termui.Attribute(theme.Fg) {
		t.Errorf("List Bg does not have expected value. Expected:%v, got: %v ", theme.Bg, l.ItemFgColor)
	}

}
