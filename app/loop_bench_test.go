package app

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkRefreshScreenNonBlocking measures how fast refreshScreen returns
// The new implementation should return immediately (non-blocking)
func BenchmarkRefreshScreenNonBlocking(b *testing.B) {
	rc := &RenderContext{
		renderChan: make(chan struct{}, 1),
	}

	// Drain the channel in background to simulate render loop
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-rc.renderChan:
			case <-done:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rc.refreshScreen()
	}
	b.StopTimer()

	close(done)
}

// BenchmarkRefreshScreenBlocking simulates the OLD blocking behavior for comparison
func BenchmarkRefreshScreenBlocking(b *testing.B) {
	renderChan := make(chan struct{}) // Unbuffered - will block

	// Simulate slow consumer
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-renderChan:
				time.Sleep(1 * time.Millisecond) // Simulate render time
			case <-done:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderChan <- struct{}{} // This blocks until consumed
	}
	b.StopTimer()

	close(done)
}

// TestCoalescingBehavior verifies that rapid refresh calls are coalesced
func TestCoalescingBehavior(t *testing.T) {
	rc := &RenderContext{
		renderChan: make(chan struct{}, 1),
	}

	var renderCount atomic.Int32
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Start consumer that counts renders
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-rc.renderChan:
				renderCount.Add(1)
				time.Sleep(10 * time.Millisecond) // Simulate render time
			case <-done:
				return
			}
		}
	}()

	// Fire 100 rapid refresh calls
	for range 100 {
		rc.refreshScreen()
	}

	// Wait for renders to complete
	time.Sleep(100 * time.Millisecond)
	close(done)
	wg.Wait()

	count := renderCount.Load()
	t.Logf("100 refresh calls resulted in %d actual renders (coalescing ratio: %.1fx)", count, 100.0/float64(count))

	// Should be significantly less than 100 due to coalescing
	if count > 20 {
		t.Errorf("Expected significant coalescing, but got %d renders for 100 calls", count)
	}
}

// BenchmarkCoalescingThroughput measures throughput with coalescing
func BenchmarkCoalescingThroughput(b *testing.B) {
	rc := &RenderContext{
		renderChan: make(chan struct{}, 1),
	}

	var renderCount atomic.Int64
	done := make(chan struct{})

	// Consumer with simulated render time
	go func() {
		for {
			select {
			case <-rc.renderChan:
				renderCount.Add(1)
				time.Sleep(5 * time.Millisecond) // 5ms render time
			case <-done:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rc.refreshScreen()
	}
	b.StopTimer()

	close(done)

	renders := renderCount.Load()
	if renders > 0 {
		b.ReportMetric(float64(b.N)/float64(renders), "coalesce_ratio")
	}
}

// BenchmarkUnbufferedChannel simulates OLD behavior - blocking on each send
func BenchmarkUnbufferedChannel(b *testing.B) {
	ch := make(chan struct{})
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ch:
				// Fast consumer
			case <-done:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch <- struct{}{}
	}
	b.StopTimer()

	close(done)
}

// BenchmarkBufferedChannelNonBlocking simulates NEW behavior
func BenchmarkBufferedChannelNonBlocking(b *testing.B) {
	ch := make(chan struct{}, 1)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ch:
				// Fast consumer
			case <-done:
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	b.StopTimer()

	close(done)
}
