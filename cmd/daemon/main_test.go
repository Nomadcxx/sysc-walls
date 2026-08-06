package main

import (
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// newTestDaemon builds a daemon with a short idle timeout for timer tests.
// The config parser only accepts whole seconds, so 1s is the shortest usable timeout.
func newTestDaemon(t *testing.T, timeout string) *Daemon {
	t.Helper()

	cfg := config.NewConfig()
	if err := cfg.SetIdleTimeout(timeout); err != nil {
		t.Fatalf("SetIdleTimeout(%q) error = %v", timeout, err)
	}

	d := NewDaemon(cfg)
	t.Cleanup(d.cancel)

	return d
}

// TestConfigureFallbackTimer_NativeDetection verifies that the wall-clock fallback
// timer stays disarmed when the compositor provides native idle notifications.
// Regression test for issue #24: the timer fired every timeout of wall-clock time
// regardless of user activity, launching the screensaver mid-use.
func TestConfigureFallbackTimer_NativeDetection(t *testing.T) {
	d := newTestDaemon(t, "1s")

	d.configureFallbackTimer(true)

	if d.useFallbackTimer.Load() {
		t.Error("useFallbackTimer = true, want false when native idle detection is active")
	}

	select {
	case <-d.idleTimer.C:
		t.Fatal("fallback timer fired while native idle detection was active")
	case <-time.After(1500 * time.Millisecond):
	}
}

// TestResetIdleTimer_NativeDetection verifies that activity handling does not
// re-arm the fallback timer once native idle detection is in charge.
func TestResetIdleTimer_NativeDetection(t *testing.T) {
	d := newTestDaemon(t, "1s")

	d.configureFallbackTimer(true)
	d.resetIdleTimer()

	select {
	case <-d.idleTimer.C:
		t.Fatal("resetIdleTimer() re-armed the fallback timer under native idle detection")
	case <-time.After(1500 * time.Millisecond):
	}
}

// TestConfigureFallbackTimer_NoNativeDetection verifies the fallback timer still
// arms when no native idle source is available, so those setups keep working.
func TestConfigureFallbackTimer_NoNativeDetection(t *testing.T) {
	d := newTestDaemon(t, "1s")

	d.configureFallbackTimer(false)

	if !d.useFallbackTimer.Load() {
		t.Error("useFallbackTimer = false, want true when no native idle detection is available")
	}

	select {
	case <-d.idleTimer.C:
	case <-time.After(3 * time.Second):
		t.Fatal("fallback timer never fired without native idle detection")
	}
}

// TestNewDaemon_TimerDisarmedUntilConfigured verifies the timer is not running
// before Run() decides whether a fallback is needed.
func TestNewDaemon_TimerDisarmedUntilConfigured(t *testing.T) {
	d := newTestDaemon(t, "1s")

	select {
	case <-d.idleTimer.C:
		t.Fatal("idle timer fired before configureFallbackTimer() was called")
	case <-time.After(1500 * time.Millisecond):
	}
}
