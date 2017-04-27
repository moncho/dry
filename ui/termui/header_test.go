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

	header.AddFixedWidthColumn("Column4", 8)

	if header.ColumnCount() != 4 {
		t.Errorf("Header does not have 4 columns after adding a fixed width column, got: %d", header.ColumnCount())
	}

	header.ColumnSpacing = 1
	header.SetWidth(40)
	cw := header.CalcColumnWidth(3)
	if cw != (40-(3+8))/3 {
		t.Errorf("Calculated column width with 4 columns (one of them with witdh 8) is: %d", cw)

	}
	if header.Columns[0].Width != cw {
		t.Errorf("Non-fixed width columns do not have the expected width, got: %d, expected :%d", header.Columns[0].Width, cw)

	}
	if header.Columns[3].Width != 8 {
		t.Errorf("Fixed width columns do not have the expected width, got: %d, expected :%d", header.Columns[3].Width, 8)

	}
}
