package termui

import "testing"

func TestGaugeColumn(t *testing.T) {
	c := NewGaugeColumn()

	if c == nil {
		t.Error("GaugeColumn is nil")
	}

	if c.Border {
		t.Error("GaugeColumn has a border")
	}
	if c.GetHeight() != 1 {
		t.Error("GaugeColumn has not the expected height")
	}
}
