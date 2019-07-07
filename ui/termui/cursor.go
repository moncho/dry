package termui

type cursor interface {
	HideCursor()
	ShowCursor(int, int)
}
