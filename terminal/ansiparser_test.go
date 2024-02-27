package terminal

import "testing"

var (
	green    = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	reset    = string([]byte{27, 91, 48, 109})
	testText = "Dry have seen things you people wouldn't believe"
)

func TestRemoveEscapeCharacters(t *testing.T) {
	text := green
	text += testText
	text += reset
	parsed := RemoveANSIEscapeCharacters(text)
	if len(parsed) != 1 {
		t.Fatalf("Parsing returned wrong number of lines, expected: %d, got: %d", 1, len(parsed))
	}
	if string(parsed[0]) != testText {
		t.Errorf("Parsing did not work, result: \"%s\"", string(parsed[0]))
	}
}

func BenchmarkRemoveEscapeCharacters(b *testing.B) {
	text := green
	text += testText
	text += reset
	var lastResult [][]rune
	for i := 0; i < b.N; i++ {
		lastResult = RemoveANSIEscapeCharacters(text)
	}
	last := lastResult
	if len(last) < 0 {
		b.Errorf("Parsing returned wrong number of lines, expected: %d, got: %d", 1, len(last))
	}

}
