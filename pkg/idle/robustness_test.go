package idle

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// testConfig returns a config with a short idle timeout.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := config.NewConfig()
	if err := cfg.SetIdleTimeout("1s"); err != nil {
		t.Fatalf("SetIdleTimeout() error = %v", err)
	}
	return cfg
}

// TestReadXprintidleTimesOut covers the hang case from #49: xprintidle talks to
// an X server over a socket, and a blocked Output() would park the caller
// forever. The startup probe runs synchronously inside Start, so a hang there
// stops the daemon reaching its event loop while systemd still sees it alive.
func TestReadXprintidleTimesOut(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	dir := t.TempDir()
	hang := filepath.Join(dir, "hang")
	// exec replaces the shell rather than forking, so the timeout's kill
	// reaches the process actually holding the output pipe.
	if err := os.WriteFile(hang, []byte("#!/bin/sh\nexec sleep 300\n"), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}

	done := make(chan struct{})
	var err error
	go func() {
		defer close(done)
		_, err = readXprintidle(hang)
	}()

	select {
	case <-done:
		if err == nil {
			t.Error("readXprintidle on a hanging process returned no error, want a timeout")
		} else {
			t.Logf("timed out as expected: %v", err)
		}
	case <-time.After(xprintidleTimeout + 3*time.Second):
		t.Fatalf("readXprintidle blocked for more than %v; the timeout is not being applied", xprintidleTimeout+3*time.Second)
	}
}

// TestReadXprintidleStillParses guards the timeout change against breaking the
// normal path.
func TestReadXprintidleStillParses(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	dir := t.TempDir()
	stub := filepath.Join(dir, "xprintidle")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho 1234\n"), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}

	got, err := readXprintidle(stub)
	if err != nil {
		t.Fatalf("readXprintidle: %v", err)
	}
	if want := 1234 * time.Millisecond; got != want {
		t.Errorf("readXprintidle = %v, want %v", got, want)
	}
}

// TestSignalResumeCoalesces covers the fork storm from #49. Every input event
// used to produce a resume event, and each one the daemon consumes may shell
// out, so ordinary mouse movement turned into a stream of subprocess spawns.
func TestSignalResumeCoalesces(t *testing.T) {
	d := NewIdleDetector(testConfig(t))

	// A burst well inside the coalescing window must produce one event, not
	// one per signal.
	for i := 0; i < 200; i++ {
		d.signalResume()
	}

	select {
	case <-d.resumeChan:
	default:
		t.Fatal("no resume event delivered for a burst of signals")
	}

	// Drain whatever else is buffered, then confirm the throttle holds.
	for {
		select {
		case <-d.resumeChan:
			continue
		default:
		}
		break
	}

	time.Sleep(resumeCoalesceWindow / 4)
	for i := 0; i < 200; i++ {
		d.signalResume()
	}
	select {
	case <-d.resumeChan:
		t.Error("resume events were delivered inside the coalescing window")
	default:
	}
}

// TestSignalResumeNeverBlocks: the channel has a small buffer and the
// detector must not stall input monitoring when the consumer is slow.
func TestSignalResumeNeverBlocks(t *testing.T) {
	d := NewIdleDetector(testConfig(t))

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Far more signals than the channel can hold, with no consumer.
		for i := 0; i < 10000; i++ {
			d.signalResume()
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("signalResume blocked when the channel was full")
	}
}

// TestHasActivitySourceDefaultsFalse: a detector that has not started has
// nothing watching, and the caller must treat that as "do not arm a timer".
func TestHasActivitySourceDefaultsFalse(t *testing.T) {
	d := NewIdleDetector(testConfig(t))
	if d.HasActivitySource() {
		t.Error("HasActivitySource() = true before Start; nothing is monitoring yet")
	}
}

// TestMarkHasActivitySticks confirms the flag is set once a monitor starts.
func TestMarkHasActivitySticks(t *testing.T) {
	d := NewIdleDetector(testConfig(t))
	d.markHasActivity()
	if !d.HasActivitySource() {
		t.Error("HasActivitySource() = false after markHasActivity()")
	}
}

// TestNativeIdleSourceCanBeCleared supports the re-arm path: once a Wayland
// event loop dies, the source must stop being reported as authoritative.
func TestNativeIdleSourceCanBeCleared(t *testing.T) {
	d := NewIdleDetector(testConfig(t))
	d.setNativeIdleSource(SourceWaylandIdleNotify)
	if got := d.NativeIdleSource(); got != SourceWaylandIdleNotify {
		t.Fatalf("NativeIdleSource() = %q, want %q", got, SourceWaylandIdleNotify)
	}
	d.clearNativeIdleSource()
	if got := d.NativeIdleSource(); got != "" {
		t.Errorf("NativeIdleSource() = %q after clear, want empty", got)
	}
}

// TestStartX11MonitorDeclinesWaylandSession covers the false-idle half of #48.
// Claiming the X11 source in a Wayland session makes the daemon trust an X
// clock that never sees native Wayland input, so it would report idle while
// the user types.
func TestStartX11MonitorDeclinesWaylandSession(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	dir := t.TempDir()
	stub := filepath.Join(dir, "xprintidle")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho 999999\n"), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("DISPLAY", "")

	d := NewIdleDetector(testConfig(t))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d.startX11Monitor(ctx)

	if got := d.NativeIdleSource(); got != "" {
		t.Errorf("NativeIdleSource() = %q in a Wayland session, want empty — the X idle clock does not observe native Wayland input", got)
	}
}

// TestStartX11MonitorClaimsPureXSession is the counterpart: on a real X
// session the source must still be claimed, so existing setups keep working.
func TestStartX11MonitorClaimsPureXSession(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	dir := t.TempDir()
	stub := filepath.Join(dir, "xprintidle")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho 0\n"), 0o755); err != nil {
		t.Fatalf("writing stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", ":0")

	d := NewIdleDetector(testConfig(t))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d.startX11Monitor(ctx)

	if got := d.NativeIdleSource(); got != SourceX11Xprintidle {
		t.Errorf("NativeIdleSource() = %q on an X11 session, want %q", got, SourceX11Xprintidle)
	}
	if !d.HasActivitySource() {
		t.Error("HasActivitySource() = false on an X11 session; the poller reports activity")
	}
}
