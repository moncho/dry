package docker

import (
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/events"
)

func TestEventLogCreation(t *testing.T) {
	eventLog := EventLog{}
	eventLog.Init(5)
	if eventLog.Capacity() != 5 || eventLog.Count() != 0 {
		t.Errorf("Event log did not intialize correctly: %d, %d", eventLog.Capacity(), eventLog.Count())
	} else if eventLog.head != 0 || eventLog.tail != 0 {
		t.Errorf("Event log state is: head %d, tail %d", eventLog.head, eventLog.tail)
	}
}

func TestEventLog(t *testing.T) {
	eventLog := EventLog{}
	eventLog.Init(5)
	eventLog.Push(&events.Message{Action: "1"})
	eventLog.Push(&events.Message{Action: "2"})
	eventLog.Push(&events.Message{Action: "3"})
	eventLog.Push(&events.Message{Action: "4"})
	if eventLog.Count() != 4 {
		t.Errorf("Event log is reporting a wrong number of events: %d", eventLog.Count())
	}
	eventLog.Push(&events.Message{Action: "5"})
	eventLog.Push(&events.Message{Action: "6"})
	if eventLog.Count() != 5 {
		t.Errorf("Event log is reporting a wrong number of events: %d", eventLog.Count())
	}

	if eventLog.Peek().Action != "6" {
		t.Errorf("Last message is not correct: %s", eventLog.Peek().Action)
	}
	for i, event := range eventLog.Events() {
		if string(event.Action) != strconv.Itoa(i+2) {
			t.Errorf("Last message is not correct: %s", event.Action)
		}
	}
	if eventLog.Capacity() != 5 || eventLog.Count() != 5 {
		t.Errorf("Event log is reporting a wrong number of elements: %d, %d", eventLog.Capacity(), eventLog.Count())
	}
}

func TestEventLogCapacity(t *testing.T) {
	eventLog := EventLog{}
	eventLog.Init(5)
	for i := 0; i < 100; i++ {
		eventLog.Push(&events.Message{Action: events.Action(strconv.Itoa(i))})
	}
	if eventLog.Capacity() != 5 || eventLog.Count() != 5 {
		t.Errorf("Event log is reporting a wrong number of elements: %d, %d", eventLog.Capacity(), eventLog.Count())
	}

	if eventLog.Peek().Action != "99" {
		t.Errorf("Last message is not correct: %s", eventLog.Peek().Action)
	}
}

func BenchmarkEventLog(b *testing.B) {
	eventLog := NewEventLog()
	for i := 0; i < b.N; i++ {
		eventLog.Push(&events.Message{Action: events.Action(strconv.Itoa(i))})
	}
}
