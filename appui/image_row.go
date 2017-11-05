package appui

import (
	"image"

	"github.com/docker/docker/api/types"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

//ImageRow is a Grid row showing information about a Docker image
type ImageRow struct {
	image             types.ImageSummary
	Repository        *drytermui.ParColumn
	Tag               *drytermui.ParColumn
	ID                *drytermui.ParColumn
	CreatedSinceValue int64
	CreatedSince      *drytermui.ParColumn
	SizeValue         int64
	Size              *drytermui.ParColumn

	drytermui.Row
}

//NewImageRow creates a new ImageRow widget
func NewImageRow(image types.ImageSummary, table drytermui.Table) *ImageRow {
	iformatter := formatter.NewImageFormatter(image, true)

	row := &ImageRow{
		image:             image,
		Repository:        drytermui.NewThemedParColumn(DryTheme, iformatter.Repository()),
		Tag:               drytermui.NewThemedParColumn(DryTheme, iformatter.Tag()),
		ID:                drytermui.NewThemedParColumn(DryTheme, iformatter.ID()),
		CreatedSince:      drytermui.NewThemedParColumn(DryTheme, iformatter.CreatedSince()),
		CreatedSinceValue: image.Created,
		Size:              drytermui.NewThemedParColumn(DryTheme, iformatter.Size()),
		SizeValue:         image.VirtualSize,
	}
	row.Height = 1
	row.Table = table
	//Columns are rendered following the slice order
	row.Columns = []termui.GridBufferer{
		row.Repository,
		row.Tag,
		row.ID,
		row.CreatedSince,
		row.Size,
	}

	return row

}

//Buffer returns this Row data as a termui.Buffer
func (row *ImageRow) Buffer() termui.Buffer {
	buf := termui.NewBuffer()
	//This set the background of the whole row
	buf.Area.Min = image.Point{row.X, row.Y}
	buf.Area.Max = image.Point{row.X + row.Width, row.Y + row.Height}
	buf.Fill(' ', row.ID.TextFgColor, row.ID.TextBgColor)

	for _, col := range row.Columns {
		buf.Merge(col.Buffer())
	}
	return buf
}

//ColumnsForFilter returns the columns that are used to filter
func (row *ImageRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Repository, row.Tag, row.ID}
}

//Highlighted marks this rows as being highlighted
func (row *ImageRow) Highlighted() {
	row.changeTextColor(
		termui.Attribute(DryTheme.Fg),
		termui.Attribute(DryTheme.CursorLineBg))
}

//NotHighlighted marks this rows as being not highlighted
func (row *ImageRow) NotHighlighted() {

	row.changeTextColor(
		termui.Attribute(DryTheme.ListItem),
		termui.Attribute(DryTheme.Bg))
}

func (row *ImageRow) changeTextColor(fg, bg termui.Attribute) {

	row.ID.TextFgColor = fg
	row.ID.TextBgColor = bg
	row.Repository.TextFgColor = fg
	row.Repository.TextBgColor = bg
	row.Tag.TextFgColor = fg
	row.Tag.TextBgColor = bg
	row.CreatedSince.TextFgColor = fg
	row.CreatedSince.TextBgColor = bg
	row.Size.TextFgColor = fg
	row.Size.TextBgColor = bg
}
