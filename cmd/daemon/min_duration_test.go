package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// TestOnActivityWaitsForMinimumDuration is the regression test for issue #93:
// a resume event used to stop the screensaver immediately, no matter what
// idle.min_duration said.
func TestOnActivityWaitsForMinimumDuration(t *testing.T) {
	bin := t.TempDir()
	kitty := filepath.Join(bin, "kitty")
	if err := os.WriteFile(kitty, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatalf("writing kitty shim: %v", err)
	}
	display := filepath.Join(bin, "sysc-walls-display")
	if err := os.WriteFile(display, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("writing display shim: %v", err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	cfg := config.NewConfig()
	configPath := filepath.Join(t.TempDir(), "daemon.conf")
	if err := os.WriteFile(configPath, []byte("[idle]\nmin_duration = 1s\n"), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	if err := cfg.LoadFromFile(configPath); err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.SetTerminalKitty(true)

	d := NewDaemon(cfg)
	t.Cleanup(d.Shutdown)
	d.setCompositor(focuslessCompositor{})

	d.LaunchScreensaver()
	if !d.systemD.IsRunning() {
		t.Fatal("screensaver did not launch")
	}

	// Measure the minimum from now, independent of how long the launch took.
	d.screensaverStarted.Store(time.Now().UnixNano())
	start := time.Now()
	d.onActivity()

	time.Sleep(200 * time.Millisecond)
	if !d.systemD.IsRunning() {
		t.Fatal("resume dismissed the screensaver before idle.min_duration elapsed")
	}

	deadline := time.Now().Add(3 * time.Second)
	for d.systemD.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if d.systemD.IsRunning() {
		t.Fatal("screensaver still running after idle.min_duration elapsed")
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Fatalf("screensaver was dismissed after %v, want at least idle.min_duration (1s)", elapsed)
	}
}

// TestOnActivityStopsScreensaverAfterMinimumDuration confirms the deferred
// path is skipped when the saver has already run long enough.
func TestOnActivityStopsScreensaverAfterMinimumDuration(t *testing.T) {
	bin := t.TempDir()
	kitty := filepath.Join(bin, "kitty")
	if err := os.WriteFile(kitty, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatalf("writing kitty shim: %v", err)
	}
	display := filepath.Join(bin, "sysc-walls-display")
	if err := os.WriteFile(display, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("writing display shim: %v", err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	cfg := config.NewConfig()
	configPath := filepath.Join(t.TempDir(), "daemon.conf")
	if err := os.WriteFile(configPath, []byte("[idle]\nmin_duration = 1s\n"), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	if err := cfg.LoadFromFile(configPath); err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.SetTerminalKitty(true)

	d := NewDaemon(cfg)
	t.Cleanup(d.Shutdown)
	d.setCompositor(focuslessCompositor{})

	d.LaunchScreensaver()
	if !d.systemD.IsRunning() {
		t.Fatal("screensaver did not launch")
	}

	d.screensaverStarted.Store(time.Now().Add(-2 * time.Second).UnixNano())
	d.onActivity()
	if d.systemD.IsRunning() {
		t.Fatal("resume did not stop a screensaver that outlived idle.min_duration")
	}
}
