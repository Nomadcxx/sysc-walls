// labwc_test.go - Tests for the labwc compositor
package compositor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// twoHeadsOneDisabled is the shape wlr-randr --json prints: one object per
// head, an "enabled" flag, and the active mode carrying "current": true.
const twoHeadsOneDisabled = `[
  {"name": "HDMI-A-1", "description": "Samsung Electric Company 24", "make": "SAM", "model": "SyncMaster", "serial": "S3N5",
   "physical_size": {"width": 500, "height": 300}, "enabled": true,
   "modes": [
     {"width": 1920, "height": 1080, "refresh": 60.0, "preferred": true, "current": true},
     {"width": 1280, "height": 720, "refresh": 59.94, "preferred": false, "current": false}
   ],
   "position": {"x": 0, "y": 0}, "transform": "normal", "scale": 1.0, "adaptive_sync": "disabled"},
  {"name": "DP-3", "description": "Dell 27", "make": "DEL", "model": "U2720Q", "serial": "7XYZ",
   "physical_size": {"width": 600, "height": 340}, "enabled": false,
   "modes": [{"width": 2560, "height": 1440, "refresh": 60.0, "preferred": true, "current": false}]}
]`

// dualHeadOutput is what labwc prints for a single output driven by two
// heads; it must collapse to one entry, not one window per head.
const dualHeadOutput = `[
  {"name": "DP-1", "enabled": true, "modes": [{"width": 2560, "height": 1440, "current": true}]},
  {"name": "DP-1", "enabled": true, "modes": [{"width": 1920, "height": 1080, "current": true}]}
]`

func labwcWithOutput(t *testing.T, json string) *LabwcCompositor {
	t.Helper()
	l := NewLabwcCompositor()
	l.runWlrRandr = func(args ...string) ([]byte, error) {
		if len(args) != 1 || args[0] != "--json" {
			t.Errorf("wlr-randr called with %v, want [--json]", args)
		}
		return []byte(json), nil
	}
	return l
}

func TestLabwcListOutputs(t *testing.T) {
	l := labwcWithOutput(t, twoHeadsOneDisabled)

	outputs, err := l.ListOutputs()
	if err != nil {
		t.Fatalf("ListOutputs() error = %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("got %d outputs, want 1 (disabled heads must not appear)", len(outputs))
	}
	got := outputs[0]
	if got.Name != "HDMI-A-1" || got.Width != 1920 || got.Height != 1080 {
		t.Errorf("got %+v, want HDMI-A-1 at its current mode 1920x1080", got)
	}
	if got.Focused {
		t.Error("Focused = true; labwc cannot report focus")
	}
}

func TestLabwcDualHeadsCollapseToOneOutput(t *testing.T) {
	l := labwcWithOutput(t, dualHeadOutput)

	outputs, err := l.ListOutputs()
	if err != nil {
		t.Fatalf("ListOutputs() error = %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("got %d outputs, want the two DP-1 heads collapsed to 1", len(outputs))
	}
}

func TestLabwcNoEnabledOutputs(t *testing.T) {
	l := labwcWithOutput(t, `[{"name": "DP-1", "enabled": false, "modes": []}]`)

	if _, err := l.ListOutputs(); err == nil {
		t.Fatal("ListOutputs() = nil error with every head disabled, want an error")
	}
}

func TestLabwcMalformedJSON(t *testing.T) {
	l := labwcWithOutput(t, `not json`)

	if _, err := l.ListOutputs(); err == nil {
		t.Fatal("ListOutputs() = nil error on malformed JSON, want an error")
	}
}

func TestLabwcFocusIsRefused(t *testing.T) {
	l := NewLabwcCompositor()

	if err := l.FocusOutput("DP-1"); !errors.Is(err, ErrNoFocusControl) {
		t.Errorf("FocusOutput() error = %v, want ErrNoFocusControl", err)
	}
	if _, err := l.GetFocusedOutput(); !errors.Is(err, ErrNoFocusControl) {
		t.Errorf("GetFocusedOutput() error = %v, want ErrNoFocusControl", err)
	}
}

func TestLabwcFullscreenNeedsNoRules(t *testing.T) {
	l := NewLabwcCompositor()

	if err := l.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Errorf("PrepareFullscreen() error = %v, want nil like sway", err)
	}
	if err := l.CleanupFullscreen("sysc-walls-screensaver"); err != nil {
		t.Errorf("CleanupFullscreen() error = %v, want nil like sway", err)
	}
	if l.Name() != "labwc" {
		t.Errorf("Name() = %q, want labwc", l.Name())
	}
}

func TestLabwcRunWlrRandrErrorPropagates(t *testing.T) {
	l := NewLabwcCompositor()
	l.runWlrRandr = func(args ...string) ([]byte, error) {
		return nil, errors.New("wlr-randr: display not found")
	}

	_, err := l.ListOutputs()
	if err == nil || !strings.Contains(err.Error(), "wlr-randr") {
		t.Fatalf("ListOutputs() error = %v, want it to name wlr-randr", err)
	}
}

// setLabwcSessionEnv gives a temp bin holding an executable wlr-randr stub as
// the only PATH entry, so the niri, hyprctl and swaymsg probes all miss and
// only the labwc branch can fire.
func setLabwcSessionEnv(t *testing.T) {
	t.Helper()

	bin := t.TempDir()
	stub := filepath.Join(bin, "wlr-randr")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("writing wlr-randr stub: %v", err)
	}

	t.Setenv("PATH", bin)
	t.Setenv("WAYLAND_DISPLAY", "wayland-1")
	t.Setenv("LABWC_PID", "")
	t.Setenv("XDG_CURRENT_DESKTOP", "")
}

func TestDetectLabwcByPid(t *testing.T) {
	setLabwcSessionEnv(t)
	t.Setenv("LABWC_PID", "4242")

	comp, err := DetectCompositor()
	if err != nil {
		t.Fatalf("DetectCompositor() error = %v, want labwc under LABWC_PID", err)
	}
	if comp.Name() != "labwc" {
		t.Errorf("DetectCompositor().Name() = %q, want labwc", comp.Name())
	}
}

func TestDetectLabwcByDesktopName(t *testing.T) {
	setLabwcSessionEnv(t)
	t.Setenv("XDG_CURRENT_DESKTOP", "labwc:wlroots")

	comp, err := DetectCompositor()
	if err != nil {
		t.Fatalf("DetectCompositor() error = %v, want labwc under XDG_CURRENT_DESKTOP", err)
	}
	if comp.Name() != "labwc" {
		t.Errorf("DetectCompositor().Name() = %q, want labwc", comp.Name())
	}
}

// TestDetectDoesNotGuessLabwcWithoutEnv: wlr-randr answers on every wlroots
// compositor, so the presence of the tool alone must not select labwc.
func TestDetectDoesNotGuessLabwcWithoutEnv(t *testing.T) {
	setLabwcSessionEnv(t)

	if _, err := DetectCompositor(); err == nil {
		t.Fatal("DetectCompositor() found a compositor with no labwc environment and no IPC tools")
	}
}
