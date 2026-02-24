package appui

import (
	"github.com/moncho/dry/docker"
)

const (
	//DownArrow character
	DownArrow = string('\U00002193')

	//MainScreenHeaderSize is the number of lines the header of the main screen uses
	MainScreenHeaderSize = 5
	//MainScreenFooterLength is the number of lines the footer of the main screen uses
	MainScreenFooterLength = 1
	//DefaultColumnSpacing defines the minimum space between columns
	DefaultColumnSpacing = 1
	//IDColumnWidth defines a fixed width for ID columns
	IDColumnWidth = docker.ShortLen
)
