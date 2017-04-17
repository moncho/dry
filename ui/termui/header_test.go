package termui

import (
	"testing"

	"github.com/moncho/dry/ui"
)

func TestHeader(t *testing.T) {
	header := NewHeader(&ui.ColorTheme{})

	if header.GetHeight() != 1 {
		t.Errorf("Header does not have the expected height. Got: %d, expected 1", header.GetHeight())
	}

	header.AddColumn("column1")
	header.AddColumn("column2")
	header.AddColumn("column3")

	if header.ColumnCount() != 3 {
		t.Errorf("Header does not have 3 columns, got: %d", header.ColumnCount())
	}
}
