package app

import (
	"fmt"
	"strings"
)

type dockerEventsListener struct {
	dry *Dry
}

func (el *dockerEventsListener) init() {
	dry := el.dry
	go func() {
		for event := range el.dry.dockerEvents {
			//exec_ messages are sent continuously if docker is checking
			//a container's health, so they are ignored
			if strings.Contains(event.Action, "exec_") {
				continue
			}
			//top messages are sent continuously on monitor mode
			if strings.Contains(event.Action, "top") {
				continue
			}
			dry.appmessage(fmt.Sprintf("Docker daemon: %s %s", event.Action, event.ID))

			var err error
			switch event.Type {
			case "network":
				err = dry.dockerDaemon.RefreshNetworks()
				dry.dockerDaemon.SortNetworks(dry.state.sortNetworksMode)
				refreshScreen()
			}
			if err != nil {
				dry.appmessage("There was an error refreshing: " + err.Error())
			}
		}
	}()
}
