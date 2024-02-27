package docker

import (
	"sync"

	"github.com/docker/docker/api/types/events"
)

const (
	//DefaultCapacity of a new EventLog.
	DefaultCapacity = 50
)

// EventLog keeps track of docker events. It has a limited capacity and
// behaves as a Circular Buffer - adding a new event removes the oldest one if
// the buffer is at its max capacity.
type EventLog struct {
	head     int // the most recent value written
	tail     int // the least recent value written
	capacity int
	messages []*events.Message
	sync.RWMutex
}

// NewEventLog creates an event log with the default capacity
func NewEventLog() *EventLog {
	log := &EventLog{}
	log.Init(DefaultCapacity)

	return log
}

// Capacity returns the capacity of the event log.
func (el *EventLog) Capacity() int {
	return el.capacity
}

// Count returns the number of events in the buffer
func (el *EventLog) Count() int {
	return el.tail - el.head
}

// Events returns a copy of the event buffer
func (el *EventLog) Events() []events.Message {
	el.RLock()
	defer el.RUnlock()
	if el.Count() == 0 {
		return nil
	}
	messages := make([]events.Message, el.Count())
	for i, message := range el.messages[el.head:el.tail] {
		messages[i] = *message
	}
	return messages
}

// Init sets the log in a working state. Must be
// called before doing any other operation
func (el *EventLog) Init(capacity int) {
	el.messages = make([]*events.Message, capacity, capacity*2)
	el.capacity = capacity
}

// Peek the latest event added
func (el *EventLog) Peek() *events.Message {
	el.RLock()
	defer el.RUnlock()
	return el.messages[el.tail-1]
}

// Push the given event to this log
func (el *EventLog) Push(message *events.Message) {
	el.Lock()
	defer el.Unlock()
	// if the array is full, rewind
	if el.tail == el.capacity {
		el.rewind()
	}
	el.messages[el.tail] = message
	// check if the buffer is full,
	// and move head pointer appropriately
	if el.tail-el.head >= el.capacity {
		el.head++
	}
	el.tail++
}

func (el *EventLog) rewind() {
	l := len(el.messages)
	for i := 0; i < el.capacity-1; i++ {
		el.messages[i] = el.messages[l-el.capacity+i+1]
	}
	el.head, el.tail = 0, el.capacity-1
}
