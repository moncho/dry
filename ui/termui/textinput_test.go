package termui

import (
	"errors"
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
		{"multiline initial value, no multi", arg{"hey\nthere", false}, "hey\nthere"},
	}

	for _, tt := range tests {
		input := NewTextInput(tt.arg.text)
		text, _ := input.Text()

		if text != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, text, tt.want)

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
			EventHandledCallback: func(e termbox.Event) error { return nil },
		}
		input := NewTextInput("")
		go func() {
			for _, e := range tt.arg.events {
				c <- e
			}
		}()
		err := input.OnFocus(events)

		if err != nil {
			t.Errorf("Got an error on Focus: %s", err.Error())
		}

		text, _ := input.Text()

		if text != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, text, tt.want)

		}
	}
}

func Test_TextInput_FocusReleaseWithEvent(t *testing.T) {

	type arg struct {
		events []termbox.Event
	}
	tests := []struct {
		name         string
		arg          arg
		expectedText string
		cancelByUser bool
	}{
		{"Enter key releases Focus on single line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"",
			false},
		{"Esc key releases Focus on single line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEsc,
					},
				},
			},
			"",
			true},
		{"Esc key releases Focus on multi line input",
			arg{
				[]termbox.Event{
					{
						Key: termbox.KeyEsc,
					},
				},
			},
			"", true},
	}

	for _, tt := range tests {
		c := make(chan termbox.Event)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e termbox.Event) error { return nil },
		}

		input := NewTextInput("")
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
		text, cancelByUser := input.Text()
		if text != tt.expectedText {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, text, tt.expectedText)

		}
		if cancelByUser != tt.cancelByUser {
			t.Errorf("%q. NewTextInput().Text() says cancelByUser = %t, but expected %t", tt.name, cancelByUser, tt.cancelByUser)

		}
	}
}

func Test_TextInput_FocusReleaseByClosingChan(t *testing.T) {
	c := make(chan termbox.Event)
	events := ui.EventSource{
		Events:               c,
		EventHandledCallback: func(e termbox.Event) error { return nil },
	}
	input := NewTextInput("")

	go close(c)
	err := input.OnFocus(events)
	if err != nil {
		t.Errorf("Got an error on Focus: %s", err.Error())
	}

}

func Test_TextInput_ErrorOnCallbackReleasesFocus(t *testing.T) {
	c := make(chan termbox.Event)
	events := ui.EventSource{
		Events:               c,
		EventHandledCallback: func(e termbox.Event) error { return errors.New("Everything is wrong") },
	}
	input := NewTextInput("")

	go func() {
		c <- termbox.Event{}
	}()

	err := input.OnFocus(events)
	close(c)

	if err == nil {
		t.Error("Was expecting an error on Focus")
	}
}

func Test_TextInput_Buffer(t *testing.T) {
	type arg struct {
		text string
	}
	tests := []struct {
		name string
		arg  arg
		want map[image.Point]termui.Cell
	}{
		{"no text provided, single line", arg{""},
			map[image.Point]termui.Cell{
				{X: 0, Y: 0}: {Ch: '┌', Fg: 8, Bg: 0},
				{X: 1, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 2, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 3, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 4, Y: 0}: {Ch: '┐', Fg: 8, Bg: 0},

				{X: 0, Y: 1}: {Ch: '│', Fg: 8, Bg: 0},
				{X: 1, Y: 1}: {Ch: ' ', Fg: 0, Bg: 0},
				{X: 2, Y: 1}: {Ch: ' ', Fg: 0, Bg: 0},
				{X: 3, Y: 1}: {Ch: ' ', Fg: 0, Bg: 0},
				{X: 4, Y: 1}: {Ch: '│', Fg: 8, Bg: 0},

				{X: 0, Y: 2}: {Ch: '└', Fg: 8, Bg: 0},
				{X: 1, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 2, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 3, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 4, Y: 2}: {Ch: '┘', Fg: 8, Bg: 0},
			},
		},
		{"text provided, single line", arg{"hey"},
			map[image.Point]termui.Cell{
				{X: 0, Y: 0}: {Ch: '┌', Fg: 8, Bg: 0},
				{X: 1, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 2, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 3, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 4, Y: 0}: {Ch: '┐', Fg: 8, Bg: 0},

				{X: 0, Y: 1}: {Ch: '│', Fg: 8, Bg: 0},
				{X: 1, Y: 1}: {Ch: 'h', Fg: 0, Bg: 0},
				{X: 2, Y: 1}: {Ch: 'e', Fg: 0, Bg: 0},
				{X: 3, Y: 1}: {Ch: 'y', Fg: 0, Bg: 0},
				{X: 4, Y: 1}: {Ch: '│', Fg: 8, Bg: 0},

				{X: 0, Y: 2}: {Ch: '└', Fg: 8, Bg: 0},
				{X: 1, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 2, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 3, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 4, Y: 2}: {Ch: '┘', Fg: 8, Bg: 0},
			},
		},
		{"text provided larger than widget width", arg{"nope"},
			map[image.Point]termui.Cell{
				{X: 0, Y: 0}: {Ch: '┌', Fg: 8, Bg: 0},
				{X: 1, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 2, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 3, Y: 0}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 4, Y: 0}: {Ch: '┐', Fg: 8, Bg: 0},

				{X: 0, Y: 1}: {Ch: '│', Fg: 8, Bg: 0},
				{X: 1, Y: 1}: {Ch: 'o', Fg: 0, Bg: 0},
				{X: 2, Y: 1}: {Ch: 'p', Fg: 0, Bg: 0},
				{X: 3, Y: 1}: {Ch: 'e', Fg: 0, Bg: 0},
				{X: 4, Y: 1}: {Ch: '│', Fg: 8, Bg: 0},

				{X: 0, Y: 2}: {Ch: '└', Fg: 8, Bg: 0},
				{X: 1, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 2, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 3, Y: 2}: {Ch: '─', Fg: 8, Bg: 0},
				{X: 4, Y: 2}: {Ch: '┘', Fg: 8, Bg: 0},
			},
		},
	}

	for _, tt := range tests {
		input := NewTextInput(tt.arg.text)
		input.Width = 5
		input.Height = 3
		if !equal(input.Buffer().CellMap, tt.want) {
			t.Errorf("%q. NewTextInput().Buffer().CellMap = %v, want %v", tt.name,
				input.Buffer().CellMap, tt.want)

		}
	}
}

