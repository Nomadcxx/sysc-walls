package idle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// TestNewIdleDetector tests idle detector creation
func TestNewIdleDetector(t *testing.T) {
	cfg := config.NewConfig()
	detector := NewIdleDetector(cfg)

	if detector == nil {
		t.Fatal("NewIdleDetector() returned nil")
	}

	if detector.config != cfg {
		t.Error("Detector config doesn't match provided config")
	}

	if detector.idleTimeout != cfg.GetIdleTimeout() {
		t.Errorf("Detector timeout = %v, want %v", detector.idleTimeout, cfg.GetIdleTimeout())
	}

	if detector.idleChan == nil {
		t.Error("idleChan is nil")
	}

	if detector.resumeChan == nil {
		t.Error("resumeChan is nil")
	}
}

// TestIdleDetector_Events tests event channel access
func TestIdleDetector_Events(t *testing.T) {
	cfg := config.NewConfig()
	detector := NewIdleDetector(cfg)

	events := detector.Events()

	if events == nil {
		t.Fatal("Events() returned nil")
	}

	if events.Idle != detector.idleChan {
		t.Error("Events.Idle doesn't match internal channel")
	}

	if events.Resume != detector.resumeChan {
		t.Error("Events.Resume doesn't match internal channel")
	}
}

// TestIdleDetector_MarkActive tests marking system as active
func TestIdleDetector_MarkActive(t *testing.T) {
	cfg := config.NewConfig()
	detector := NewIdleDetector(cfg)

	// Record time before marking active
	beforeTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	detector.MarkActive()

	// lastActive should be updated
	if detector.lastActive.Before(beforeTime) {
		t.Error("MarkActive() didn't update lastActive time")
	}

	// Resume event should be fired
	select {
	case <-detector.resumeChan:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("MarkActive() didn't fire resume event")
	}
}

// TestIdleDetector_MarkActiveMultiple tests multiple MarkActive calls
func TestIdleDetector_MarkActiveMultiple(t *testing.T) {
	cfg := config.NewConfig()
	detector := NewIdleDetector(cfg)

	// Call MarkActive multiple times rapidly
	for i := 0; i < 5; i++ {
		detector.MarkActive()
		time.Sleep(1 * time.Millisecond)
	}

	// Should not panic or block
	// Drain resume channel
	timeout := time.After(50 * time.Millisecond)
	for {
		select {
		case <-detector.resumeChan:
			// Drain
		case <-timeout:
			return // Success
		}
	}
}

// TestDetectDisplayServer tests display server detection
func TestDetectDisplayServer(t *testing.T) {
	// Save original env vars
	originalWayland := os.Getenv("WAYLAND_DISPLAY")
	originalDisplay := os.Getenv("DISPLAY")
	defer func() {
		os.Setenv("WAYLAND_DISPLAY", originalWayland)
		os.Setenv("DISPLAY", originalDisplay)
	}()

	tests := []struct {
		name            string
		waylandDisplay  string
		x11Display      string
		expectedServer  string
	}{
		{
			name:           "Wayland",
			waylandDisplay: "wayland-0",
			x11Display:     "",
			expectedServer: "wayland",
		},
		{
			name:           "X11",
			waylandDisplay: "",
			x11Display:     ":0",
			expectedServer: "x11",
		},
		{
			name:           "Both (Wayland priority)",
			waylandDisplay: "wayland-0",
			x11Display:     ":0",
			expectedServer: "wayland",
		},
		{
			name:           "Neither",
			waylandDisplay: "",
			x11Display:     "",
			expectedServer: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("WAYLAND_DISPLAY", tt.waylandDisplay)
			os.Setenv("DISPLAY", tt.x11Display)

			result := detectDisplayServer()
			if result != tt.expectedServer {
				t.Errorf("detectDisplayServer() = %s, want %s", result, tt.expectedServer)
			}
		})
	}
}

// TestHasXprintidle tests xprintidle detection
func TestHasXprintidle(t *testing.T) {
	// This test just verifies the function doesn't panic
	// Actual result depends on system setup
	result := hasXprintidle()
	t.Logf("hasXprintidle() = %v", result)
}

// TestDiscoverInputDevices tests input device discovery
func TestDiscoverInputDevices(t *testing.T) {
	// This test verifies the function doesn't panic
	// Actual devices depend on system and permissions
	devices, err := discoverInputDevices()
	
	if err != nil {
		t.Logf("discoverInputDevices() error: %v (may be expected on systems without /dev/input)", err)
	}
	
	t.Logf("Found %d input devices", len(devices))
	for _, dev := range devices {
		t.Logf("  - %s", dev)
	}
}

