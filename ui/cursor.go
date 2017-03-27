package ui

import "sync"

//Cursor represents the cursor position on the screen
type Cursor struct {
	pos       int
	downwards bool
	max       int
	sync.RWMutex
}

//MovingDown returns true if the cursor is moving downwards after the last movement.
func (cursor *Cursor) MovingDown() bool {
	return cursor.downwards
}

//Position tells on which screen pos the cursor is
func (cursor *Cursor) Position() int {
	cursor.RLock()
	defer cursor.RUnlock()
	return cursor.pos
}

//Reset sets the cursor in the initial position
func (cursor *Cursor) Reset() {
	cursor.Lock()
	defer cursor.Unlock()
	cursor.pos = 0
	cursor.downwards = false

}

//ScrollCursorDown moves the cursor to the pos below the current one
func (cursor *Cursor) ScrollCursorDown() {
	cursor.Lock()
	defer cursor.Unlock()
	cursor.pos = cursor.pos + 1
	cursor.downwards = true
}

//ScrollCursorUp moves the cursor to the pos above the current one
func (cursor *Cursor) ScrollCursorUp() {
	cursor.Lock()
	defer cursor.Unlock()
	if cursor.pos > 0 {
		cursor.pos = cursor.pos - 1
	} else {
		cursor.pos = 0
	}
	cursor.downwards = false
}

//ScrollTo moves the cursor to the given pos
func (cursor *Cursor) ScrollTo(pos int) {
	cursor.Lock()
	defer cursor.Unlock()
	if pos > cursor.pos {
		cursor.downwards = true
	} else {
		cursor.downwards = false
	}
	cursor.pos = pos
}

//Bottom sets the cursor to the bottom
func (cursor *Cursor) Bottom() {
	cursor.Lock()
	defer cursor.Unlock()
	if cursor.max > 0 {
		cursor.pos = cursor.max
		cursor.downwards = true
	}
}

//Max sets the max position allowed to this cursor
func (cursor *Cursor) Max(max int) {
	cursor.Lock()
	defer cursor.Unlock()
	cursor.max = max
}
