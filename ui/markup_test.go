package ui

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	result := Tokenize("Dry is an interactive console application", regexp.MustCompile(" "))
	expected := []string{"Dry", " ", "is", " ", "an", " ", "interactive", " ", "console", " ", "application"}

	if len(result) != len(expected) {
		t.Errorf("Tokenization didnt work, expected: %d tokens, got: %d.",
			len(expected),
			len(result))
	}
	expectedJoin := strings.Join(expected, "")
	resultJoin := strings.Join(result, "")

	if expectedJoin != resultJoin {
		t.Errorf("Tokenization didnt work. Expected: '%s', got: '%s'.",
			expectedJoin,
			resultJoin)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Tokenization didnt work. Expected: '%s', got: '%s'.",
			expectedJoin,
			resultJoin)
	}
}

func TestTokenizeEmptyString(t *testing.T) {
	result := Tokenize("", regexp.MustCompile(" "))
	if len(result) != 0 {
		t.Errorf("Tokenization didnt work, expected: %d tokens, got: %d.",
			0,
			len(result))
	}

}

func TestTokenizeNil(t *testing.T) {
	var emptyRunes []rune
	emptyRunesString := string(emptyRunes)
	result := Tokenize(emptyRunesString, regexp.MustCompile(" "))
	if len(result) != 0 {
		t.Errorf("Tokenization didnt work, expected: %d tokens, got: %d.",
			0,
			len(result))
	}
}
