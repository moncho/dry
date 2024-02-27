package termui

import (
	gizaktermui "github.com/gizak/termui"
	"github.com/moncho/dry/ui"
)

// MarkupPar is a paragraph with marked-up text
type MarkupPar struct {
	gizaktermui.Paragraph
	textBuilder gizaktermui.TextBuilder
}

// NewParFromMarkupText creates a new termui paragraph from marked-up text.
func NewParFromMarkupText(theme *ui.ColorTheme, str string) *MarkupPar {
	return &MarkupPar{Paragraph: *gizaktermui.NewParagraph(str), textBuilder: &markupTextBuilder{ui.NewMarkup(theme)}}
}

// Content sets the paragraph content to the given text.
func (p *MarkupPar) Content(str string) {
	p.Paragraph.Text = str
}

// Buffer return this paragraph content as a termui.Buffer
func (p *MarkupPar) Buffer() gizaktermui.Buffer {
	buf := p.Block.Buffer()

	fg, bg := p.TextFgColor, p.TextBgColor
	cs := p.textBuilder.Build(p.Text, fg, bg)

	// wrap if WrapLength set
	if p.WrapLength < 0 {
		cs = wrapTx(cs, p.Width-2)
	} else if p.WrapLength > 0 {
		cs = wrapTx(cs, p.WrapLength)
	}

	y, x, n := 0, 0, 0
	for y < p.InnerHeight() && n < len(cs) {
		w := cs[n].Width()
		if cs[n].Ch == '\n' || x+w > p.InnerWidth() {
			y++
			x = 0
			if cs[n].Ch == '\n' {
				n++
			}

			if y >= p.InnerHeight() {
				break
			}
			continue
		}

		buf.Set(p.InnerX()+x, p.InnerY()+y, cs[n])

		n++
		x += w
	}

	return buf
}
