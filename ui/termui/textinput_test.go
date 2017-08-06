package termui

import (
	"testing"

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
					termbox.Event{
						Key: termbox.KeyEnter,
					},
				},
			},
			""},
		{"no initial value, no multi, input send with events",
			arg{
				[]termbox.Event{
					termbox.Event{
						Ch: 'h',
					},
					termbox.Event{
						Ch: 'e',
					},
					termbox.Event{
						Ch: 'y',
					},
					termbox.Event{
						Key: termbox.KeyEnter,
					},
				},
			},
			"hey"},
	}

	for _, tt := range tests {
		events := make(chan termbox.Event)
		input := NewTextInput("", false)
		go func() {
			for _, e := range tt.arg.events {
				events <- e
			}
		}()
		err := input.Focus(events)

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
					termbox.Event{
						Key: termbox.KeyEnter,
					},
				},
				false,
			},
			""},
		{"Esc key releases Focus on single line input",
			arg{
				[]termbox.Event{
					termbox.Event{
						Key: termbox.KeyEsc,
					},
				},
				false,
			},
			""},
		{"Esc key releases Focus on multi line input",
			arg{
				[]termbox.Event{
					termbox.Event{
						Key: termbox.KeyEsc,
					},
				},
				true,
			},
			""},
		{"Enter key does not release Focus on multi line input",
			arg{
				[]termbox.Event{
					termbox.Event{
						Key: termbox.KeyEnter,
					},
					termbox.Event{
						Key: termbox.KeyEsc,
					},
				},
				true,
			},
			"\n"},
	}

	for _, tt := range tests {
		events := make(chan termbox.Event)
		input := NewTextInput("", tt.arg.multi)
		go func() {
			for _, e := range tt.arg.events {
				events <- e
			}
		}()
		err := input.Focus(events)
		close(events)
		if err != nil {
			t.Errorf("Got an error on Focus: %s", err.Error())
		}

		if input.Text() != tt.want {
			t.Errorf("%q. NewTextInput().Text() = %v, want %v", tt.name, input.Text(), tt.want)

		}
	}
}

func Test_TextInput_FocusReleaseByClosingChan(t *testing.T) {
	events := make(chan termbox.Event)
	input := NewTextInput("", false)

	go close(events)
	err := input.Focus(events)
	if err != nil {
		t.Errorf("Got an error on Focus: %s", err.Error())
	}

}
