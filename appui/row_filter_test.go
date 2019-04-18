package appui

import (
	gizak "github.com/gizak/termui"
	"github.com/moncho/dry/ui/termui"
	"testing"
)

func TestRowFilter_ByPattern(t *testing.T) {
	type args struct {
		pattern string
	}
	tests := []struct {
		name       string
		filterable FilterableRow
		args       args
		want       bool
	}{
		{
			"blabla -> true",
			filterable{
				columns: []*termui.ParColumn{
					{
						Paragraph: gizak.Paragraph{Text: "blabla"},
					},
				},
			},
			args{
				"blabla",
			},
			true,
		},
		{
			"nope -> false",
			filterable{
				columns: []*termui.ParColumn{
					{
						Paragraph: gizak.Paragraph{Text: "yes"},
					},
				},
			},
			args{
				"nope",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := RowFilters.ByPattern(tt.args.pattern)
			got := filter(tt.filterable)
			if got != tt.want {
				t.Errorf("RowFilter.ByPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

type filterable struct {
	columns []*termui.ParColumn
}

func (f filterable) ColumnsForFilter() []*termui.ParColumn {
	return f.columns
}
