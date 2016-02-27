package ui

import (
	`regexp`
	`strings`

	"github.com/nsf/termbox-go"
)

var supportedTags = supportedTagsRegexp()
var tagsToAttribute = tags()

func supportedTagsRegexp() *regexp.Regexp {
	arr := []string{}

	for tag := range tagsToAttribute {
		arr = append(arr, `</?`+tag+`>`)
	}

	return regexp.MustCompile(strings.Join(arr, `|`))
}

// tags returns regular expression that matches all possible tags
// supported by the markup, i.e. </?black>|</?red>| ... |<?b>| ... |</?right>
func tags() map[string]termbox.Attribute {
	tags := make(map[string]termbox.Attribute)
	tags[`/`] = termbox.ColorDefault
	tags[`black`] = termbox.ColorBlack
	tags[`red`] = termbox.ColorRed
	tags[`red00`] = termbox.ColorRed
	tags[`green`] = termbox.ColorGreen
	tags[`yellow`] = termbox.ColorYellow
	tags[`blue`] = termbox.ColorBlue
	tags[`magenta`] = termbox.ColorMagenta
	tags[`cyan`] = termbox.ColorCyan
	tags[`cyan0`] = termbox.ColorCyan
	tags[`white`] = termbox.ColorWhite
	tags[`grey`] = 0xE9
	tags[`grey2`] = 0xF4
	tags[`darkgrey`] = 0xE8
	tags[`right`] = termbox.ColorDefault // Termbox can combine attributes and a single color using bitwise OR.
	tags[`b`] = termbox.AttrBold
	tags[`u`] = termbox.AttrUnderline
	tags[`r`] = termbox.AttrReverse
	return tags
}

// Markup implements some minimalistic text formatting conventions that
// get translated to Termbox colors and attributes. To colorize a string
// wrap it in <color-name>...</> tags. Unlike HTML each tag sets a new
// color whereas the </> tag changes color back to default. For example:
//
// <green>Hello, <red>world!</>
//
// The color tags could be combined with the attributes: <b>...</b> for
// bold, <u>...</u> for underline, and <r>...</r> for reverse. Unlike
// colors the attributes require matching closing tag.
//
// The <right>...</right> tag is used to right align the enclosed string
type Markup struct {
	Foreground   termbox.Attribute            // Foreground color.
	Background   termbox.Attribute            // Background color (so far always termbox.ColorDefault).
	RightAligned bool                         // True when the string is right aligned.
	tags         map[string]termbox.Attribute // Tags to Termbox translation hash.
	regex        *regexp.Regexp               // Regex to identify the supported tag names.
}

// NewMarkup creates a markup to define tag to Termbox translation rules and store default
// colors and column alignments.
func NewMarkup() *Markup {
	//Due to how markup is currently being used, the tag character lengh must be
	//the same for all color tags (the magic number is 5, because white) to avoid,
	//text alignment problems, hence the strange tag names.

	markup := &Markup{}
	markup.Foreground = termbox.ColorDefault
	markup.Background = termbox.ColorDefault
	markup.RightAligned = false
	markup.tags = tagsToAttribute
	markup.regex = supportedTags
	return markup
}

// IsTag returns true when the given string looks like markup tag. When the
// tag name matches one of the markup-supported tags it gets translated to
// relevant Termbox attributes and colors.
func (markup *Markup) IsTag(str string) bool {
	tag, open := probeForTag(str)

	if tag == `` {
		return false
	}

	return markup.process(tag, open)
}

//-----------------------------------------------------------------------------
func (markup *Markup) process(tag string, open bool) bool {
	if attribute, ok := markup.tags[tag]; ok {
		switch tag {
		case `right`:
			markup.RightAligned = open // On for <right>, off for </right>.
		default:
			if open {
				if attribute >= termbox.AttrBold {
					markup.Foreground |= attribute // Set the Termbox attribute.
				} else {
					markup.Foreground = attribute // Set the Termbox color.
				}
			} else {
				if attribute >= termbox.AttrBold {
					markup.Foreground &= ^attribute // Clear the Termbox attribute.
				} else {
					markup.Foreground = termbox.ColorDefault
				}
			}
		}
	}

	return true
}

//-----------------------------------------------------------------------------
func probeForTag(str string) (string, bool) {
	if len(str) > 2 && str[0:1] == `<` && str[len(str)-1:] == `>` {
		return extractTagName(str), str[1:2] != `/`
	}

	return ``, false
}

// Extract tag name from the given tag, i.e. `<hello>` => `hello`.
func extractTagName(str string) string {
	if len(str) < 3 {
		return ``
	} else if str[1:2] != `/` {
		return str[1 : len(str)-1]
	} else if len(str) > 3 {
		return str[2 : len(str)-1]
	}

	return `/`
}

// Tokenize works just like strings.Split() except the resulting array includes
// the delimiters. For example, the "<green>Hello, <red>world!</>" string when
// tokenized by tags produces the following:
//
//   [0] "<green>"
//   [1] "Hello, "
//   [2] "<red>"
//   [3] "world!"
//   [4] "</>"
//
func Tokenize(str string, regex *regexp.Regexp) []string {
	matches := regex.FindAllStringIndex(str, -1)
	strings := make([]string, 0, len(matches))

	head, tail := 0, 0
	for _, match := range matches {
		tail = match[0]
		if match[1] != 0 {
			if head != 0 || tail != 0 {
				// Apend the text between tags.
				strings = append(strings, str[head:tail])
			}
			// Append the tag itmarkup.
			strings = append(strings, str[match[0]:match[1]])
		}
		head = match[1]
	}

	if head != len(str) && tail != len(str) {
		strings = append(strings, str[head:])
	}

	return strings
}
