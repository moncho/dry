package ui

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell"
)

type screenMock struct {
}

func (s screenMock) Dimensions() *Dimensions {
	return nil
}
func (s screenMock) Render(x int, y int, r rune, style tcell.Style) {

}
func (s screenMock) Style() tcell.Style {
	return tcell.StyleDefault
}

// TestLessScrolling tests cursor position in less when scrolling, the cursor has to stay
// all the time at the bottom.
func TestLessScrolling(t *testing.T) {
	less := newLess(10, 10)
	testLessCursor(t, less, 0, 9)
	//
	less.ScrollDown()
	testLessCursor(t, less, 0, 9)

	less.ScrollToBottom()
	testLessCursor(t, less, 0, 9)

	less.ScrollToTop()
	testLessCursor(t, less, 0, 9)

	less.ScrollDown()
	testLessCursor(t, less, 0, 9)

	less.ScrollUp()
	testLessCursor(t, less, 0, 9)

	less.ScrollUp()
	testLessCursor(t, less, 0, 9)
}

// TestLessBufferPosition tests less internal buffer positioning
// when scrolling.
// In a 10x10 screen with 20 lines added to the buffer, the following rules apply:
//   - The last row is reserved for the cursor, hence there are 9 rows
//     available to render the buffer content (rows 0-8).
//   - The cursor stays at the bottom ( row 9, counting from 0).
//
// It is expected that, the internal buffer marker, at most goes down until
//
//line 11 (counting from 0) = 20 (buffer size) - 8 (last row position counting from 0)
func TestLessBufferPosition(t *testing.T) {
	less := newLess(10, 10)

	for i := 0; i < 20; i++ {
		fmt.Fprintf(less, "Line %d\n", i)
	}
	firstLine, _ := less.Line(0)
	if firstLine != "Line 0" {
		t.Errorf("Buffer content is not right, expected: %s got: %s",
			"Line 0",
			firstLine)
	}
	lastLine, _ := less.Line(19)
	if lastLine != "Line 19" {
		t.Errorf("Buffer content is not right, expected: %s got: %s",
			"Line 19",
			lastLine)
	}

	testLessCursor(t, less, 0, 9)
	testLessBufferPosition(t, less, 0, 0)
	testEndOfBufferReached(t, less, false)

	less.ScrollPageDown()
	testLessCursor(t, less, 0, 9)
	testLessBufferPosition(t, less, 0, 8)
	testEndOfBufferReached(t, less, false)

	less.ScrollPageDown()
	testLessCursor(t, less, 0, 9)
	testLessBufferPosition(t, less, 0, 11)
	testEndOfBufferReached(t, less, true)

	less.ScrollPageDown()
	testLessCursor(t, less, 0, 9)
	testLessBufferPosition(t, less, 0, 11)
	testEndOfBufferReached(t, less, true)

}

func TestLessSearch(t *testing.T) {
	less := newLess(10, 10)

	for i := 0; i < 20; i++ {
		fmt.Fprintf(less, "Line %d\n", i)
	}

	err := less.search("Line")

	if err != nil {
		t.Errorf(err.Error())
	}
	result := less.searchResult

	if result.Hits != 20 {
		t.Errorf("Expected to find %d occurrences, got: %d",
			20,
			result.Hits)

	}
}

func testLessCursor(t *testing.T, less *Less, expectedX int, expectedY int) {
	t.Helper()
	x, y := less.Cursor()
	if x != expectedX || y != expectedY {
		t.Errorf("Cursor osition, expected: (%d, %d) got: (%d, %d)",
			expectedX,
			expectedY,
			x,
			y)
	}
}

func testLessBufferPosition(t *testing.T, less *Less, expectedX int, expectedY int) {
	t.Helper()

	x, y := less.Position()
	if x != expectedX || y != expectedY {
		t.Errorf("Less buffer position, expected: (%d, %d) got: (%d, %d)",
			expectedX,
			expectedY,
			x,
			y)
	}
}

func testEndOfBufferReached(t *testing.T, less *Less, expected bool) {
	t.Helper()
	if less.atTheEndOfBuffer() != expected {
		t.Errorf("Less end-of-buffer status is: %t, expected %t.", less.atTheEndOfBuffer(), expected)
	}
}

func newLess(width int, height int) *Less {
	view := view(width, height)
	view.cursorY = height - 1 //Last line is at height -1

	return &Less{
		View:    view,
		refresh: make(chan struct{}, 10),
	}
}
