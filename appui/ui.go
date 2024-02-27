package appui

import (
	"github.com/moncho/dry/docker"
)

const (
	//DownArrow character
	DownArrow = string('\U00002193')
	//DownArrowLength is the length of the DownArrow string
	DownArrowLength = len(DownArrow)
	//RightArrow character
	RightArrow = string('\U00002192')

	//MainScreenHeaderSize is the number of lines the header of the main screen uses
	MainScreenHeaderSize = 5
	//MainScreenFooterLength is the number of lines the footer of the main screen uses
	MainScreenFooterLength = 1
	//DefaultColumnSpacing defines the minimun space between columns in pixels
	DefaultColumnSpacing = 1
	//IDColumnWidth defines a fixed width for ID columns
	IDColumnWidth = docker.ShortLen
)

// CalcItemWidth calculates the width of each item for the given total width and item count
func CalcItemWidth(width, items int) int {
	spacing := DefaultColumnSpacing * items
	return (width - spacing) / items
}
