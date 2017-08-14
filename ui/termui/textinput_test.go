package termui

import (
	"image"
	"sort"
	"testing"

	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

func Test_TextInput_Build(t *testing.T) {
	type arg struct {
		text  string
		multi bool
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{"no initial value, no multi", arg{"", false}, ""},
		{"one line initial value, no multi", arg{"hey", false}, "hey"},
		{"multiline initial value, multi", arg{"hey\nthere", true}, "hey\nthere"},
		{"multiline initial value, no multi", arg{"hey\nthere", false}, "hey"},
	}

	for _, tt := range tests {
		input := NewTextInput(tt.arg.text, tt.arg.multi)
		if input.Text() != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, input.Text(), tt.want)

		}
	}
}

func Test_TextInput_Focus(t *testing.T) {

	type arg struct {
		events []termbox.Event
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{"no initial value, no multi, no input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			""},
		{"no initial value, no multi, input send with events",
			arg{
				[]termbox.Event{
					{
						Ch: 'h',
					},
					{
						Ch: 'e',
					},
					{
						Ch: 'y',
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"hey"},
	}

	for _, tt := range tests {
		c := make(chan termbox.Event)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func() error { return nil },
		}
		input := NewTextInput("", false)
		go func() {
			for _, e := range tt.arg.events {
				c <- e
			}
		}()
		err := input.OnFocus(events)

		if err != nil {
			t.Errorf("Got an error on Focus: %s", err.Error())
		}

		if input.Text() != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, input.Text(), tt.want)

		}
	}
}

func Test_TextInput_FocusReleaseWithEvent(t *testing.T) {

	type arg struct {
		events []termbox.Event
		multi  bool
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{"Enter key releases Focus on single line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEnter,
					},
				},
				false,
			},
			""},
		{"Esc key releases Focus on single line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEsc,
					},
				},
				false,
			},
			""},
		{"Esc key releases Focus on multi line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEsc,
					},
				},
				true,
			},
			""},
		{"Enter key does not release Focus on multi line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEnter,
					},
					{
						Key: termbox.KeyEsc,
					},
				},
				true,
			},
			"\n"},
	}

	for _, tt := range tests {
		c := make(chan termbox.Event)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func() error { return nil },
		}

		input := NewTextInput("", tt.arg.multi)
		go func() {
			for _, e := range tt.arg.events {
				c <- e
			}
		}()
		err := input.OnFocus(events)
		close(c)
		if err != nil {
			t.Errorf("Got an error on Focus: %s", err.Error())
		}

		if input.Text() != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, input.Text(), tt.want)

		}
	}
}

func Test_TextInput_FocusReleaseByClosingChan(t *testing.T) {
	c := make(chan termbox.Event)
	events := ui.EventSource{
		Events:               c,
		EventHandledCallback: func() error { return nil },
	}
	input := NewTextInput("", false)

	go close(c)
	err := input.OnFocus(events)
	if err != nil {
		t.Errorf("Got an error on Focus: %s", err.Error())
	}

}

func Test_TextInput_Buffer(t *testing.T) {
	type arg struct {
		text  string
		multi bool
	}
	tests := []struct {
		name string
		arg  arg
		want map[image.Point]termui.Cell
	}{
		{"no text provided, single line", arg{"", false},
			map[image.Point]termui.Cell{
				image.Point{X: 0, Y: 0}: termui.Cell{Ch: '┌', Fg: 8, Bg: 0},
				image.Point{X: 1, Y: 0}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 2, Y: 0}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 3, Y: 0}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 4, Y: 0}: termui.Cell{Ch: '┐', Fg: 8, Bg: 0},

				image.Point{X: 0, Y: 1}: termui.Cell{Ch: '│', Fg: 8, Bg: 0},
				image.Point{X: 1, Y: 1}: termui.Cell{Ch: ' ', Fg: 0, Bg: 0},
				image.Point{X: 2, Y: 1}: termui.Cell{Ch: ' ', Fg: 0, Bg: 0},
				image.Point{X: 3, Y: 1}: termui.Cell{Ch: ' ', Fg: 0, Bg: 0},
				image.Point{X: 4, Y: 1}: termui.Cell{Ch: '│', Fg: 8, Bg: 0},

				image.Point{X: 0, Y: 2}: termui.Cell{Ch: '└', Fg: 8, Bg: 0},
				image.Point{X: 1, Y: 2}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 2, Y: 2}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 3, Y: 2}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 4, Y: 2}: termui.Cell{Ch: '┘', Fg: 8, Bg: 0},
			},
		},
		{"text provided, single line", arg{"hey", false},
			map[image.Point]termui.Cell{
				image.Point{X: 0, Y: 0}: termui.Cell{Ch: '┌', Fg: 8, Bg: 0},
				image.Point{X: 1, Y: 0}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 2, Y: 0}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 3, Y: 0}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 4, Y: 0}: termui.Cell{Ch: '┐', Fg: 8, Bg: 0},

				image.Point{X: 0, Y: 1}: termui.Cell{Ch: '│', Fg: 8, Bg: 0},
				image.Point{X: 1, Y: 1}: termui.Cell{Ch: 'h', Fg: 0, Bg: 0},
				image.Point{X: 2, Y: 1}: termui.Cell{Ch: 'e', Fg: 0, Bg: 0},
				image.Point{X: 3, Y: 1}: termui.Cell{Ch: 'y', Fg: 0, Bg: 0},
				image.Point{X: 4, Y: 1}: termui.Cell{Ch: '│', Fg: 8, Bg: 0},

				image.Point{X: 0, Y: 2}: termui.Cell{Ch: '└', Fg: 8, Bg: 0},
				image.Point{X: 1, Y: 2}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 2, Y: 2}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 3, Y: 2}: termui.Cell{Ch: '─', Fg: 8, Bg: 0},
				image.Point{X: 4, Y: 2}: termui.Cell{Ch: '┘', Fg: 8, Bg: 0},
			},
		},
	}

	for _, tt := range tests {
		input := NewTextInput(tt.arg.text, tt.arg.multi)
		input.Width = 5
		input.Height = 3
		if !equal(input.Buffer().CellMap, tt.want) {
			t.Errorf("%q. NewTextInput().Buffer().CellMap = %v, want %v", tt.name,
				input.Buffer().CellMap, tt.want)

		}
	}
}

func equal(m1, m2 map[image.Point]termui.Cell) bool {
	if len(m1) != len(m2) {
		return false
	}
	keys1 := sortedKeys(m1)
	keys2 := sortedKeys(m2)
	for i, key := range keys1 {
		key2 := keys2[i]
		if key != key2 {
			return false
		}
		if m1[key] != m2[key2] {
			return false
		}

	}

	return true

}
func sortedKeys(m map[image.Point]termui.Cell) []image.Point {
	var result []image.Point
	for k := range m {
		result = append(result, k)
	}
	sort.SliceStable(result, func(i int, j int) bool {
		if result[i].X < result[j].X {
			return true
		} else if result[i].X == result[j].X {
			return result[i].Y <= result[j].Y
		}
		return false
	})
	return result
}
