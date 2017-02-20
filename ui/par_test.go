package ui

import (
	"testing"

	"github.com/gizak/termui"
)

func TestNewPar(t *testing.T) {

	theme := &ColorTheme{Bg: 1, Fg: 2}
	p := NewPar("test", theme)

	if p == nil {
		t.Error("Par is nil")
	}
	if p.Text != "test" {
		t.Errorf("Par does not have expected text, got: %s ", p.Text)
	}

	if p.Bg != termui.Attribute(theme.Bg) {
		t.Errorf("Par BG attribute does not have expected value. Got: %v, expected: %v", p.Bg, theme.Bg)
	}
	if p.BorderFg != termui.Attribute(theme.Fg) {
		t.Errorf("Par BorderFg attribute does not have expected value. Got: %v, expected: %v", p.BorderFg, theme.Fg)
	}
	if p.BorderBg != termui.Attribute(theme.Bg) {
		t.Errorf("Par BorderBg attribute does not have expected value. Got: %v, expected: %v", p.BorderBg, theme.Bg)
	}
	if p.TextFgColor != termui.Attribute(theme.Fg) {
		t.Errorf("Par TextFgColor attribute does not have expected value. Got: %v, expected: %v", p.TextFgColor, theme.Fg)
	}
	if p.TextBgColor != termui.Attribute(theme.Bg) {
		t.Errorf("Par TextBgColor attribute does not have expected value. Got: %v, expected: %v", p.TextBgColor, theme.Bg)
	}
	if p.BorderLabelFg != termui.Attribute(theme.Fg) {
		t.Errorf("Par BorderLabelFg attribute does not have expected value. Got: %v, expected: %v", p.BorderLabelFg, theme.Fg)
	}
	if p.BorderLabelBg != termui.Attribute(theme.Bg) {
		t.Errorf("Par BorderLabelBg attribute does not have expected value. Got: %v, expected: %v", p.BorderLabelBg, theme.Bg)
	}
}
