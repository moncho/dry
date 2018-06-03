package colorwriter

import (
	"github.com/maxzender/jv/jsonfmt"
	"github.com/maxzender/jv/jsontree"
	"github.com/nsf/termbox-go"
)

type colorWriter struct {
	Lines    []jsontree.Line
	colorMap map[jsonfmt.TokenType]termbox.Attribute
	line     int
	bgColor  termbox.Attribute
}

func New(colorMap map[jsonfmt.TokenType]termbox.Attribute, bgColor termbox.Attribute) *colorWriter {
	writer := &colorWriter{
		colorMap: colorMap,
		bgColor:  bgColor,
	}

	writer.Lines = append(writer.Lines, jsontree.Line{})

	return writer
}

func (w *colorWriter) Write(s string, t jsonfmt.TokenType) {
	for _, c := range s {
		w.Lines[w.line] = append(w.Lines[w.line], jsontree.Char{c, w.colorMap[t]})
	}
}

func (w *colorWriter) Newline() {
	w.Lines = append(w.Lines, jsontree.Line{})
	w.line++
}
