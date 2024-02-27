package terminal

import (
	"unicode"
	"unicode/utf8"
)

const (
	modeNormal    = iota
	modePreEscape = iota
	modeEscape    = iota
)

// ansiParser can handle text with ANSI escape codes
type ansiParser struct {
	mode                 int
	cursor               int
	escapeStartedAt      int
	instructions         []string
	instructionStartedAt int
	buffer               buffer
}

// RemoveANSIEscapeCharacters removes from the given string ANSI escape codes
func RemoveANSIEscapeCharacters(s string) [][]rune {
	p := ansiParser{mode: modeNormal} //, ansi: ansi}
	text := []byte(s)
	length := len(text)
	for p.cursor = 0; p.cursor < length; {
		char, charLen := utf8.DecodeRune(text[p.cursor:])

		switch p.mode {
		case modeEscape:
			// Inside an escape code - figure out its code and its instructions.
			p.handleEscape(char)
		case modePreEscape:
			// Received an escape character but aren't inside an escape sequence yet
			p.handlePreEscape(char)
		case modeNormal:
			// Outside of an escape sequence entirely, normal input
			p.handleNormal(char)
		}

		p.cursor += charLen
	}
	return p.buffer.content
}

func (p *ansiParser) handleEscape(char rune) {
	char = unicode.ToUpper(char)
	switch char {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// Part of an instruction
	case ';':
		//p.addInstruction()
		p.instructionStartedAt = p.cursor + utf8.RuneLen(';')
	case 'Q', 'K', 'G', 'A', 'B', 'C', 'D', 'M':
		//p.addInstruction()
		//p.applyEscape(char, p.instructions)
		p.mode = modeNormal
	default:
		// unrecognized character, abort the escapeCode
		p.cursor = p.escapeStartedAt
		p.mode = modeNormal
	}
}

func (p *ansiParser) handleNormal(char rune) {
	switch char {
	case '\n': //newLine
	case '\r': //carriageReturn
	case '\b': //backspace
	case '\x1b': //Esc+[
		p.escapeStartedAt = p.cursor
		p.mode = modePreEscape
	default:
		p.buffer.append(char)
	}
}

func (p *ansiParser) handlePreEscape(char rune) {
	switch char {
	case '[':
		p.instructionStartedAt = p.cursor + utf8.RuneLen('[')
		p.instructions = make([]string, 0, 1)
		p.mode = modeEscape
	default:
		// Not an escape code, false alarm
		p.cursor = p.escapeStartedAt
		p.mode = modeNormal
	}
}

/*func (p *ansiParser) addInstruction() {
	instruction := string(p.ansi[p.instructionStartedAt:p.cursor])
	if instruction != "" {
		p.instructions = append(p.instructions, instruction)
	}
}*/

// Apply an escape sequence to the buffer, does nothing by default
func (p *ansiParser) applyEscape(code rune, instructions []string) {

	/*
		if len(instructions) == 0 {
			// Ensure we always have a first instruction
			instructions = []string{""}
		}

		switch code {
		case 'M':
		case 'G':
		case 'K':
		case 'A':
		case 'B':
		case 'C':
		case 'D':
		}*/
}

// A buffer for the parser, tracks current position and characters
type buffer struct {
	x       int
	y       int
	content [][]rune
}

// Write a character to the buffer's current position
func (s *buffer) write(data rune) {
	s.growHeight()

	line := s.content[s.y]
	line = s.growLineWidth(line)

	line[s.x] = data
	s.content[s.y] = line
}

// Append a character to the buffer
func (s *buffer) append(data rune) {
	s.write(data)
	s.x++
}

// Add rows to the buffer if necessary
func (s *buffer) growHeight() {
	contentLen := len(s.content)
	if contentLen <= s.y {
		for i := len(s.content); i <= s.y; i++ {
			s.content = append(s.content, make([]rune, 0, 80))
		}
	}
}

// Add columns to the current line if necessary
func (s *buffer) growLineWidth(line []rune) []rune {
	for i := len(line); i <= s.x; i++ {
		line = append(line, 0)
	}
	return line
}
