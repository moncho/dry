package ui

import "fmt"

//Renderer is a mark for anything worth rendering as text
type Renderer interface {
	Render() string
}

//StringRenderer adapts a string to the render interface.
type StringRenderer string

//Render a string
func (s StringRenderer) Render() string {
	return fmt.Sprint(s)
}
