package jsonfmt

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode"
)

type example struct {
	input    string
	expected string
}

// Writes the received string into a byte buffer and
// maps the received token type to a (color) value
type stringWriter struct {
	bytes.Buffer
	colorMap map[TokenType]string
}

func (w *stringWriter) Write(s string, t TokenType) {
	w.Buffer.WriteString(fmt.Sprintf("%s%s", w.colorMap[t], s))
}

func (w *stringWriter) Newline() {
	w.Buffer.WriteString("\n")
}

// Test correct colorization of output
var colorExamples = []example{
	{`{}`, `RED{}`},
	{`{"test":true}`, `RED{WHITE"test"RED:BLUEtrueRED}`},
	{`{"test":"foo"}`, `RED{WHITE"test"RED:GREEN"foo"RED}`},
	{`{"test":4}`, `RED{WHITE"test"RED:YELLOW4RED}`},
	{`{"test":3.14159265359}`, `RED{WHITE"test"RED:YELLOW3.14159265359RED}`},
	{`{"test":null}`, `RED{WHITE"test"RED:BLACKnullRED}`},
}

func TestFormatColors(t *testing.T) {
	colorMap := map[TokenType]string{
		DelimiterType: "RED",
		BoolType:      "BLUE",
		StringType:    "GREEN",
		NumberType:    "YELLOW",
		NullType:      "BLACK",
		KeyType:       "WHITE",
	}

	removeWhiteSpace := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}

	for _, tt := range colorExamples {
		writer := &stringWriter{colorMap: colorMap}
		formatter := New([]byte(tt.input), writer)

		if err := formatter.Format(); err != nil {
			t.Errorf("Format(%v): %v", tt.expected, err)
		}

		actual := removeWhiteSpace(writer.String())
		if actual != tt.expected {
			t.Errorf("Format(%v): %v, want %v", tt.input, actual, tt.expected)
		}
	}
}

// Test correct indentation of output
var indentationExamples = []example{
	{`{}`, `{}`},
	{`{"test":true}`, "{\n    \"test\": true\n}"},
	{`{"foo":true,"bar":"baz"}`, "{\n    \"bar\": \"baz\",\n    \"foo\": true\n}"},
	{`{"foo":{},"bar":["test", "baz"]}`, "{\n    \"bar\": [\n        \"test\",\n        \"baz\"\n    ],\n    \"foo\": {}\n}"},
	{`{"foo":{},"bar":{"test": "baz"}}`, "{\n    \"bar\": {\n        \"test\": \"baz\"\n    },\n    \"foo\": {}\n}"},
}

func TestFormatIndentation(t *testing.T) {
	for _, tt := range indentationExamples {
		writer := &stringWriter{}
		formatter := New([]byte(tt.input), writer)

		if err := formatter.Format(); err != nil {
			t.Errorf("Format(%v): %v", tt.expected, err)
		}

		actual := writer.String()
		if actual != tt.expected {
			t.Errorf("Format(%v):\n%v\nwant:\n%v", tt.input, actual, tt.expected)
		}
	}
}
