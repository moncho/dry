package docker

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types/events"
)

// streamEvents sends incoming events to the provided channel.
func streamEvents(out chan<- events.Message) eventProcessor {
	return func(event events.Message) error {
		out <- event
		return nil
	}
}

func logEvents(log *EventLog) eventProcessor {
	return func(event events.Message) error {
		if log == nil {
			return errors.New("No logger given")
		}
		log.Push(&event)
		return nil
	}
}

type eventProcessor func(event events.Message) error

func handleEvent(
	ctx context.Context,
	event events.Message,
	processors ...eventProcessor) error {

	for _, ep := range processors {
		if procErr := ep(event); procErr != nil {
			return procErr
		}
	}

	return nil
}
