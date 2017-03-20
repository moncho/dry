package termui

import (
	ui "github.com/gizak/termui"
)

//Grid is a custom termui.Grid which expects rows as GridBufferer(s).
type Grid struct {
	ui.GridBufferer
	rows   []ui.GridBufferer
	X, Y   int
	Width  int
	Offset int
}

//NewGrid creates a new Grid
func NewGrid(x, y, width int) *Grid {
	return &Grid{
		X:     x,
		Y:     y,
		Width: width,
	}
}

//Align aligns rows
func (g *Grid) Align() {
	y := g.Y
	if g.Offset >= len(g.rows) {
		g.Offset = 0
	}
	for _, r := range g.pageRows() {
		r.SetY(y)
		r.SetX(g.X)
		y += r.GetHeight()
		r.SetWidth(g.Width)
	}
}

//Clear this Grid content
func (g *Grid) Clear() { g.rows = []ui.GridBufferer{} }

//GetHeight return this Grid height
func (g *Grid) GetHeight() int { return len(g.rows) }

//SetX sets the X pos of this Grid
func (g *Grid) SetX(x int) { g.X = x }

//SetY sets the Y pos of this Grid
func (g *Grid) SetY(y int) { g.Y = y }

//SetWidth sets the width of this Grid
func (g *Grid) SetWidth(w int) { g.Width = w }

//MaxRows returns the max number of visible rows of this Grid
func (g *Grid) MaxRows() int { return ui.TermHeight() - g.Y - 1 }

func (g *Grid) pageRows() (rows []ui.GridBufferer) {
	rows = append(rows, g.rows[g.Offset:]...)
	return rows
}

//Buffer returns the content of this Grid as a Buffer
func (g *Grid) Buffer() ui.Buffer {
	buf := ui.NewBuffer()
	for _, r := range g.pageRows() {
		buf.Merge(r.Buffer())
	}
	return buf
}

//AddRows adds the given GridBufferer(s) as rows of this Grid
func (g *Grid) AddRows(rows ...ui.GridBufferer) {
	for _, r := range rows {
		g.rows = append(g.rows, r)
	}
}
