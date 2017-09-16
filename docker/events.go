package docker

import (
	"errors"

	"context"

	"github.com/docker/docker/api/types/events"
)

type EventCallback func(ctx context.Context, event events.Message) error

// streamEvents sends incoming events to the provided channel.
func streamEvents(out chan<- events.Message) EventCallback {
	return func(ctx context.Context, event events.Message) error {
		select {
		case <-ctx.Done():
		case out <- event:
		}
		return nil
	}
}

func logEvents(log *EventLog) EventCallback {
	return func(ctx context.Context, event events.Message) error {
		if log == nil {
			return errors.New("No logger given")
		}
		log.Push(&event)
		return nil
	}
}

func handleEvent(
	ctx context.Context,
	event events.Message,
	processors ...EventCallback) error {

	for _, ep := range processors {
		if procErr := ep(ctx, event); procErr != nil {
			return procErr
		}
	}

	return nil
}
