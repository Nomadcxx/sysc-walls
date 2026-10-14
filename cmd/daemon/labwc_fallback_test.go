// labwc_fallback_test.go - Focus-unavailable fallback for issue #16
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
)

// focuslessCompositor mimics labwc: outputs can be enumerated through
// wlr-randr, but nothing outside the compositor can query or move keyboard
// focus, because labwc has no IPC at all.
type focuslessCompositor struct{}

func (focuslessCompositor) Name() string { return "labwc" }

func (focuslessCompositor) ListOutputs() ([]compositor.Output, error) {
	return []compositor.Output{
		{Name: "HDMI-A-1", Width: 1920, Height: 1080},
		{Name: "DP-3", Width: 2560, Height: 1440},
	}, nil
}

func (focuslessCompositor) GetFocusedOutput() (string, error) {
	return "", compositor.ErrNoFocusControl
}

func (focuslessCompositor) FocusOutput(string) error { return compositor.ErrNoFocusControl }

func (focuslessCompositor) PrepareFullscreen(string) error { return nil }

func (focuslessCompositor) CleanupFullscreen(string) error { return nil }

// TestLaunchScreensaverFallsBackWhenFocusCannotMove is the regression test for
// issue #16: before labwc was known, a focus refusal simply `continue`d, so
// every output was skipped and the screens never got covered. The per-output
// loop must hand over to the single-instance launch instead.
func TestLaunchScreensaverFallsBackWhenFocusCannotMove(t *testing.T) {
	// The real command would open a terminal on this session, so put shims
	// at the front of PATH: kitty sleeps so its PID stays alive for the
	// assertion, and the display binary only has to exist for the command
	// to validate.
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

	d := newTestDaemon(t)
	d.config.SetTerminalKitty(true)
	d.setCompositor(focuslessCompositor{})
	t.Cleanup(d.StopScreensaver)

	d.LaunchScreensaver()

	if n := d.systemD.GetProcessCount(); n != 1 {
		t.Fatalf("launched %d screensaver processes, want exactly 1: a compositor without focus control must fall back to a single instance", n)
	}
}
