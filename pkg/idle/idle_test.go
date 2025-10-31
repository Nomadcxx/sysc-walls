package idle

import (
	"context"
	"os"
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

// TestTrimWhitespace tests whitespace trimming
func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{" hello ", "hello"},
		{"\thello\t", "hello"},
		{"\n hello \n", "hello"},
		{"  multiple   spaces  ", "multiple   spaces"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("trimWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseInt tests integer parsing
func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"0", 0},
		{"123", 123},
		{"  456  ", 456},
		{"-10", -10},
		{"invalid", 0},
		{"", 0},
		{"12.34", 0}, // strconv.Atoi fails on decimals
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
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


