package appui

import (
	"context"

	"github.com/docker/docker/api/types/events"
	"github.com/moncho/dry/docker"
	"github.com/moncho/dry/ui/termui"
)

//RegisterWidget registers the given widget for updates from the given source
func RegisterWidget(source docker.SourceType, w termui.Widget) {
	docker.GlobalRegistry.Register(
		source,
		func(ctx context.Context, message events.Message) error {
			return w.Unmount()
		})
}
