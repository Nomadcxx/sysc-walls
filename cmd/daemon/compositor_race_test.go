package main

import (
	"sync"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
)

// racingCompositor is a compositor stub that only needs to be safe to hold;
// the point of these tests is concurrent access to the daemon's field, not
// what the compositor does.
type racingCompositor struct {
	name string
	mu   sync.Mutex
	ops  int
}

func (c *racingCompositor) Name() string { return c.name }

func (c *racingCompositor) ListOutputs() ([]compositor.Output, error) {
	return nil, nil
}
func (c *racingCompositor) GetFocusedOutput() (string, error) { return "", nil }
func (c *racingCompositor) FocusOutput(string) error          { return nil }
func (c *racingCompositor) PrepareFullscreen(string) error    { return nil }
func (c *racingCompositor) CleanupFullscreen(string) error    { return nil }

// TestStopScreensaverConcurrentWithSetCompositor reproduces the access pattern
// of the signal handler racing the event loop.
//
// SIGUSR1/SIGUSR2 reach onActivity from the signal goroutine, which calls
// StopScreensaver and reads d.compositor, while the event loop assigns it in
// LaunchScreensaver. Run this under -race to see the unsynchronised interface
// write and read.
func TestStopScreensaverConcurrentWithSetCompositor(t *testing.T) {
	d := newTestDaemon(t)

	const iterations = 200

	// Writer: the event loop assigning the detected compositor.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			d.setCompositor(&racingCompositor{name: "test"})
		}
	}()

	// Reader: the signal handler tearing the screensaver down.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			d.StopScreensaver()
		}
	}()

	wg.Wait()
}

// TestGetCompositorSeesWhatWasSet is the functional half: the accessor pair has
// to agree, not merely avoid tripping the race detector.
func TestGetCompositorSeesWhatWasSet(t *testing.T) {
	d := newTestDaemon(t)

	if got := d.getCompositor(); got != nil {
		t.Fatalf("getCompositor() = %v before any set, want nil", got)
	}

	want := &racingCompositor{name: "hyprland"}
	d.setCompositor(want)

	got := d.getCompositor()
	if got == nil {
		t.Fatal("getCompositor() = nil after setCompositor()")
	}
	if got.Name() != want.Name() {
		t.Errorf("getCompositor().Name() = %q, want %q", got.Name(), want.Name())
	}
}

// TestWaitForActivityReturnsEarlyOnResume covers the responsiveness half of the
// multi-monitor delay: a queued resume must cut the settle wait short instead
// of the sequence sitting it out.
func TestWaitForActivityReturnsEarlyOnResume(t *testing.T) {
	d := newTestDaemon(t)

	go func() {
		time.Sleep(50 * time.Millisecond)
		d.idleDet.Events().Resume <- struct{}{}
	}()

	start := time.Now()
	ok := d.waitForActivity(5 * time.Second)
	elapsed := time.Since(start)

	if ok {
		t.Error("waitForActivity() = true, want false when a resume was signalled")
	}
	if elapsed > 2*time.Second {
		t.Errorf("waitForActivity() took %v; it should return as soon as activity arrives", elapsed)
	}
}

// TestWaitForActivityHoldsWhenIdle: with no activity it must still wait the
// full period, or windows would launch before the compositor has settled.
func TestWaitForActivityHoldsWhenIdle(t *testing.T) {
	d := newTestDaemon(t)

	start := time.Now()
	if ok := d.waitForActivity(200 * time.Millisecond); !ok {
		t.Error("waitForActivity() = false with no activity, want true")
	}
	if elapsed := time.Since(start); elapsed < 150*time.Millisecond {
		t.Errorf("waitForActivity() returned after %v, want it to wait ~200ms", elapsed)
	}
}

// TestWaitForActivityReturnsOnShutdown: a cancelled daemon must not sit out a
// settle delay during shutdown.
func TestWaitForActivityReturnsOnShutdown(t *testing.T) {
	d := newTestDaemon(t)
	d.cancel()

	start := time.Now()
	if ok := d.waitForActivity(5 * time.Second); ok {
		t.Error("waitForActivity() = true after shutdown, want false")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("waitForActivity() took %v during shutdown, want an immediate return", elapsed)
	}
}

// TestActivityPendingDrainsTheEvent confirms a queued resume is consumed, so
// the event loop does not process the same activity a second time.
func TestActivityPendingDrainsTheEvent(t *testing.T) {
	d := newTestDaemon(t)

	if d.activityPending() {
		t.Error("activityPending() = true with no resume queued")
	}

	d.idleDet.Events().Resume <- struct{}{}

	if !d.activityPending() {
		t.Error("activityPending() = false with a resume queued")
	}
	if d.activityPending() {
		t.Error("activityPending() = true on the second call; the event was not consumed")
	}
}
