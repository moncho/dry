package jsonfmt

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type TokenType int

const (
	DelimiterType = iota
	BoolType
	StringType
	NumberType
	NullType
	WhiteSpaceType
	KeyType
)

type FormatWriter interface {
	Write(string, TokenType)
	Newline()
}

const IndentationDepth = 4

type Formatter struct {
	rawJson []byte
	depth   int
	FormatWriter
}

func New(data []byte, w FormatWriter) *Formatter {
	return &Formatter{data, 0, w}
}

func (f *Formatter) Format() error {
	var v interface{}
	if err := json.Unmarshal(f.rawJson, &v); err != nil {
		return fmt.Errorf("parse error: %v", err)
	}

	f.format(v)

	return nil
}

func (f *Formatter) format(v interface{}) {
	switch value := v.(type) {
	case map[string]interface{}:
		f.formatObject(value)
	case []interface{}:
		f.formatArray(value)
	case bool:
		f.Write(fmt.Sprintf("%t", value), BoolType)
	case string:
		f.Write(fmt.Sprintf(`"%s"`, value), StringType)
	case float64:
		f.Write(strconv.FormatFloat(value, 'f', -1, 64), NumberType)
	case nil:
		f.Write("null", NullType)
	}
}

func (f *Formatter) formatObject(obj map[string]interface{}) {
	if len(obj) == 0 {
		f.Write("{}", DelimiterType)
		return
	}

	f.Write("{", DelimiterType)
	f.Newline()
	f.depth++

	var keys []string
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	end := len(keys)
	for i, key := range keys {
		val := obj[key]
		jsonKey := fmt.Sprintf("\"%s\"", key)

		f.writeIndent()
		f.Write(jsonKey, KeyType)
		f.Write(":", DelimiterType)
		f.Write(" ", WhiteSpaceType)

		f.format(val)

		i++
		if i < end {
			f.Write(",", DelimiterType)
		}

		f.Newline()
	}

	f.depth--
	f.writeIndent()
	f.Write("}", DelimiterType)
}

func (f *Formatter) formatArray(a []interface{}) {
	if len(a) == 0 {
		f.Write("[]", DelimiterType)
		return
	}

	f.Write("[", DelimiterType)
	f.Newline()
	f.depth++

	i, end := 0, len(a)
	for _, v := range a {
		f.writeIndent()
		f.format(v)

		i++
		if i < end {
			f.Write(",", DelimiterType)
		}

		f.Newline()
	}

	f.depth--
	f.writeIndent()
	f.Write("]", DelimiterType)
}

func (f *Formatter) writeIndent() {
	indentation := strings.Repeat(` `, f.depth*IndentationDepth)
	f.Write(indentation, WhiteSpaceType)
}
