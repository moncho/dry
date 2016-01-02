package ui

import "fmt"

//Renderer is a mark for anything worth rendering as text
type Renderer interface {
	Render() string
}

//stringRenderer adapts a string to the render interface.
type stringRenderer string

//Render a string
func (s stringRenderer) Render() string {
	return fmt.Sprint(s)
}
