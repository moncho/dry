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
	cw := header.calcColumnWidth()
	if cw != (40-(3+8))/3 {
		t.Errorf("Calculated column width with 4 columns (one of them with witdh 8) is: %d", cw)

	}
	if len(header.ColumnWidths) != 4 {
		t.Errorf("Individual column widths does not have the expected length of 4, got %d", len(header.ColumnWidths))

	}

	if header.ColumnWidths[0] != cw {
		t.Errorf("Non-fixed width columns do not have the expected width, got: %d, expected :%d", header.ColumnWidths[0], cw)

	}

	if header.ColumnWidths[1] != cw {
		t.Errorf("Non-fixed width columns do not have the expected width, got: %d, expected :%d", header.ColumnWidths[1], cw)

	}

	if header.ColumnWidths[2] != cw {
		t.Errorf("Non-fixed width columns do not have the expected width, got: %d, expected :%d", header.ColumnWidths[2], cw)

	}
	if header.ColumnWidths[3] != 8 {
		t.Errorf("Fixed width columns do not have the expected width, got: %d, expected :%d", header.ColumnWidths[3], 8)

	}
}
