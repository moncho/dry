package docker

import (
	"context"

	"github.com/docker/docker/api/types/events"
)

// EventCallback defines a callback function for messages
type EventCallback func(ctx context.Context, event events.Message)

// streamEvents sends incoming events to the provided channel.
func streamEvents(out chan<- events.Message) EventCallback {
	return func(ctx context.Context, event events.Message) {
		select {
		case <-ctx.Done():
		case out <- event:
		}
	}
}

func logEvents(log *EventLog) EventCallback {
	return func(ctx context.Context, event events.Message) {
		if log == nil {
			return
		}
		log.Push(&event)
	}
}

func handleEvent(
	ctx context.Context,
	event events.Message,
	processors ...EventCallback) {

	for _, ep := range processors {
		ep(ctx, event)
	}
}
