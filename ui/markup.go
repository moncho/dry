package ui

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

// tagStyles maps markup tag names to lipgloss styles.
// Color values match the original tcell/termbox attribute mappings.
var tagStyles = map[string]lipgloss.Style{
	"black":    lipgloss.NewStyle().Foreground(lipgloss.Color("0")),
	"red":      lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	"red00":    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	"green":    lipgloss.NewStyle().Foreground(lipgloss.Color("190")),
	"yellow":   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
	"blue":     lipgloss.NewStyle().Foreground(lipgloss.Color("188")),
	"magenta":  lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	"cyan":     lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
	"cyan0":    lipgloss.NewStyle().Foreground(lipgloss.Color("181")),
	"white":    lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
	"grey":     lipgloss.NewStyle().Foreground(lipgloss.Color("233")),
	"grey2":    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	"darkgrey": lipgloss.NewStyle().Foreground(lipgloss.Color("232")),
	"b":        lipgloss.NewStyle().Bold(true),
	"u":        lipgloss.NewStyle().Underline(true),
	"r":        lipgloss.NewStyle().Reverse(true),
}

// SupportedTags is a regexp matching all supported markup tags.
var SupportedTags = supportedTagsRegexp()

func supportedTagsRegexp() *regexp.Regexp {
	var arr []string
	for tag := range tagStyles {
		arr = append(arr, `</?`+tag+`>`)
	}
	// Also match the reset tag </>
	arr = append(arr, `</>`)
	return regexp.MustCompile(strings.Join(arr, `|`))
}

// RenderMarkup converts markup-tagged strings to lipgloss-styled strings.
// Tags like <white>, <blue>, <b> are converted to ANSI-styled text.
// The </> tag (or any closing tag) resets to default foreground.
func RenderMarkup(s string) string {
	tokens := Tokenize(s, SupportedTags)
	if len(tokens) == 0 {
		return s
	}

	var b strings.Builder
	var activeStyle *lipgloss.Style

	for _, token := range tokens {
		tag, isOpen := probeForTag(token)
		if tag == "" {
			// Regular text — apply current style if any
			if activeStyle != nil {
				b.WriteString(activeStyle.Render(token))
			} else {
				b.WriteString(token)
			}
			continue
		}

		if !isOpen || tag == "/" {
			// Closing tag or </> — reset
			activeStyle = nil
			continue
		}

		if style, ok := tagStyles[tag]; ok {
			activeStyle = &style
		}
	}
	return b.String()
}

func probeForTag(str string) (string, bool) {
	if len(str) > 2 && str[0:1] == `<` && str[len(str)-1:] == `>` {
		return extractTagName(str), str[1:2] != "/"
	}
	return ``, false
}

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

// Tokenize splits the string by tag matches, including the tags themselves.
func Tokenize(str string, regex *regexp.Regexp) []string {
	matches := regex.FindAllStringIndex(str, -1)
	result := make([]string, 0, len(matches)*2)

	head := 0
	for _, match := range matches {
		if match[0] > head {
			result = append(result, str[head:match[0]])
		}
		result = append(result, str[match[0]:match[1]])
		head = match[1]
	}
	if head < len(str) {
		result = append(result, str[head:])
	}
	return result
}