func Test_TextInput_RemoveCharsFromInput(t *testing.T) {

	type arg struct {
		text   string
		events []termbox.Event
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{"move to the end, remove last character",
			arg{
				"here we are",
				[]termbox.Event{
					{
						Key: termbox.KeyCtrlE,
					},
					{
						Key: termbox.KeyBackspace,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"here we ar"},
		{"move to the start, remove first character",
			arg{
				"here we are",
				[]termbox.Event{
					{
						Key: termbox.KeyCtrlA,
					},
					{
						Key: termbox.KeyCtrlD,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"ere we are"},
		{"move to the start, remove until the end",
			arg{
				"here we are",
				[]termbox.Event{
					{
						Key: termbox.KeyCtrlA,
					},
					{
						Key: termbox.KeyCtrlK,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			""},
		{"move to the end, then back two characters, remove until the end",
			arg{
				"here we are",
				[]termbox.Event{
					{
						Key: termbox.KeyCtrlE,
					},
					{
						Key: termbox.KeyArrowLeft,
					},
					{
						Key: termbox.KeyArrowLeft,
					},
					{
						Key: termbox.KeyCtrlK,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"here we a"},
	}

	for _, tt := range tests {
		c := make(chan termbox.Event)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e termbox.Event) error { return nil },
		}
		input := NewTextInput(tt.arg.text)
		go func() {
			for _, e := range tt.arg.events {
				c <- e
			}
		}()
		err := input.OnFocus(events)

		if err != nil {
			t.Errorf("Got an error on Focus: %s", err.Error())
		}

		text, _ := input.Text()

		if text != tt.want {
			t.Errorf("%q. NewTextInput().Text() = '%v', want '%v'", tt.name, text, tt.want)

		}
	}
}

func Test_TextInput_NonAsciiChars(t *testing.T) {
	type arg struct {
		text   string
		events []termbox.Event
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{
			"add non-ascii char",
			arg{
				"lets try with",
				[]termbox.Event{
					{
						Key: termbox.KeySpace,
					},
					{
						Ch: 'ñ',
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"lets try with ñ"},
		{
			"remove non-ascii char",
			arg{
				"lets try with ñ",
				[]termbox.Event{
					{
						Key: termbox.KeyBackspace,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"lets try with "},
		{
			"lets try some Chinese",
			arg{
				"世界",
				[]termbox.Event{
					{
						Key: termbox.KeyBackspace,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"世"},
		{
			"Chinese seems like fun",
			arg{
				"Hello, ",
				[]termbox.Event{
					{
						Ch: '世',
					},
					{
						Ch: '界',
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			"Hello, 世界"},
	}

	for _, tt := range tests {
		c := make(chan termbox.Event)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e termbox.Event) error { return nil },
		}
		input := NewTextInput(tt.arg.text)
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

		text, _ := input.Text()

		if text != tt.want {
			t.Errorf("%q. NewTextInput().Text() = '%v', want '%v'", tt.name, text, tt.want)

		}
	}
}

func Test_TextInput_CursorPosition(t *testing.T) {
	text := "here we are"
	textLength := len(text)

	type arg struct {
		text   string
		events []termbox.Event
	}
	tests := []struct {
		name   string
		arg    arg
		cursor int
	}{
		{"at the end already",
			arg{
				text,
				[]termbox.Event{
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			textLength,
		},
		{"move to the end",
			arg{
				text,
				[]termbox.Event{
					{
						Key: termbox.KeyCtrlE,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			textLength,
		},
		{"move to the start",
			arg{
				text,
				[]termbox.Event{
					{
						Key: termbox.KeyCtrlA,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			0,
		},
		{"move backwards twice",
			arg{
				text,
				[]termbox.Event{
					{
						Key: termbox.KeyArrowLeft,
					},
					{
						Key: termbox.KeyArrowLeft,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			textLength - 2,
		},
		{"move backwards twice, then forward",
			arg{
				text,
				[]termbox.Event{
					{
						Key: termbox.KeyArrowLeft,
					},
					{
						Key: termbox.KeyArrowLeft,
					},
					{
						Key: termbox.KeyArrowRight,
					},
					{
						Key: termbox.KeyEnter,
					},
				},
			},
			textLength - 1,
		},
	}

	for _, tt := range tests {
		c := make(chan termbox.Event)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e termbox.Event) error { return nil },
		}
		input := NewTextInput(tt.arg.text)
		go func() {
			for _, e := range tt.arg.events {
				c <- e
			}
		}()
		err := input.OnFocus(events)

		if err != nil {
			t.Errorf("Got an error on Focus: %s", err.Error())
		}

		if input.cursorLinePos != tt.cursor {
			t.Errorf("%q. Expected cursor on %d, got %d", tt.name, tt.cursor, input.cursorLinePos)

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
