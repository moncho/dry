package jsontree

import (
	"reflect"
	"strings"
	"testing"
)

var sampleJson = createLinesFromString(`{
    "foo": 0,
    "bar": {
        "baz": true
    }
}`)

func TestToggleLine(t *testing.T) {
	tree := New(sampleJson)

	tree.ToggleLine(2)
	actual := tree.Line(3)
	expected := createLinesFromString(`        "baz": true`)[0]

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Line: %v, want %v", actual, expected)
	}

	tree.ToggleLine(2)
	actual = tree.Line(2)
	expected = createLinesFromString(`    "bar": {…}`)[0]

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Line: %v, want %v", actual, expected)
	}

}

var sampleJsonWithEmptyObject = createLinesFromString(`{
    "foo": {},
    "bar": {
        "baz": 0
    }
}`)

func TestEmptyObjects(t *testing.T) {
	tree := New(sampleJsonWithEmptyObject)

	actual := tree.Line(1)
	expected := createLinesFromString(`    "foo": {},`)[0]

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Line: %v, want %v", actual, expected)
	}

	actual = tree.Line(2)
	expected = createLinesFromString(`    "bar": {…}`)[0]

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Line: %v, want %v", actual, expected)
	}
}

func createLinesFromString(s string) []Line {
	var lines []Line
	for _, ln := range strings.Split(s, "\n") {
		var resultLine Line
		for _, c := range ln {
			resultLine = append(resultLine, Char{Val: c})
		}
		lines = append(lines, resultLine)
	}

	return lines
}
