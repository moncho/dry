package termui

import (
	"testing"

	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

var theme = &ui.ColorTheme{Fg: ui.ColorBlack, Bg: ui.ColorWhite}

func TestMarkupTextParsing(t *testing.T) {
	tbls := [][]string{
		{"hello world", "hello world"},
		{"<red>hello world</>", "hello world"},
		{"<red>[hello]</> world", "[hello] world"},
		{"[1] hello world", "[1] hello world"},
		{"<white>[1]</> [hello] world", "[1] [hello] world"},
		{"[hello world]", "[hello world]"},
		{"", ""},
		{"[hello world)", "[hello world)"},
		{"[0] <red>hello</> <blue>world</>!", "[0] hello world!"},
	}
	tb := markupTextBuilder{ui.NewMarkup(theme)}
	for _, s := range tbls {
		result := cellsToString(tb.Build(s[0], termui.Attribute(theme.Fg), termui.Attribute(theme.Bg)))
		if s[1] != result {
			t.Errorf("\ninput :%s\nshould be :%s\ngot:%s", s[0], s[1], result)
		}
	}
}

func cellsToString(cells []termui.Cell) string {
	var result []rune
	for _, c := range cells {
		result = append(result, c.Ch)
	}
	return string(result)
}
