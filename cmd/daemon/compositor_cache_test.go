package main

import (
	"fmt"
	"testing"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
)

// TestGetCompositorCachesDetection verifies the compositor is detected once
// and reused across calls, so per-compositor state survives screensaver
// relaunches (#38).
func TestGetCompositorCachesDetection(t *testing.T) {
	d := &Daemon{}
	calls := 0
	d.detectCompositor = func() (compositor.Compositor, error) {
		calls++
		return compositor.NewNiriCompositor(), nil
	}

	first, err := d.getCompositor()
	if err != nil {
		t.Fatalf("getCompositor() error = %v", err)
	}
	second, err := d.getCompositor()
	if err != nil {
		t.Fatalf("getCompositor() error = %v", err)
	}

	if calls != 1 {
		t.Errorf("detection called %d times, want 1", calls)
	}
	if first != second {
		t.Error("getCompositor() returned different instances across calls")
	}
}

// TestGetCompositorRetriesAfterDetectionFailure verifies a failed detection
// is not cached, so a later launch can retry.
func TestGetCompositorRetriesAfterDetectionFailure(t *testing.T) {
	d := &Daemon{}
	calls := 0
	want := compositor.NewNiriCompositor()
	d.detectCompositor = func() (compositor.Compositor, error) {
		calls++
		if calls == 1 {
			return nil, fmt.Errorf("not running on Wayland")
		}
		return want, nil
	}

	if _, err := d.getCompositor(); err == nil {
		t.Fatal("getCompositor() error = nil, want detection failure")
	}
	got, err := d.getCompositor()
	if err != nil {
		t.Fatalf("getCompositor() retry error = %v", err)
	}
	if got != want {
		t.Error("getCompositor() did not return the compositor from the retried detection")
	}
	if calls != 2 {
		t.Errorf("detection called %d times, want 2", calls)
	}
}
