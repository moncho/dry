package ui

import (
	"bytes"
	`fmt`
	`strings`
	"text/template"
	`time`

	"github.com/moncho/dry/app"
	mdocker "github.com/moncho/dry/docker"
	"github.com/nsf/termbox-go"
)

// Screen is thin wrapper aroung Termbox library to provide basic display
// capabilities as required by dry.
type Screen struct {
	width    int     // Current number of columns.
	height   int     // Current number of rows.
	cleared  bool    // True after the screens gets cleared.
	layout   *Layout // Pointer to layout (gets created by screen).
	markup   *Markup // Pointer to markup processor (gets created by screen).
	pausedAt *time.Time
	cursor   *Cursor // Pointer to cursor (gets created by screen).
	header   *header
	App      *app.Dry
	footer   *Renderer
}

type Cursor struct {
	Line int
	Fg   termbox.Attribute
	Ch   rune
}

// Initializes Termbox, creates screen along with layout and markup, and
// calculates current screen dimensions. Once initialized the screen is
// ready for display.
func NewScreen(dry *app.Dry) *Screen {

	if err := termbox.Init(); err != nil {
		panic(err)
	}
	termbox.SetOutputMode(termbox.Output256)
	screen := &Screen{}
	screen.layout = NewLayout()
	screen.markup = NewMarkup()
	screen.cursor = &Cursor{Line: 0, Fg: termbox.ColorRed, Ch: '옷'}
	screen.header = newHeader(dry.State)
	screen.App = dry

	return screen.Resize()
}

// Close gets called upon program termination to close the Termbox.
func (screen *Screen) Close() *Screen {
	termbox.Close()
	return screen
}

// Resize gets called when the screen is being resized. It recalculates screen
// dimensions and requests to clear the screen on next update.
func (screen *Screen) Resize() *Screen {
	screen.width, screen.height = termbox.Size()
	screen.cleared = false
	return screen
}

// Pause is a toggle function that either creates a timestamp of the pause
// request or resets it to nil.
func (screen *Screen) Pause(pause bool) *Screen {
	if pause {
		screen.pausedAt = new(time.Time)
		*screen.pausedAt = time.Now()
	} else {
		screen.pausedAt = nil
	}

	return screen
}

// Clear makes the entire screen blank using default background color.
func (screen *Screen) Clear() *Screen {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	screen.cleared = true
	termbox.Flush()
	return screen
}

func (screen *Screen) Sync() *Screen {
	termbox.Sync()
	return screen
}

// ClearLine erases the contents of the line starting from (x,y) coordinate
// till the end of the line.
func (screen *Screen) ClearLine(x int, y int) *Screen {
	for i := x; i < screen.width; i++ {
		termbox.SetCell(i, y, ' ', termbox.ColorDefault, termbox.ColorDefault)
	}
	termbox.Flush()

	return screen
}

// Render accepts variable number of arguments and knows how to display
// containers, current time, and an arbitrary string.
func (screen *Screen) Render(objects ...interface{}) *Screen {
	for _, ptr := range objects {
		switch ptr.(type) {
		case *mdocker.DockerDaemon:
			server := ptr.(*mdocker.DockerDaemon)
			screen.layout.Content = NewDockerRenderer(
				server,
				screen.cursor,
				screen.App.State.SortMode)

		case *mdocker.Stats:
			_ = "breakpoint"
			stats := ptr.(*mdocker.Stats)
			screen.layout.Content = NewDockerStatsRenderer(
				stats,
			)
		case string:
			s := stringRenderer(ptr.(string))
			screen.layout.Content = s
		default:
			s := stringRenderer(fmt.Sprint("Dont know how to render a %s", ptr))
			screen.layout.Content = s

		}
	}
	screen.layout.Header = screen.header
	screen.render(0, screen.layout.Render())
	screen.renderLine(0, screen.height-1, app.KeyMappings)
	termbox.Flush()
	return screen
}

// RenderLine takes the incoming string, tokenizes it to extract markup
// elements, and displays it all starting at (x,y) location.
func (screen *Screen) RenderLine(x int, y int, str string) {
	screen.renderLine(x, y, str)
	termbox.Flush()
}

// RenderLine takes the incoming string, tokenizes it to extract markup
// elements, and displays it all starting at (x,y) location.
func (screen *Screen) renderLine(x int, y int, str string) {
	start, column := 0, 0

	for _, token := range screen.markup.Tokenize(str) {
		// First check if it's a tag. Tags are eaten up and not displayed.
		if screen.markup.IsTag(token) {
			continue
		}

		// Here comes the actual text: display it one character at a time.
		for i, char := range token {
			if !screen.markup.RightAligned {
				start = x + column
				column++
			} else {
				start = screen.width - len(token) + i
			}
			termbox.SetCell(start, y, char, screen.markup.Foreground, screen.markup.Background)
		}
	}
}

func (screen *Screen) MoveCursorDown() {
	screen.cursor.Line = screen.cursor.Line + 1

}
func (screen *Screen) MoveCursorUp() {
	screen.cursor.Line = screen.cursor.Line - 1
}

func (screen *Screen) CursorPosition() int {
	return screen.cursor.Line
}

func (screen *Screen) render(column int, str string) {
	if !screen.cleared {
		screen.Clear()
	}
	for row, line := range strings.Split(str, "\n") {
		screen.RenderLine(column, row, line)
	}
}

type header struct {
	template *template.Template
	appState *app.AppState
}

func newHeader(state *app.AppState) *header {
	return &header{
		buildHeaderTemplate(),
		state,
	}
}
func buildHeaderTemplate() *template.Template {
	markup := `{{.AppMessage}}<right><white>{{.Now}}</></right>`
	return template.Must(template.New(`header`).Parse(markup))
}

func (h *header) Render() string {
	vars := struct {
		Now        string // Current timestamp.
		AppMessage string
	}{
		time.Now().Format(`3:04:05pm PST`),
		h.appState.Render(),
	}

	_ = "breakpoint"
	buffer := new(bytes.Buffer)
	h.template.Execute(buffer, vars)
	return buffer.String()
}
