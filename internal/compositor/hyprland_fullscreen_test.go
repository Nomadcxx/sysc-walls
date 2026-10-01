package compositor

import (
	"fmt"
	"slices"
	"testing"
)

// TestHyprlandPrepareFullscreenAppliesOnce verifies the fullscreen rule is
// applied once per compositor instance, not on every call (#38).
func TestHyprlandPrepareFullscreenAppliesOnce(t *testing.T) {
	h := NewHyprlandCompositor()
	var calls [][]string
	h.runHyprctl = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}

	for i := 0; i < 3; i++ {
		if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
			t.Fatalf("PrepareFullscreen() call %d error = %v", i+1, err)
		}
	}

	if len(calls) != 1 {
		t.Fatalf("hyprctl invoked %d times, want 1", len(calls))
	}
	want := []string{"keyword", "windowrulev2", "fullscreen, class:(sysc-walls-screensaver)"}
	if !slices.Equal(calls[0], want) {
		t.Errorf("hyprctl args = %v, want %v", calls[0], want)
	}
}

// TestHyprlandPrepareFullscreenRetriesAfterFailure verifies a failed
// application is not recorded as applied, so a later launch retries.
func TestHyprlandPrepareFullscreenRetriesAfterFailure(t *testing.T) {
	h := NewHyprlandCompositor()
	fail := true
	calls := 0
	h.runHyprctl = func(args ...string) error {
		calls++
		if fail {
			return fmt.Errorf("hyprland not running")
		}
		return nil
	}

	for i := 0; i < 2; i++ {
		if err := h.PrepareFullscreen("sysc-walls-screensaver"); err == nil {
			t.Fatalf("PrepareFullscreen() call %d error = nil, want failure", i+1)
		}
	}
	if calls != 2 {
		t.Fatalf("hyprctl invoked %d times, want 2 (failure must not be cached)", calls)
	}

	fail = false
	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatalf("PrepareFullscreen() after recovery error = %v", err)
	}
	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatalf("PrepareFullscreen() repeat error = %v", err)
	}
	if calls != 3 {
		t.Fatalf("hyprctl invoked %d times, want 3 (success applied once)", calls)
	}
}

// TestHyprlandPrepareFullscreenSeparateClasses verifies distinct window
// classes still each get their own rule.
func TestHyprlandPrepareFullscreenSeparateClasses(t *testing.T) {
	h := NewHyprlandCompositor()
	calls := 0
	h.runHyprctl = func(args ...string) error {
		calls++
		return nil
	}

	for _, class := range []string{"class-a", "class-b", "class-a"} {
		if err := h.PrepareFullscreen(class); err != nil {
			t.Fatalf("PrepareFullscreen(%q) error = %v", class, err)
		}
	}
	if calls != 2 {
		t.Fatalf("hyprctl invoked %d times, want 2", calls)
	}
}
