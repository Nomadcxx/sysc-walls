package main

import (
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

// newTestDaemon builds a daemon with a short idle timeout for timer tests.
// The config parser only accepts whole seconds, so 1s is the shortest usable timeout.
func newTestDaemon(t *testing.T) *Daemon {
	t.Helper()

	cfg := config.NewConfig()
	if err := cfg.SetIdleTimeout("1s"); err != nil {
		t.Fatalf("SetIdleTimeout() error = %v", err)
	}

	d := NewDaemon(cfg)
	t.Cleanup(d.cancel)

	return d
}

// assertTimerQuiet fails if the fallback timer fires within a window generous
// enough that an armed 1s timer would have been seen.
func assertTimerQuiet(t *testing.T, d *Daemon, msg string) {
	t.Helper()

	select {
	case <-d.idleTimer.C:
		t.Fatal(msg)
	case <-time.After(1500 * time.Millisecond):
	}
}

// TestConfigureFallbackTimer_NativeSource verifies that a native idle source
// disarms the wall-clock fallback timer, including one already running.
//
// Regression test for issue #24: the timer fired every timeout of wall-clock
// time regardless of user activity, launching the screensaver mid-use.
func TestConfigureFallbackTimer_NativeSource(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t)

	// Arm it first, so this exercises the teardown rather than a timer that
	// was never running in the first place.
	d.useFallbackTimer.Store(true)
	d.resetIdleTimer()

	d.configureFallbackTimer(idle.SourceWaylandIdleNotify)

	if d.useFallbackTimer.Load() {
		t.Error("useFallbackTimer = true, want false when a native idle source is active")
	}

	assertTimerQuiet(t, d, "fallback timer fired while a native idle source was active")
}

// TestResetIdleTimer_NativeSource verifies that activity handling does not
// re-arm the fallback timer once a native source is in charge.
func TestResetIdleTimer_NativeSource(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t)

	d.configureFallbackTimer(idle.SourceX11Xprintidle)
	d.resetIdleTimer()

	assertTimerQuiet(t, d, "resetIdleTimer() re-armed the fallback timer under a native idle source")
}

// TestConfigureFallbackTimer_NoNativeSource verifies the fallback timer still
// arms when no native idle source is available, so those setups keep working.
func TestConfigureFallbackTimer_NoNativeSource(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t)

	start := time.Now()
	d.configureFallbackTimer("")

	if !d.useFallbackTimer.Load() {
		t.Error("useFallbackTimer = false, want true when no native idle source is available")
	}

	select {
	case <-d.idleTimer.C:
	case <-time.After(3 * time.Second):
		t.Fatal("fallback timer never fired without a native idle source")
	}

	// The timer must run for the configured timeout, not some other duration
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond || elapsed > 2*time.Second {
		t.Errorf("fallback timer fired after %v, want ~%v", elapsed, d.config.GetIdleTimeout())
	}
}

// TestNewDaemon_TimerDisarmedUntilConfigured verifies the timer is not running
// before Run() decides whether a fallback is needed.
func TestNewDaemon_TimerDisarmedUntilConfigured(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t)

	assertTimerQuiet(t, d, "idle timer fired before configureFallbackTimer() was called")
}

// TestEventLoop_IgnoresTickWithoutFallback verifies the event loop treats
// useFallbackTimer as the authority on whether a tick may launch anything,
// so a stray tick cannot start a screensaver under native idle detection.
func TestEventLoop_IgnoresTickWithoutFallback(t *testing.T) {
	t.Parallel()

	d := newTestDaemon(t)
	d.configureFallbackTimer(idle.SourceWaylandIdleNotify)

	launched := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-d.idleTimer.C:
				if !d.useFallbackTimer.Load() {
					continue
				}
				launched <- struct{}{}
			}
		}
	}()

	// Force a tick the way a timer that was already firing when it got
	// disabled would deliver one.
	d.idleTimer.Reset(10 * time.Millisecond)

	select {
	case <-launched:
		t.Fatal("event loop acted on a fallback tick while native idle detection was active")
	case <-time.After(500 * time.Millisecond):
	}
}
