package idle

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// Stop must not free the display while the event loop may still be using it:
// it waits for the loop to return before cleanup (audit sysc-2367).
func TestWaylandCGOStopWaitsForTheEventLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &WaylandCGODetector{ctx: ctx, cancel: cancel}
	done := make(chan struct{})
	w.loopDone = done

	var loopReturned atomic.Bool
	go func() {
		<-ctx.Done()
		// Still inside poll or dispatch for a moment after the cancel.
		time.Sleep(50 * time.Millisecond)
		loopReturned.Store(true)
		close(done)
	}()

	w.Stop()
	if !loopReturned.Load() {
		t.Fatal("Stop returned while the event loop was still running")
	}

	// A second Stop, or one before Start, does not wait on anything.
	stopped := make(chan struct{})
	go func() { w.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("a repeated Stop blocked")
	}
}
