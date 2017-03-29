package ui

import "testing"

func TestCursor(t *testing.T) {
	c := Cursor{}

	if c.Position() != 0 {
		t.Error("New cursor does not start at position 0")
	}
	c.ScrollCursorDown()
	if c.Position() != 1 || !c.MovingDown() {
		t.Errorf("Invalid cursor state after moving down. %v", c)
	}

	c.ScrollCursorUp()
	if c.Position() != 0 || c.MovingDown() {
		t.Errorf("Invalid cursor state after moving up. %v", c)
	}

	c.ScrollCursorUp()
	if c.Position() != 0 {
		t.Errorf("Invalid cursor state after moving up too many times. %v", c)
	}

	c.ScrollCursorDown()
	c.ScrollCursorDown()
	c.ScrollCursorDown()

	if c.Position() != 3 || !c.MovingDown() {
		t.Errorf("Invalid cursor state after moving down 3 times. %v", c)
	}

	c.ScrollTo(5)

	if c.Position() != 5 || !c.MovingDown() {
		t.Errorf("Invalid cursor state after scrolling to position 5. %v", c)
	}

	c.ScrollTo(3)

	if c.Position() != 3 || c.MovingDown() {
		t.Errorf("Invalid cursor state after scrolling back to position 3 from pos 5. %v", c)
	}
}
