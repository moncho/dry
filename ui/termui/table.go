package termui

// DefaultColumnSpacing default the spacing (in chars) between columns of a table
const DefaultColumnSpacing = 1

// Table defines common behaviour for table widgets
type Table interface {
	ColumnWidths() []int
}
