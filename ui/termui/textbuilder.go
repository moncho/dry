package termui

import (
	gizaktermui "github.com/gizak/termui"
	wordwrap "github.com/mitchellh/go-wordwrap"
	"github.com/moncho/dry/ui"
)

type markupTextBuilder struct {
	markup *ui.Markup
}

// Build implements TextBuilder interface.
func (mtb markupTextBuilder) Build(str string, fg, bg gizaktermui.Attribute) []gizaktermui.Cell {

	var result []gizaktermui.Cell
	markup := mtb.markup

	for _, token := range ui.Tokenize(str, ui.SupportedTags) {
		// Tags are ignored.
		if markup.IsTag(token) {
			continue
		}
		for _, char := range token {
			result = append(result,
				gizaktermui.Cell{
					Ch: char,
					Fg: gizaktermui.Attribute(markup.Foreground),
					Bg: bg})
		}
	}

	return result
}

func wrapTx(cs []gizaktermui.Cell, wl int) []gizaktermui.Cell {
	tmpCell := make([]gizaktermui.Cell, len(cs))
	copy(tmpCell, cs)

	// get the plaintext
	plain := gizaktermui.CellsToStr(cs)

	// wrap
	plainWrapped := wordwrap.WrapString(plain, uint(wl))

	// find differences and insert
	finalCell := tmpCell // finalcell will get the inserts and is what is returned

	plainRune := []rune(plain)
	plainWrappedRune := []rune(plainWrapped)
	trigger := "go"
	plainRuneNew := plainRune

	for trigger != "stop" {
		plainRune = plainRuneNew
		for i := range plainRune {
			if plainRune[i] == plainWrappedRune[i] {
				trigger = "stop"
			} else if plainRune[i] != plainWrappedRune[i] && plainWrappedRune[i] == 10 {
				trigger = "go"
				cell := gizaktermui.Cell{Ch: 10, Fg: 0, Bg: 0}
				j := i - 0

				// insert a cell into the []Cell in correct position
				tmpCell[i] = cell

				// insert the newline into plain so we avoid indexing errors
				plainRuneNew = append(plainRune, 10)
				copy(plainRuneNew[j+1:], plainRuneNew[j:])
				plainRuneNew[j] = plainWrappedRune[j]

				// restart the inner for loop until plain and plain wrapped are
				// the same; yeah, it's inefficient, but the text amounts
				// should be small
				break

			} else if plainRune[i] != plainWrappedRune[i] &&
				plainWrappedRune[i-1] == 10 && // if the prior rune is a newline
				plainRune[i] == 32 { // and this rune is a space
				trigger = "go"
				// need to delete plainRune[i] because it gets rid of an extra
				// space
				plainRuneNew = append(plainRune[:i], plainRune[i+1:]...)
				break

			} else {
				trigger = "stop" // stops the outer for loop
			}
		}
	}

	finalCell = tmpCell

	return finalCell
}
