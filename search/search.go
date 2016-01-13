package search

import (
	"errors"
	"strings"
)

import "fmt"

const (
	lines = `line 1
	lien 2
	line3
line 4
line 5
Nope
Still nope
Really, nope`
)

//Result describes the results of a search
type Result struct {
	Hits    int
	Lines   []int
	Pattern string
	index   int //the current index i
}

//NewSearch searchs in a multiline string thos lines that match the given pattern,
//it returns:
//* the number of hits (lines)
//* the line index
func NewSearch(text [][]rune, pattern string) (*Result, error) {
	if text != nil {
		sr := &Result{Pattern: pattern, index: -1}
		for i, l := range text {
			line := string(l)
			if strings.Contains(line, pattern) {
				sr.Hits++
				sr.Lines = append(sr.Lines, i)
			}
		}
		return sr, nil
	}
	return nil, errors.New("Cannot search in an empty text.")

	/*reader := strings.NewReader(s)
	scanner := bufio.NewScanner(reader)
	line := 0

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), pattern) {
			sr.Hits++
			sr.Lines = append(sr.Lines, line)
		}
		line++
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return sr, nil*/
}

func (result *Result) String() string {
	if result.Hits > 0 {
		return fmt.Sprintf("Pattern found %d times", result.Hits)
	}
	return "Pattern not found"
}

//InitialLine sets the position for iterating the search results on the first line
//that is either has the same value or is the closest from 0 to the given line number.
//So, for a result that found that lines (1, 3, 5) were a hit:
// * InitialLine(-1) will set the internal iteration index at 0 (the default starting index)
// * InitialLine(3) will set the internal iteration index at 1.
// * InitialLine(4) will set the internal iteration index at 1.
// * InitialLine(10) will set the internal iteration index at 2.
func (result *Result) InitialLine(lineNumber int) error {
	if result.Lines == nil {
		return nohitsError()
	}
	candidate := 0
	for i, line := range result.Lines {
		if line < lineNumber {
			candidate = i
		} else {
			break
		}
	}
	result.index = candidate
	return nil
}

//NextLine returns the previous line while iterating the search results.
//So, for a result that found that lines (1, 3, 5) were a hit:
// *NextLine() should give 1
// *NextLine() should give 3
// *NextLine() should give 5
// *NextLine() should give 1
func (result *Result) NextLine() (int, error) {
	if result.Lines == nil {
		return -1, nohitsError()
	}
	result.index++
	if result.index >= len(result.Lines) {
		result.index = 0
	}
	return result.Lines[result.index], nil
}

//PreviousLine returns the previous line while iterating the search results.
//So, for a result that found that lines (1, 3, 5) were a hit:
// * PreviousLine() should give 5
// * PreviousLine() should give 3
// * PreviousLine() should give 1
// * PreviousLine() should give 5
func (result *Result) PreviousLine() (int, error) {
	if result.Lines == nil {
		return -1, nohitsError()
	}
	result.index--
	if result.index < 0 {
		result.index = len(result.Lines) - 1
	}
	return result.Lines[result.index], nil
}

func nohitsError() error {
	return errors.New("Trying to iterate through the search result when there are no hits")
}
