package compositor

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// recordHyprctl captures the hyprctl invocations PrepareFullscreen makes and
// fails the first `failsAt` of them.
func recordHyprctl(calls *[][]string, failsAt int) func(...string) error {
	return func(args ...string) error {
		*calls = append(*calls, args)
		if failsAt > 0 && len(*calls) == failsAt {
			return errors.New("hyprctl: keyword failed")
		}
		return nil
	}
}

// Hyprland dropped windowrulev2 in 0.53: the parser answers "windowrulev2 is
// deprecated" and does nothing. The rule has to use the current keyword and
// value shape, or the screensaver silently stays tiled.
func TestPrepareFullscreenUsesTheCurrentKeyword(t *testing.T) {
	var calls [][]string
	h := NewHyprlandCompositor()
	h.runHyprctl = recordHyprctl(&calls, 0)

	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatalf("PrepareFullscreen: %v", err)
	}
	want := []string{"keyword", "windowrule", "match:class sysc-walls-screensaver, fullscreen on"}
	if !reflect.DeepEqual(calls, [][]string{want}) {
		t.Fatalf("hyprctl calls = %v, want %v", calls, [][]string{want})
	}
}

// The rule cannot be removed once added, so re-adding it on the next launch
// only appends a duplicate. One launch must produce exactly one call (#38).
func TestPrepareFullscreenAppliesOnce(t *testing.T) {
	var calls [][]string
	h := NewHyprlandCompositor()
	h.runHyprctl = recordHyprctl(&calls, 0)

	for i := 0; i < 3; i++ {
		if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
			t.Fatalf("PrepareFullscreen %d: %v", i, err)
		}
	}
	if len(calls) != 1 {
		t.Fatalf("hyprctl called %d times over 3 launches, want 1", len(calls))
	}
}

// A rule that was never applied must be retried; recording a failure as done
// would leave the screensaver tiled for the rest of the daemon's life.
func TestPrepareFullscreenRetriesAfterFailure(t *testing.T) {
	var calls [][]string
	h := NewHyprlandCompositor()
	h.runHyprctl = recordHyprctl(&calls, 1)

	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err == nil {
		t.Fatal("PrepareFullscreen hid a failed hyprctl")
	}
	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("hyprctl called %d times, want 2", len(calls))
	}
	// The retry must not repeat once it has succeeded.
	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatalf("third launch: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("hyprctl called %d times, want 2 once the retry succeeded", len(calls))
	}
}

// Two compositor instances stand for two daemon runs. Each applies its own
// rule once, so a restart still fixes a session where the rule was lost.
func TestPrepareFullscreenIsPerInstance(t *testing.T) {
	var first, second [][]string
	a := NewHyprlandCompositor()
	a.runHyprctl = recordHyprctl(&first, 0)
	b := NewHyprlandCompositor()
	b.runHyprctl = recordHyprctl(&second, 0)

	if err := a.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatal(err)
	}
	if err := a.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatal(err)
	}
	if err := b.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("hyprctl calls = %d and %d, want 1 and 1", len(first), len(second))
	}
}

// The guard must not race when two launches overlap.
func TestPrepareFullscreenIsSafeConcurrently(t *testing.T) {
	var calls [][]string
	h := NewHyprlandCompositor()
	h.runHyprctl = func(args ...string) error {
		calls = append(calls, args)
		return nil
	}

	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
				t.Error(err)
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	if len(calls) != 1 {
		t.Fatalf("hyprctl called %d times under -race, want 1", len(calls))
	}
}

// Niri and Sway need no rule, so their no-op contract must stay a no-op.
func TestOtherCompositorsPrepareNothing(t *testing.T) {
	for _, c := range []Compositor{NewNiriCompositor(), NewSwayCompositor()} {
		if err := c.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
			t.Errorf("%s PrepareFullscreen: %v", c.Name(), err)
		}
		if err := c.CleanupFullscreen("sysc-walls-screensaver"); err != nil {
			t.Errorf("%s CleanupFullscreen: %v", c.Name(), err)
		}
	}
}

// The rule value is one comma separated field per match or effect; the class
// must not be wrapped in the old v2 regex-ish syntax.
func TestPrepareFullscreenValueHasNoV2Syntax(t *testing.T) {
	var calls [][]string
	h := NewHyprlandCompositor()
	h.runHyprctl = recordHyprctl(&calls, 0)
	if err := h.PrepareFullscreen("sysc-walls-screensaver"); err != nil {
		t.Fatal(err)
	}
	rule := calls[0][2]
	for _, dead := range []string{"windowrulev2", "class:(", "class:("} {
		if contains(rule, dead) {
			t.Errorf("rule %q still uses %q", rule, dead)
		}
	}
	want := fmt.Sprintf("match:class %s, fullscreen on", "sysc-walls-screensaver")
	if rule != want {
		t.Errorf("rule = %q, want %q", rule, want)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
