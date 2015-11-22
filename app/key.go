package app

import "github.com/moncho/dry/ui"

//KeyPressEvent maps a key to an app action
type KeyPressEvent struct {
	Key    ui.Key
	Action func(dry Dry)
}
