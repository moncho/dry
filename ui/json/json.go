package json

import (
	"fmt"

	enconding "encoding/json"

	"github.com/maxzender/jv/colorwriter"
	"github.com/maxzender/jv/jsonfmt"
	"github.com/maxzender/jv/jsontree"
	"github.com/maxzender/jv/terminal"
	"github.com/moncho/dry/ui"
	termbox "github.com/nsf/termbox-go"
)

var (
	colorMap = map[jsonfmt.TokenType]termbox.Attribute{
		jsonfmt.DelimiterType: termbox.ColorDefault,
		jsonfmt.BoolType:      termbox.ColorRed,
		jsonfmt.StringType:    termbox.ColorGreen,
		jsonfmt.NumberType:    termbox.ColorYellow,
		jsonfmt.NullType:      termbox.ColorMagenta,
		jsonfmt.KeyType:       termbox.ColorBlue,
	}
)

//JSONViewer is a JSON viewer
type JSONViewer struct {
	screen *ui.Screen
	term   *terminal.Terminal
	tree   *jsontree.JsonTree
}

//NewJSONViewer creates a new JSONViewer with the given content
func NewJSONViewer(screen *ui.Screen, content interface{}) (*JSONViewer, error) {
	jv := JSONViewer{
		screen: screen,
	}
	err := jv.Content(content)
	return &jv, err
}

//Focus is a blocking call that starts handling terminal events
func (jv *JSONViewer) Focus(events <-chan termbox.Event) error {
	jv.term = &terminal.Terminal{
		Width:  jv.screen.Dimensions.Width,
		Height: jv.screen.Dimensions.Height,
		Tree:   jv.tree}
	jv.term.Render()

	for e := range events {
		if e.Ch == 'q' || e.Key == termbox.KeyEsc {
			break
		}
		jv.handleKeypress(e)
		jv.term.Render()
	}

	termbox.HideCursor()
	jv.screen.ClearAndFlush()
	jv.screen.Sync()
	return nil
}

//Content sets the content of this viewer
func (jv *JSONViewer) Content(content interface{}) error {

	json, err := toJSON(content)
	if err != nil {
		return fmt.Errorf("Error converting %v to json: %s", content, err.Error())
	}
	writer := colorwriter.New(colorMap, termbox.ColorDefault)
	formatter := jsonfmt.New(json, writer)
	if err := formatter.Format(); err != nil {
		return err
	}
	formattedJSON := writer.Lines

	jv.tree = jsontree.New(formattedJSON)
	return nil
}

func (jv *JSONViewer) handleKeypress(e termbox.Event) {
	t := jv.term
	j := jv.tree
	if e.Ch == 0 {
		switch e.Key {
		case termbox.KeyArrowUp:
			t.MoveCursor(0, -1)
		case termbox.KeyArrowDown:
			t.MoveCursor(0, +1)
		case termbox.KeyArrowLeft:
			t.MoveCursor(-1, 0)
		case termbox.KeyArrowRight:
			t.MoveCursor(+1, 0)
		case termbox.KeyEnter, termbox.KeySpace:
			j.ToggleLine(t.CursorY + t.OffsetY)
		}
	} else {
		switch e.Ch {
		case 'h':
			t.MoveCursor(-1, 0)
		case 'j':
			t.MoveCursor(0, +1)
		case 'k':
			t.MoveCursor(0, -1)
		case 'l':
			t.MoveCursor(+1, 0)
		}
	}
}

func toJSON(data interface{}) ([]byte, error) {
	return enconding.Marshal(data)

}
