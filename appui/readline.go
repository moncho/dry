package appui

import "github.com/chzyer/readline"

//ReadLine reads input
func ReadLine(prompt string) (string, error) {
	rl, err := readline.NewEx(&readline.Config{
		UniqueEditLine:         true,
		VimMode:                false,
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		return "", err
	}
	defer rl.Close()
	rl.SetPrompt(prompt)
	input, err := rl.Readline()
	if err == nil {
		return input, nil
	}

	return "", err
}
