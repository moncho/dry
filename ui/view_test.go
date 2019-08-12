package ui

import (
	"fmt"
	"testing"
)

func TestCursorScrolling(t *testing.T) {
	view := view(10, 10)
	testCursor(t, view, 0, 0)
	//
	view.CursorToBottom()
	testCursor(t, view, 0, 10)

	view.CursorToBottom()
	testCursor(t, view, 0, 10)

	view.CursorToTop()
	testCursor(t, view, 0, 0)

	view.CursorDown()
	testCursor(t, view, 0, 1)

	view.CursorUp()
	testCursor(t, view, 0, 0)

	view.CursorUp()
	testCursor(t, view, 0, 0)
}

func TestViewBufferPosition(t *testing.T) {
	view := view(10, 10)
	numberOfLinesToWrite := 20

	for i := 0; i < numberOfLinesToWrite; i++ {
		fmt.Fprintf(view, "Line %d\n", i)
	}
	firstLine, _ := view.Line(0)
	if firstLine != "Line 0" {
		t.Errorf("Buffer content is not right, expected: %s got: %s",
			"Line 0",
			firstLine)
	}
	view.PageDown()
	testCursor(t, view, 0, 0)
	testViewPosition(t, view, 0, 9)
	view.PageDown()
	testCursor(t, view, 0, 0)
	testViewPosition(t, view, 0, 18)
	view.PageDown()
	testCursor(t, view, 0, 10)
	//The buffer
	testViewBufferSize(t, view, numberOfLinesToWrite+1)

}

func testViewBufferSize(t *testing.T, view *View, expected int) {
	if expected != len(view.lines) {
		t.Errorf("View buffer has not the expected size, expected: %d got: %d",
			expected,
			len(view.lines))
	}
}

func testCursor(t *testing.T, view *View, expectedX int, expectedY int) {
	t.Helper()
	x, y := view.Cursor()
	if x != expectedX || y != expectedY {
		t.Errorf("Cursor is not at the right position, expected: (%d, %d) got: (%d, %d)",
			expectedX,
			expectedY,
			x,
			y)
	}
}

func testViewPosition(t *testing.T, view *View, expectedX int, expectedY int) {
	t.Helper()

	x, y := view.Position()
	if x != expectedX || y != expectedY {
		t.Errorf("View buffer is not at the right position, expected: (%d, %d) got: (%d, %d)",
			expectedX,
			expectedY,
			x,
			y)
	}

}

func view(width, height int) *View {

	view := View{
		name:            "",
		x0:              0,
		y0:              0,
		x1:              width,
		y1:              height,
		width:           width,
		height:          height - 1,
		showCursor:      true,
		theme:           nil,
		newLineCallback: func() {},
	}

	view.renderer = NewRenderer(screenMock{}).WithWidth(view.width)

	//view.cursorY = height - 1 //Last line i
	return &view
}
