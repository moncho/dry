package ui

import "testing"

var colorizeTestTable = []struct {
	colorFunc func(string) string
	in        string
	out       string
}{
	{Blue, "textToColorize", "<blue>textToColorize</>"},
	{Red, "textToColorize", "<red>textToColorize</>"},
	{White, "textToColorize", "<white>textToColorize</>"},
	{Yellow, "textToColorize", "<yellow>textToColorize</>"},
	{Cyan, "textToColorize", "<cyan>textToColorize</>"},
}

func TestColorizers(t *testing.T) {

	for _, tt := range colorizeTestTable {
		if tt.colorFunc(tt.in) != tt.out {
			t.Errorf("%T => %s, want %s", tt.colorFunc, tt.colorFunc(tt.in), tt.out)
		}
	}

}
