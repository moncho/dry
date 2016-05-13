package docker

import (
	"encoding/json"
	"errors"
	"io"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/types/events"
)

// streamEvents sends incoming events to the provided channel.
func streamEvents(out chan<- events.Message) eventProcessor {
	return func(event events.Message, err error) error {
		if err != nil {
			return err
		}
		out <- event
		return nil
	}
}

func logEvents(log *EventLog) eventProcessor {
	return func(event events.Message, err error) error {
		if err != nil {
			return err
		} else if log == nil {
			return errors.New("No logger given")
		}
		log.Push(&event)
		return nil
	}
}

type eventProcessor func(event events.Message, err error) error

func decodeEvents(
	ctx context.Context,
	input io.Reader,
	processors ...eventProcessor) error {
	dec := json.NewDecoder(input)
	for {
		var event events.Message
		err := dec.Decode(&event)
		if err != nil && err == io.EOF {
			break
		}
		for _, ep := range processors {
			if procErr := ep(event, err); procErr != nil {
				return procErr
			}
		}
	}
	return nil
}
