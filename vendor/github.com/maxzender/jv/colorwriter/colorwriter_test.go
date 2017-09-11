package colorwriter

import (
	"reflect"
	"testing"

	"github.com/maxzender/jv/jsonfmt"
	"github.com/maxzender/jv/jsontree"
	"github.com/nsf/termbox-go"
)

var testColorMap = map[jsonfmt.TokenType]termbox.Attribute{
	jsonfmt.DelimiterType: termbox.ColorWhite,
	jsonfmt.BoolType:      termbox.ColorBlue,
	jsonfmt.StringType:    termbox.ColorRed,
	jsonfmt.NumberType:    termbox.ColorYellow,
	jsonfmt.NullType:      termbox.ColorCyan,
	jsonfmt.KeyType:       termbox.ColorBlack,
}

var defaultColor = termbox.ColorDefault

func TestWrite(t *testing.T) {
	writer := New(testColorMap, defaultColor)
	writer.Write(`{`, jsonfmt.DelimiterType)
	writer.Newline()
	writer.Write(`    `, jsonfmt.WhiteSpaceType)
	writer.Write(`"test"`, jsonfmt.KeyType)
	writer.Write(`:`, jsonfmt.DelimiterType)
	writer.Write(` `, jsonfmt.WhiteSpaceType)
	writer.Write(`4`, jsonfmt.NumberType)
	writer.Newline()
	writer.Write(`}`, jsonfmt.DelimiterType)

	expected := []jsontree.Line{
		{{'{', termbox.ColorWhite}},
		{
			{' ', 0},
			{' ', 0},
			{' ', 0},
			{' ', 0},
			{'"', termbox.ColorBlack},
			{'t', termbox.ColorBlack},
			{'e', termbox.ColorBlack},
			{'s', termbox.ColorBlack},
			{'t', termbox.ColorBlack},
			{'"', termbox.ColorBlack},
			{':', termbox.ColorWhite},
			{' ', 0},
			{'4', termbox.ColorYellow},
		},
		{{'}', termbox.ColorWhite}},
	}
	actual := writer.Lines

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected:\n%v but received:\n%v", expected, actual)
	}
}
