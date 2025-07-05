package ui

import (
	"os"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/termbox"
)

type styledRuneRenderer interface {
	Dimensions() *Dimensions
	Render(x int, y int, r rune, style tcell.Style)
	Style() tcell.Style
}

type screenStyledRuneRenderer struct {
	screen *Screen
}

func (r screenStyledRuneRenderer) Dimensions() *Dimensions {
	return r.screen.dimensions
}
func (r screenStyledRuneRenderer) Render(x int, y int, ru rune, style tcell.Style) {
	r.screen.screen.SetContent(x, y, ru, nil, style)

}
func (r screenStyledRuneRenderer) Style() tcell.Style {
	return r.screen.themeStyle
}

// InitScreen creates and initializes the tcell screen
func initScreen() (tcell.Screen, error) {
	// To enable true color, TERM is set to `xterm-truecolor` before
	// initializing tcell. Once done after that, TERM is set back to whatever it was
	term := os.Getenv("TERM")
	defer os.Setenv("TERM", term)
	os.Setenv("TERM", "xterm-truecolor")

	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	screen, err := tcell.NewScreen()

	if err != nil {
		return nil, err
	}
	if err = screen.Init(); err != nil {
		return nil, err
	}

	screen.EnableMouse()
	return screen, nil
}

func mkStyle(fg, bg termbox.Attribute) tcell.Style {
	st := tcell.StyleDefault

	f := tcell.Color(int(fg)&0x1ff) - 1
	b := tcell.Color(int(bg)&0x1ff) - 1

	f = fixColor(termbox.Output256, f)
	b = fixColor(termbox.Output256, b)
	st = st.Foreground(f).Background(b)
	if (fg|bg)&termbox.AttrBold != 0 {
		st = st.Bold(true)
	}
	if (fg|bg)&termbox.AttrUnderline != 0 {
		st = st.Underline(true)
	}
	if (fg|bg)&termbox.AttrReverse != 0 {
		st = st.Reverse(true)
	}
	return st
}
func fixColor(outputMode termbox.OutputMode, c tcell.Color) tcell.Color {
	if c == tcell.ColorDefault {
		return c
	}
	switch outputMode {
	case termbox.OutputNormal:
		c %= tcell.Color(16)
	case termbox.Output256:
		c %= tcell.Color(256)
	case termbox.Output216:
		c %= tcell.Color(216)
		c += tcell.Color(16)
	case termbox.OutputGrayscale:
		c %= tcell.Color(24)
		c += tcell.Color(232)
	default:
		c = tcell.ColorDefault
	}
	return c
}

func screenDimensions(s tcell.Screen) *Dimensions {
	w, h := s.Size()
	return &Dimensions{Width: w, Height: h}
}
