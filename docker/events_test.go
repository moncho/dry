package docker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/events"
)

func TestHandleEvent_RunsCallbacksSynchronously(t *testing.T) {
	// Verify that handleEvent runs all callbacks before returning,
	// so there are no fire-and-forget goroutines that could race.
	var order []int
	var mu sync.Mutex

	cb := func(n int) EventCallback {
		return func(ctx context.Context, event events.Message) error {
			mu.Lock()
			order = append(order, n)
			mu.Unlock()
			return nil
		}
	}

	handleEvent(context.Background(), events.Message{Action: "test"}, cb(1), cb(2), cb(3))

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("expected 3 callbacks executed, got %d", len(order))
	}
	// Since callbacks run synchronously and sequentially, order must be 1,2,3
	for i, v := range order {
		if v != i+1 {
			t.Fatalf("expected callback %d at position %d, got %d", i+1, i, v)
		}
	}
}

func TestStreamEvents_SendsEvent(t *testing.T) {
	ch := make(chan events.Message, 1)
	cb := streamEvents(ch)

	msg := events.Message{Action: "start"}
	err := cb(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case got := <-ch:
		if got.Action != "start" {
			t.Fatalf("expected action 'start', got %q", got.Action)
		}
	default:
		t.Fatal("expected event on channel")
	}
}

func TestStreamEvents_RespectsContextCancellation(t *testing.T) {
	// Unbuffered channel — send would block if context isn't checked.
	ch := make(chan events.Message)
	cb := streamEvents(ch)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should return without blocking (no goroutine to drain ch).
	done := make(chan struct{})
	go func() {
		cb(ctx, events.Message{Action: "should-not-send"})
		close(done)
	}()

	select {
	case <-done:
		// good — returned promptly
	case <-time.After(time.Second):
		t.Fatal("streamEvents blocked despite cancelled context")
	}

	// Channel should be empty
	select {
	case msg := <-ch:
		t.Fatalf("expected no event, got %v", msg)
	default:
	}
}

func TestStreamEvents_NoPanicAfterChannelDrain(t *testing.T) {
	// Simulate the coordinated shutdown: context cancelled, channel drained.
	// streamEvents must not panic.
	ch := make(chan events.Message, 1)
	ctx, cancel := context.WithCancel(context.Background())

	cb := streamEvents(ch)

	// Send one event successfully
	cb(ctx, events.Message{Action: "first"})
	<-ch // drain

	// Now cancel context — subsequent calls should be no-ops
	cancel()
	cb(ctx, events.Message{Action: "after-cancel"})

	select {
	case msg := <-ch:
		t.Fatalf("expected no event after cancel, got %v", msg)
	default:
	}
}

func TestLogEvents_NilLogger(t *testing.T) {
	cb := logEvents(nil)
	err := cb(context.Background(), events.Message{})
	if err == nil {
		t.Fatal("expected error for nil logger")
	}
}

func TestLogEvents_PushesEvent(t *testing.T) {
	log := &EventLog{}
	log.Init(10)

	cb := logEvents(log)
	err := cb(context.Background(), events.Message{Action: "create"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.Count() != 1 {
		t.Fatalf("expected 1 event in log, got %d", log.Count())
	}
	if log.Peek().Action != "create" {
		t.Fatalf("expected action 'create', got %q", log.Peek().Action)
	}
}

func TestCleanup_ClosesChannelAfterContextCancel(t *testing.T) {
	// Simulate the Events() cleanup pattern: cancel context, goroutines
	// stop, WaitGroup completes, channel is closed.
	// Verify no panic from send-on-closed-channel.

	ctx, cancel := context.WithCancel(context.Background())
	eventC := make(chan events.Message)

	var wg sync.WaitGroup

	// Simulate a local event goroutine that streams events
	localEvents := make(chan events.Message)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-localEvents:
				handleEvent(ctx, event, streamEvents(eventC))
			case <-ctx.Done():
				return
			}
		}
	}()

	// Simulate a swarm event goroutine
	swarmEvents := make(chan events.Message)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-swarmEvents:
				handleEvent(ctx, event, streamEvents(eventC))
			case <-ctx.Done():
				return
			}
		}
	}()

	// Cleanup goroutine (mirrors daemon.go)
	go func() {
		<-ctx.Done()
		wg.Wait()
		close(eventC)
	}()

	// Send some events
	localEvents <- events.Message{Action: "local-1"}
	got := <-eventC
	if got.Action != "local-1" {
		t.Fatalf("expected 'local-1', got %q", got.Action)
	}

	swarmEvents <- events.Message{Action: "swarm-1"}
	got = <-eventC
	if got.Action != "swarm-1" {
		t.Fatalf("expected 'swarm-1', got %q", got.Action)
	}

	// Cancel context to trigger shutdown
	cancel()

	// eventC should be closed after goroutines finish
	for range eventC {
		// drain any remaining
	}
	// If we reach here without panic, the test passes.
}
