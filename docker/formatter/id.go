package formatter

import (
	"strings"
)

const shortLen = 12

// TruncateID returns a shorthand version of the given string identifier
func TruncateID(id string) string {
	if i := strings.IndexRune(id, ':'); i >= 0 {
		id = id[i+1:]
	}
	if len(id) > shortLen {
		id = id[:shortLen]
	}
	return id
}
