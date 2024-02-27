package ui

import (
	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

// TextRenderer renders text
type TextRenderer interface {
	Render(string) int
}

// ScreenTextRenderer renders text on a screen
type ScreenTextRenderer struct {
	renderer  styledRuneRenderer
	style     tcell.Style
	withStyle bool
	x, y      int
	width     int
}

// NewRenderer creates ScreenTextRenderer
func NewRenderer(s styledRuneRenderer) ScreenTextRenderer {
	return ScreenTextRenderer{
		renderer: s,
	}
}

// WithStyle returns a renderer that will use the provided style on rendering
func (renderer ScreenTextRenderer) WithStyle(style tcell.Style) ScreenTextRenderer {
	return ScreenTextRenderer{
		renderer:  renderer.renderer,
		x:         renderer.x,
		y:         renderer.y,
		width:     renderer.width,
		withStyle: true,
		style:     style,
	}
}

// On returns a renderer that will start rendering on the given positions of the
// screen.
func (renderer ScreenTextRenderer) On(x, y int) ScreenTextRenderer {
	return ScreenTextRenderer{
		renderer:  renderer.renderer,
		x:         x,
		y:         y,
		width:     renderer.width,
		style:     renderer.style,
		withStyle: renderer.withStyle,
	}
}

// WithWidth returns a renderer with its width set
func (renderer ScreenTextRenderer) WithWidth(w int) ScreenTextRenderer {
	return ScreenTextRenderer{
		renderer:  renderer.renderer,
		x:         renderer.x,
		y:         renderer.y,
		width:     w,
		style:     renderer.style,
		withStyle: renderer.withStyle,
	}
}

// Render renders the given text on this renderer screen
func (renderer ScreenTextRenderer) Render(s string) int {
	stringWidth := 0
	maxWidth := renderer.renderer.Dimensions().Width
	virtualScreenWidth := renderer.width
	//tracks the number of screen lines used to render
	additionalLines := 0
	startCol := renderer.x
	y := renderer.y

	var style tcell.Style
	if renderer.withStyle {
		style = renderer.style
	} else {
		style = renderer.renderer.Style()
	}
	for _, char := range s {
		runewidth := runewidth.RuneWidth(char)
		stringWidth += runewidth
		//Check if a new line is going to be needed
		if stringWidth > virtualScreenWidth {
			//A new line is going to be used, the virtual screen width has to be
			//extended
			virtualScreenWidth += virtualScreenWidth + maxWidth
			additionalLines++
			y += additionalLines
			//new line, start column goes back to the beginning
			startCol = renderer.x
		}
		renderer.renderer.Render(startCol, y, char, style)
		startCol += runewidth
	}
	return additionalLines + 1
}
