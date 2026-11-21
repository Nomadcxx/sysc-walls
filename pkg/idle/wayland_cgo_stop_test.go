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

	// A second Stop returns at once.
	stopped := make(chan struct{})
	go func() { w.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("a repeated Stop blocked")
	}
}

// A concurrent Stop must not skip the wait and free the display under the
// loop, and a new detector must not initialise the shared Wayland state while
// an earlier Stop is still tearing it down.
func TestWaylandCGOConcurrentStopsBothWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := &WaylandCGODetector{ctx: ctx, cancel: cancel}
	done := make(chan struct{})
	w.loopDone = done
	var loopReturned atomic.Bool
	go func() {
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		loopReturned.Store(true)
		close(done)
	}()

	var early atomic.Bool
	finished := make(chan struct{}, 3)
	for i := 0; i < 2; i++ {
		go func() {
			w.Stop()
			if !loopReturned.Load() {
				early.Store(true)
			}
			finished <- struct{}{}
		}()
	}
	// What newWaylandCGODetector does before touching the C state.
	go func() {
		time.Sleep(5 * time.Millisecond)
		cState.Lock()
		cState.Unlock() //nolint:staticcheck // waiting for the teardown is the point
		if !loopReturned.Load() {
			early.Store(true)
		}
		finished <- struct{}{}
	}()
	for i := 0; i < 3; i++ {
		<-finished
	}
	if early.Load() {
		t.Fatal("a Stop or a new detector went ahead while the loop was still running")
	}
}
