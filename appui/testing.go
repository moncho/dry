package appui

import (
	"flag"
	"image"

	"github.com/moncho/dry/ui"
)

var update = flag.Bool("update", false, "update .golden files")

type testScreen struct {
	cursor         *ui.Cursor
	x0, y0, x1, y1 int
}

func (ts *testScreen) Cursor() *ui.Cursor {
	return ts.cursor
}

func (ts *testScreen) Bounds() image.Rectangle {
	return image.Rect(ts.x0, ts.y0, ts.x1, ts.y1)
}