// TestIdleDetector_Start tests starting the detector
func TestIdleDetector_Start(t *testing.T) {
	cfg := config.NewConfig()
	detector := NewIdleDetector(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should not block
	err := detector.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	// Wait for context timeout
	<-ctx.Done()
}

// TestIdleDetector_IdleTimeout tests idle timeout behavior
func TestIdleDetector_IdleTimeout(t *testing.T) {
	// This test is tricky because the idle detector checks actual idle time
	// Not just a timer. It checks if time.Since(lastActive) > timeout
	// Since we just created the detector, lastActive was set to time.Now()
	// So we need to wait for the full timeout period
	t.Skip("Skipping idle timeout test - requires waiting for full timeout period or mocking time")
}

// TestIdleDetector_ActivityResets tests that activity resets idle timer
func TestIdleDetector_ActivityResets(t *testing.T) {
	cfg := config.NewConfig()
	cfg.SetIdleTimeout("2s")
	
	detector := NewIdleDetector(cfg)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start detector
	go detector.Start(ctx)

	// Mark active after 1 second (before timeout)
	time.Sleep(1 * time.Second)
	detector.MarkActive()

	// Drain resume event
	select {
	case <-detector.Events().Resume:
	case <-time.After(100 * time.Millisecond):
	}

	// Should NOT receive idle event immediately since we reset
	select {
	case <-detector.Events().Idle:
		t.Error("Received idle event too soon after MarkActive()")
	case <-time.After(500 * time.Millisecond):
		t.Log("Correctly did not receive idle event immediately")
	}
}



// writeFakeXprintidle puts an executable named xprintidle in a fresh directory
// and points PATH at it, so the X11 probe can be driven deterministically.
func writeFakeXprintidle(t *testing.T, script string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "xprintidle")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatalf("writing fake xprintidle: %v", err)
	}

	t.Setenv("PATH", dir)
}

// TestNativeIdleSource_BeforeStart verifies no source is claimed until Start runs.
func TestNativeIdleSource_BeforeStart(t *testing.T) {
	detector := NewIdleDetector(config.NewConfig())

	if got := detector.NativeIdleSource(); got != "" {
		t.Errorf("NativeIdleSource() = %q before Start, want \"\"", got)
	}
}

// TestNativeIdleSource_ByEnvironment verifies which environments claim a native
// idle source. Anything that reports "" must keep the daemon's wall-clock
// fallback timer armed, and anything that reports a source must be able to
// deliver idle events on its own.
func TestNativeIdleSource_ByEnvironment(t *testing.T) {
	tests := []struct {
		name string
		// xprintidle is the body of a fake xprintidle on PATH; "" means none
		xprintidle string
		display    string
		want       string
	}{
		{
			name:    "no display server",
			display: "",
			want:    "",
		},
		{
			name:    "x11 without xprintidle",
			display: ":0",
			want:    "",
		},
		{
			// Regression guard: a present but unusable xprintidle must not
			// claim the source. It reports no idle time, so claiming it would
			// disable the fallback timer and the screensaver would never launch.
			name:       "x11 with xprintidle that cannot reach a display",
			xprintidle: "echo 'Cannot open display' >&2; exit 1",
			display:    ":99",
			want:       "",
		},
		{
			name:       "x11 with xprintidle returning garbage",
			xprintidle: "echo not-a-number",
			display:    ":0",
			want:       "",
		},
		{
			name:       "x11 with working xprintidle",
			xprintidle: "echo 1234",
			display:    ":0",
			want:       SourceX11Xprintidle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xprintidle != "" {
				writeFakeXprintidle(t, tt.xprintidle)
			} else {
				t.Setenv("PATH", t.TempDir())
			}

			// Never Wayland: that path needs a live compositor
			t.Setenv("WAYLAND_DISPLAY", "")
			t.Setenv("DISPLAY", tt.display)

			detector := NewIdleDetector(config.NewConfig())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := detector.Start(ctx); err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			if got := detector.NativeIdleSource(); got != tt.want {
				t.Errorf("NativeIdleSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReadXprintidle tests the probe used to decide whether xprintidle works.
func TestReadXprintidle(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		want    time.Duration
		wantErr bool
	}{
		{name: "reports idle time", script: "echo 4500", want: 4500 * time.Millisecond},
		{name: "trims whitespace", script: "echo '  120  '", want: 120 * time.Millisecond},
		{name: "command fails", script: "exit 1", wantErr: true},
		{name: "non-numeric output", script: "echo nope", wantErr: true},
		{name: "empty output", script: "true", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "xprintidle")
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+tt.script+"\n"), 0o755); err != nil {
				t.Fatalf("writing fake xprintidle: %v", err)
			}

			got, err := readXprintidle(path)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("readXprintidle() = %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("readXprintidle() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("readXprintidle() = %v, want %v", got, tt.want)
			}
		})
	}
}
