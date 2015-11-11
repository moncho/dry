package ui

import "fmt"

//Renderable is a mark for anything worth rendering as text
type Renderer interface {
	Render() string
}

//
type stringRenderer string

func (s stringRenderer) Render() string {
	return fmt.Sprint(s)
}
