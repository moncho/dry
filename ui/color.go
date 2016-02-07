package ui

import "fmt"

//Colors
const (
	MenuBarBackgroundColor = 0x19
)

//Blue blues the given string
func Blue(text string) string {
	return fmt.Sprintf("<blue>%s</>", text)
}

//Red reddens the given string
func Red(text string) string {
	return fmt.Sprintf("<red>%s</>", text)
}

//White whites the given string
func White(text string) string {
	return fmt.Sprintf("<white>%s</>", text)
}

//Yellow yellows the given string
func Yellow(text string) string {
	return fmt.Sprintf("<yellow>%s</>", text)
}
