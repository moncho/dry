package ui

import "testing"

func TestCursor(t *testing.T) {
	c := NewCursor()

	if c.Position() != 0 {
		t.Error("New cursor does not start at position 0")
	}

	c.Max(10)

	if c.max != 10 {
		t.Errorf("Cursor does not have expected max value: %d", c.max)
	}

	if c.String() != "[0, true, 10]" {
		t.Errorf("Unexpected cursor string representation: %s", c.String())
	}
}

func TestScrolling(t *testing.T) {
	c := Cursor{}
	c.ScrollCursorDown()
	if c.Position() != 1 || !c.MovingDown() {
		t.Errorf("Invalid cursor state after moving down. %s", c)
	}

	c.ScrollCursorUp()
	if c.Position() != 0 || c.MovingDown() {
		t.Errorf("Invalid cursor state after moving up. %s", c)
	}

	c.ScrollCursorUp()
	if c.Position() != 0 {
		t.Errorf("Invalid cursor state after moving up too many times. %s", c)
	}

	c.ScrollCursorDown()
	c.ScrollCursorDown()
	c.ScrollCursorDown()

	if c.Position() != 3 || !c.MovingDown() {
		t.Errorf("Invalid cursor state after moving down 3 times. %s", c)
	}

	c.ScrollTo(5)

	if c.Position() != 5 || !c.MovingDown() {
		t.Errorf("Invalid cursor state after scrolling to position 5. %s", c)
	}

	c.ScrollTo(3)

	if c.Position() != 3 || c.MovingDown() {
		t.Errorf("Invalid cursor state after scrolling back to position 3 from pos 5. %s", c)
	}
}
