package jsontree

import (
	"unicode"

	"github.com/nsf/termbox-go"
)

type JsonTree struct {
	lines         []Line
	expandedLines map[int]struct{}
	lineMap       map[int]int
	segments      map[int]int
}

type Char struct {
	Val   rune
	Color termbox.Attribute
}

type Line []Char

func New(lines []Line) *JsonTree {
	model := &JsonTree{
		lines:         lines,
		expandedLines: map[int]struct{}{},
		lineMap:       make(map[int]int),
		segments:      parseSegments(lines),
	}
	model.recalculateLineMap()
	model.ToggleLine(0)

	return model
}

func (t *JsonTree) ToggleLine(virtualLn int) {
	actualLn := t.lineMap[virtualLn]
	if !t.isBeginningOfSegment(actualLn) {
		return
	}

	if t.isExpanded(actualLn) {
		delete(t.expandedLines, actualLn)
	} else {
		t.expandedLines[actualLn] = struct{}{}
	}
	t.recalculateLineMap()
}

func (t *JsonTree) Line(virtualLn int) Line {
	actualLn, ok := t.lineMap[virtualLn]
	if ok {
		ln := t.lines[actualLn]
		if t.isBeginningOfSegment(actualLn) && !t.isExpanded(actualLn) {
			ln = t.lineWithDots(actualLn)
		}
		return ln
	}

	return nil
}

func (t *JsonTree) lineWithDots(actualLn int) Line {
	ln := t.lines[actualLn]

	lastChar := ln[len(ln)-1]
	ln = append(ln, Char{'…', lastChar.Color})

	matchingBrace := t.lines[t.segments[actualLn]]
	for _, c := range matchingBrace {
		if !unicode.IsSpace(c.Val) {
			ln = append(ln, c)
		}
	}

	return ln
}

func (t *JsonTree) isExpanded(actualLn int) bool {
	_, isExpanded := t.expandedLines[actualLn]
	return isExpanded
}

func (t *JsonTree) isBeginningOfSegment(actualLn int) bool {
	_, ok := t.segments[actualLn]
	return ok
}

func (t *JsonTree) recalculateLineMap() {
	t.lineMap = make(map[int]int)
	skipTill := 0
	virtualLn := 0
	for actualLn := range t.lines {
		if actualLn < skipTill {
			continue
		}

		if t.isBeginningOfSegment(actualLn) && !t.isExpanded(actualLn) {
			skipTill = t.segments[actualLn] + 1
		}

		t.lineMap[virtualLn] = actualLn
		virtualLn++
	}
}

func parseSegments(lines []Line) map[int]int {
	resultSegments := make(map[int]int)
	bracketBalances := make(map[int]int)
	var bal int
	for num, line := range lines {
		for _, c := range line {
			switch c.Val {
			case '{', '[':
				bal++
				bracketBalances[bal] = num
			case '}', ']':
				if startLn := bracketBalances[bal]; startLn != num {
					resultSegments[startLn] = num
				}
				bal--
			}
		}
	}

	return resultSegments
}
