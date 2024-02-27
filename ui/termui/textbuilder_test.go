package termui

import (
	"reflect"
	"testing"

	"github.com/gdamore/tcell/termbox"
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

var theme = &ui.ColorTheme{Fg: ui.ColorBlack, Bg: ui.ColorWhite}

func TestMarkupTextParsing(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
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

	for _, tt := range tests {

		result := cellsToString(tb.Build(
			tt.arg,
			gizaktermui.Attribute(theme.Fg), gizaktermui.Attribute(theme.Bg)))
		if tt.want != result {
			t.Errorf("\ninput :%s\nshould be :%s\ngot:%s", tt.arg, tt.want, result)
		}
	}

}

func cellsToString(cells []gizaktermui.Cell) string {
	var result []rune
	for _, c := range cells {
		result = append(result, c.Ch)
	}
	return string(result)
}

func Test_markupTextBuilder_Build(t *testing.T) {
	type fields struct {
		markup *ui.Markup
	}
	type args struct {
		str string
		fg  gizaktermui.Attribute
		bg  gizaktermui.Attribute
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []gizaktermui.Cell
	}{
		{
			name: "basic test",
			fields: fields{markup: &ui.Markup{
				Foreground: termbox.ColorBlack,
				Background: termbox.ColorBlue,
			}},
			args: args{str: "yup", fg: gizaktermui.ColorBlack, bg: gizaktermui.ColorBlue},
			want: []gizaktermui.Cell{
				{
					Ch: 'y',
					Fg: gizaktermui.ColorBlack,
					Bg: gizaktermui.ColorBlue,
				},
				{
					Ch: 'u',
					Fg: gizaktermui.ColorBlack,
					Bg: gizaktermui.ColorBlue,
				},
				{
					Ch: 'p',
					Fg: gizaktermui.ColorBlack,
					Bg: gizaktermui.ColorBlue,
				},
			},
		},
	}
	for _, tt := range tests {
		mtb := markupTextBuilder{
			markup: tt.fields.markup,
		}
		if got := mtb.Build(tt.args.str, tt.args.fg, tt.args.bg); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. markupTextBuilder.Build() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_wrapTx(t *testing.T) {
	type args struct {
		cs []gizaktermui.Cell
		wl int
	}
	tests := []struct {
		name string
		args args
		want []gizaktermui.Cell
	}{
		{
			name: "test wrapping no need for wrapping",
			args: args{
				cs: []gizaktermui.Cell{
					{
						Ch: 'y',
					},
					{
						Ch: 'u',
					},
					{
						Ch: 'p',
					},
				},
				wl: 10,
			},
			want: []gizaktermui.Cell{
				{
					Ch: 'y',
				},
				{
					Ch: 'u',
				},
				{
					Ch: 'p',
				},
			},
		},
	}
	for _, tt := range tests {
		if got := wrapTx(tt.args.cs, tt.args.wl); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. wrapTx() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
