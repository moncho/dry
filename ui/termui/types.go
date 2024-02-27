package termui

import gizak "github.com/gizak/termui"

// SizableBufferer is a termui.Bufferer with dimensions and position
type SizableBufferer interface {
	gizak.Bufferer
	GetHeight() int
	SetWidth(int)
	SetX(int)
	SetY(int)
}
