package appui

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"
)

var timeBetweenRefresh = 500 * time.Millisecond

//RegisterWidget registers the given widget for updates from the given source
func RegisterWidget(source docker.SourceType, w termui.Widget) {
	last := time.Now()
	docker.GlobalRegistry.Register(
		source,
		func(ctx context.Context, message events.Message) error {
			if time.Now().Sub(last) > timeBetweenRefresh {
				last = time.Now()

				return w.Unmount()
			}
			return nil
		})
}
