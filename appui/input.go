package appui

// Input reads input
func Input(done chan<- struct{}) {
	done <- struct{}{}
}
