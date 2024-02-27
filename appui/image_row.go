package appui

import (
	"github.com/docker/docker/api/types"
	termui "github.com/gizak/termui"
	"github.com/moncho/dry/docker/formatter"
	drytermui "github.com/moncho/dry/ui/termui"
)

// ImageRow is a Grid row showing information about a Docker image
type ImageRow struct {
	image             types.ImageSummary
	Repository        *drytermui.ParColumn
	Tag               *drytermui.ParColumn
	ID                *drytermui.ParColumn
	CreatedSinceValue int64
	CreatedSince      *drytermui.ParColumn
	SizeValue         int64
	Size              *drytermui.ParColumn

	Row
}

// NewImageRow creates a new ImageRow widget
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
	row.ParColumns = []*drytermui.ParColumn{
		row.Repository,
		row.Tag,
		row.ID,
		row.CreatedSince,
		row.Size,
	}

	return row

}

// ColumnsForFilter returns the columns that are used to filter
func (row *ImageRow) ColumnsForFilter() []*drytermui.ParColumn {
	return []*drytermui.ParColumn{row.Repository, row.Tag, row.ID}
}
