package appui

import (
	"strings"

	"github.com/moncho/dry/ui/termui"
)

// FilterableRow is the interface for filterable columns
type FilterableRow interface {
	ColumnsForFilter() []*termui.ParColumn
}

// RowFilter function for filtering rows
type RowFilter func(FilterableRow) bool

// RowFilters holds the existing RowFilter
var RowFilters RowFilter

// ByPattern filters row by the given pattern
func (rf RowFilter) ByPattern(pattern string) RowFilter {
	return func(row FilterableRow) bool {
		columns := row.ColumnsForFilter()
		for _, column := range columns {
			if strings.Contains(column.Text, pattern) {
				return true
			}
		}
		return false
	}
}
