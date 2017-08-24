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
			dry.appmessage(fmt.Sprintf("Docker daemon: %s %s", event.Action, event.ID))

			var err error
			switch event.Type {
			case "container":
				f := func(err error) {
					if err == nil {
						refreshScreen()
					} else {
						dry.appmessage("There was an error refreshing: " + err.Error())
					}
				}
				dry.dockerDaemon.Refresh(f)
			case "volume":
				err = dry.dockerDaemon.RefreshImages()
				dry.dockerDaemon.SortImages(dry.state.sortImagesMode)
				refreshScreen()
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
