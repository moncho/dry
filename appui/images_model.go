package appui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/docker/docker/api/types/image"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/docker/formatter"
)

// imageRow wraps a Docker image as a TableRow.
type imageRow struct {
	image   image.Summary
	columns []string
}

func newImageRow(img image.Summary) imageRow {
	f := formatter.NewImageFormatter(img, true)
	return imageRow{
		image: img,
		columns: []string{
			f.Repository(), f.Tag(), f.ID(), f.CreatedSince(), f.Size(),
		},
	}
}

func (r imageRow) Columns() []string { return r.columns }
func (r imageRow) ID() string        { return r.image.ID }

// ImagesLoadedMsg carries the loaded images.
type ImagesLoadedMsg struct {
	Images []image.Summary
}

// ImagesModel is the images list view sub-model.
type ImagesModel struct {
	table  TableModel
	filter FilterInputModel
	daemon docker.ContainerDaemon
}

// NewImagesModel creates an images list model.
func NewImagesModel() ImagesModel {
	columns := []Column{
		{Title: "REPOSITORY"},
		{Title: "TAG", Width: 20, Fixed: true},
		{Title: "ID", Width: IDColumnWidth, Fixed: true},
		{Title: "CREATED", Width: 16, Fixed: true},
		{Title: "SIZE", Width: 10, Fixed: true},
	}
	return ImagesModel{
		table:  NewTableModel(columns),
		filter: NewFilterInputModel(),
	}
}

// SetDaemon sets the Docker daemon reference.
func (m *ImagesModel) SetDaemon(d docker.ContainerDaemon) {
	m.daemon = d
}

// SetSize updates the table dimensions.
func (m *ImagesModel) SetSize(w, h int) {
	filterH := 0
	if m.filter.Active() {
		filterH = 1
	}
	m.table.SetSize(w, h-1-filterH)
	m.filter.SetWidth(w)
}

// SetImages replaces the image list.
func (m *ImagesModel) SetImages(images []image.Summary) {
	rows := make([]TableRow, len(images))
	for i, img := range images {
		rows[i] = newImageRow(img)
	}
	m.table.SetRows(rows)
}

// SelectedImage returns the image under the cursor, or nil.
func (m ImagesModel) SelectedImage() *image.Summary {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	if ir, ok := row.(imageRow); ok {
		return &ir.image
	}
	return nil
}

// Update handles image-list-specific key events.
func (m ImagesModel) Update(msg tea.Msg) (ImagesModel, tea.Cmd) {
	if m.filter.Active() {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.table.SetFilter(m.filter.Value())
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1":
			m.table.NextSort()
			return m, nil
		case "f5":
			return m, nil // parent handles reload
		case "%":
			cmd := m.filter.Activate()
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the images list.
func (m ImagesModel) View() string {
	header := m.widgetHeader()
	tableView := m.table.View()
	result := header + "\n" + tableView
	if filterView := m.filter.View(); filterView != "" {
		result += "\n" + filterView
	}
	return result
}

func (m ImagesModel) widgetHeader() string {
	return RenderWidgetHeader(WidgetHeaderOpts{
		Icon:     "📦",
		Title:    "Images",
		Total:    m.table.TotalRowCount(),
		Filtered: m.table.RowCount(),
		Filter:   m.table.FilterText(),
		Width:    m.table.Width(),
		Accent:   DryTheme.Secondary,
	})
}
