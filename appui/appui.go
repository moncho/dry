package appui

import "fmt"

type invalidRow struct {
	selected int
	max      int
}

// Error returns the error message
func (i invalidRow) Error() string {
	return fmt.Sprintf("invalid row index (selected: %d) (max: %d)", i.selected, i.max)
}
