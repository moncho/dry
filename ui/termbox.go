package ui

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

//fill fills the screen with the given cell starting at x,y until w,h.
func fill(x, y, w, h int, cell termbox.Cell) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			termbox.SetCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

//renderString renders the given string starting at x, y in the screen
func renderString(x, y int, word string, foreground, background termbox.Attribute) {
	for _, char := range word {
		termbox.SetCell(x, y, char, foreground, background)
		x += runewidth.RuneWidth(char)
	}
}

func runeAdvanceLen(r rune, pos int) int {
	if r == '\t' {
		return tabstopLength - pos%tabstopLength
	}
	return 1
}

func vOffsetToCOffset(text []byte, boffset int) (voffset, coffset int) {
	text = text[:boffset]
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		text = text[size:]
		coffset++
		voffset += runeAdvanceLen(r, voffset)
	}
	return
}

func byteSliceGrow(s []byte, desiredCap int) []byte {
	if cap(s) < desiredCap {
		ns := make([]byte, len(s), desiredCap)
		copy(ns, s)
		return ns
	}
	return s
}

func byteSliceRemove(text []byte, from, to int) []byte {
	size := to - from
	copy(text[from:], text[to:])
	text = text[:len(text)-size]
	return text
}

func byteSliceInsert(text []byte, offset int, what []byte) []byte {
	n := len(text) + len(what)
	text = byteSliceGrow(text, n)
	text = text[:n]
	copy(text[offset+len(what):], text[offset:])
	copy(text[offset:], what)
	return text
}
