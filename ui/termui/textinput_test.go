package termui

import (
	"errors"
	"image"
	"testing"

	"github.com/gdamore/tcell"
	"github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

type cursorMock struct{}

func (c cursorMock) HideCursor() {

}
func (c cursorMock) ShowCursor(x, y int) {

}

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
		input := NewTextInput(cursorMock{}, tt.arg.text)
		text, _ := input.Text()

		if text != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, text, tt.want)

		}
	}
}

func Test_TextInput_Focus(t *testing.T) {
	type arg struct {
		events []*tcell.EventKey
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{"no initial value, no multi, no input",
			arg{
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			""},
		{"no initial value, no multi, input send with events",
			arg{
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyRune, 'h', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"hey"},
	}

	for _, tt := range tests {
		c := make(chan *tcell.EventKey)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e *tcell.EventKey) error { return nil },
		}
		input := NewTextInput(cursorMock{}, "")
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
		events []*tcell.EventKey
	}
	tests := []struct {
		name         string
		arg          arg
		expectedText string
		cancelByUser bool
	}{
		{"Enter key releases Focus on single line input",
			arg{
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"",
			false},
		{"Esc key releases Focus on single line input",
			arg{
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone),
				},
			},
			"",
			true},
		{"Esc key releases Focus on multi line input",
			arg{
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone),
				},
			},
			"", true},
	}

	for _, tt := range tests {
		c := make(chan *tcell.EventKey)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e *tcell.EventKey) error { return nil },
		}

		input := NewTextInput(cursorMock{}, "")
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
	c := make(chan *tcell.EventKey)
	events := ui.EventSource{
		Events:               c,
		EventHandledCallback: func(e *tcell.EventKey) error { return nil },
	}
	input := NewTextInput(cursorMock{}, "")

	go close(c)
	err := input.OnFocus(events)
	if err != nil {
		t.Errorf("Got an error on Focus: %s", err.Error())
	}

}

func Test_TextInput_ErrorOnCallbackReleasesFocus(t *testing.T) {
	c := make(chan *tcell.EventKey)
	events := ui.EventSource{
		Events:               c,
		EventHandledCallback: func(e *tcell.EventKey) error { return errors.New("Everything is wrong") },
	}
	input := NewTextInput(cursorMock{}, "")

	go func() {
		c <- &tcell.EventKey{}
	}()

	err := input.OnFocus(events)
	close(c)

	if err == nil {
		t.Error("Was expecting an error on OnFocus")
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
		input := NewTextInput(cursorMock{}, tt.arg.text)
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
		events []*tcell.EventKey
	}
	tests := []struct {
		name string
		arg  arg
		want string
	}{
		{"move to the end, remove last character",
			arg{
				"here we are",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"here we ar"},
		{"move to the start, remove first character",
			arg{
				"here we are",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyCtrlD, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"ere we are"},
		{"move to the start, remove until the end",
			arg{
				"here we are",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyCtrlK, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			""},
		{"move to the end, then back two characters, remove until the end",
			arg{
				"here we are",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyCtrlK, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"here we a"},
	}

	for _, tt := range tests {
		c := make(chan *tcell.EventKey)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e *tcell.EventKey) error { return nil },
		}
		input := NewTextInput(cursorMock{}, tt.arg.text)
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
		events []*tcell.EventKey
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
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyRune, 'ñ', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"lets try with ñ"},
		{
			"remove non-ascii char",
			arg{
				"lets try with ñ",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"lets try with "},
		{
			"lets try some Chinese",
			arg{
				"世界",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"世"},
		{
			"Chinese seems like fun",
			arg{
				"Hello, ",
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyRune, '世', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyRune, '界', tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			"Hello, 世界"},
	}

	for _, tt := range tests {
		c := make(chan *tcell.EventKey)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e *tcell.EventKey) error { return nil },
		}
		input := NewTextInput(cursorMock{}, tt.arg.text)
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
		events []*tcell.EventKey
	}
	tests := []struct {
		name   string
		arg    arg
		cursor int
	}{
		{"at the end already",
			arg{
				text,
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			textLength,
		},
		{"move to the end",
			arg{
				text,
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyCtrlE, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			textLength,
		},
		{"move to the start",
			arg{
				text,
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyCtrlA, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			0,
		},
		{"move backwards twice",
			arg{
				text,
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			textLength - 2,
		},
		{"move backwards twice, then forward",
			arg{
				text,
				[]*tcell.EventKey{
					tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone),
					tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone),
				},
			},
			textLength - 1,
		},
	}

	for _, tt := range tests {
		c := make(chan *tcell.EventKey)
		events := ui.EventSource{
			Events:               c,
			EventHandledCallback: func(e *tcell.EventKey) error { return nil },
		}
		input := NewTextInput(cursorMock{}, tt.arg.text)
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
