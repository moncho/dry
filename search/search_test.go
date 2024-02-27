package search

import (
	"reflect"
	"testing"
)

const (
	searchPattern = "line"

	lines = `line 1
		lien 2
		line3
	line 4
	line 5
	Nope
	Still nope
	Really, nope`
)

// TestSearch tests basic search
func TestSearch(t *testing.T) {
	expected := Result{
		Hits:    5,
		Lines:   []int{2, 3, 4, 8, 9},
		Pattern: searchPattern,
	}
	rs, _ := NewSearch(testText(), searchPattern)

	if expected.Hits != rs.Hits ||
		!reflect.DeepEqual(expected.Lines, rs.Lines) ||
		expected.Pattern != rs.Pattern {
		t.Errorf("Expected search result: %s, got: %s", expected.String(), rs.String())
	}
}

// TestResultIteration tests iterating the search results
func TestResultIteration(t *testing.T) {
	rs, _ := NewSearch(testText(), searchPattern)
	line, _ := rs.NextLine()
	if line != 2 {
		t.Errorf("Expected line: %d, got: %d", 2, line)
	}

	line, _ = rs.PreviousLine()

	if line != 2 {
		t.Errorf("Expected line: %d, got: %d", 2, line)
	}

	line, _ = rs.NextLine()

	if line != 3 {
		t.Errorf("Expected line: %d, got: %d", 3, line)
	}

	line, _ = rs.NextLine()
	if line != 4 {
		t.Errorf("Expected line: %d, got: %d", 4, line)

	}

	line, _ = rs.PreviousLine()
	if line != 3 {
		t.Errorf("Expected line: %d, got: %d", 3, line)

	}

	line, _ = rs.NextLine()
	if line != 4 {
		t.Errorf("Expected line: %d, got: %d", 4, line)

	}

	line, _ = rs.NextLine()
	if line != 8 {
		t.Errorf("Expected line: %d, got: %d", 8, line)

	}

	rs.InitialLine(5)
	line, _ = rs.NextLine()
	if line != 8 {
		t.Errorf("Expected line: %d, got: %d", 8, line)

	}

	rs.InitialLine(9)
	line, _ = rs.NextLine()
	if line != 9 {
		t.Errorf("Expected line: %d, got: %d", 9, line)

	}

	line, _ = rs.NextLine()
	if line != 9 {
		t.Errorf("Expected line: %d, got: %d", 9, line)

	}

}

func testText() [][]rune {
	return [][]rune{[]rune("ine 1 nope"),
		[]rune("lien 2 nope"),
		[]rune("line3 yes"),
		[]rune("line 4"),
		[]rune("line 5"),
		[]rune("Nope 6"),
		[]rune("Still nope 7"),
		[]rune("Really, nope 8"),
		[]rune("yes line 9"),
		[]rune("line 10"),
		[]rune("lin 11")}
}
