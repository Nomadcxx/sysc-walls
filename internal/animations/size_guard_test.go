package animations

import (
	"strings"
	"testing"
)

// allEffectNames is every effect the registry accepts, so the guard is proven
// against the whole set rather than the two that were reported.
var allEffectNames = []string{
	"matrix", "matrix-art", "fire", "fire-text", "fireworks", "rain",
	"rain-art", "beams", "beam-text", "decrypt", "pour", "aquarium",
	"print", "blackhole", "ring-text",
}

// TestEffectsDoNotPanicOnSmallTerminals drives every effect at a range of
// undersized and degenerate terminals.
//
// Two effects reach a math/rand.Intn whose argument is derived from the
// dimensions: fireworks uses width-20, and aquarium places fish in the range
// [height/15+2, height-10], which collapses on a short terminal. Intn panics
// on n <= 0, and a panic in the display process freezes the terminal on a
// partial frame.
func TestEffectsDoNotPanicOnSmallTerminals(t *testing.T) {
	sizes := [][2]int{
		{0, 0}, {1, 1}, {1, 40}, {80, 1}, {80, 2}, {80, 3}, {80, 6},
		{19, 39}, {20, 40}, {21, 40}, {80, 23}, {80, 24}, {10, 10},
		{-5, -5}, {-1, 40}, {80, -1},
	}

	for _, effect := range allEffectNames {
		for _, size := range sizes {
			w, h := size[0], size[1]
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("effect %q panicked at %dx%d: %v", effect, w, h, r)
					}
				}()

				anim, err := CreateAnimation(effect, w, h, "rama")
				if err != nil {
					// A clean rejection is fine; a panic is not.
					return
				}

				// Drive several frames, the way the render loop does.
				for frame := 0; frame < 5; frame++ {
					anim.Update(frame)
					_ = anim.Render()
				}
			}()
		}
	}
}

// TestClampEffectSizeLeavesRealTerminalsAlone is the guard against this
// change quietly altering rendering. A terminal at or above the minimums must
// pass through byte-identical, so normal screensaver output is unaffected.
func TestClampEffectSizeLeavesRealTerminalsAlone(t *testing.T) {
	unchanged := [][2]int{
		{21, 24}, {40, 24}, {80, 24}, {21, 40}, {80, 40},
		{120, 40}, {192, 108}, {3840, 2160}, {1000, 1000},
	}
	for _, size := range unchanged {
		w, h := size[0], size[1]
		gotW, gotH := clampEffectSize(w, h)
		if gotW != w || gotH != h {
			t.Errorf("clampEffectSize(%d, %d) = (%d, %d); a real terminal must pass through unchanged", w, h, gotW, gotH)
		}
	}
}

// TestClampEffectSizeRaisesUndersizedTerminals documents the intended floor.
func TestClampEffectSizeRaisesUndersizedTerminals(t *testing.T) {
	cases := []struct {
		w, h  int
		wantW int
		wantH int
	}{
		{0, 0, minEffectWidth, minEffectHeight},
		{1, 1, minEffectWidth, minEffectHeight},
		{20, 40, minEffectWidth, 40},
		{80, 2, 80, minEffectHeight},
		{80, 23, 80, minEffectHeight},
		{-5, -5, minEffectWidth, minEffectHeight},
		{-1, 40, minEffectWidth, 40},
		{80, -1, 80, minEffectHeight},
	}
	for _, c := range cases {
		gotW, gotH := clampEffectSize(c.w, c.h)
		if gotW != c.wantW || gotH != c.wantH {
			t.Errorf("clampEffectSize(%d, %d) = (%d, %d), want (%d, %d)", c.w, c.h, gotW, gotH, c.wantW, c.wantH)
		}
	}
}

// TestSmallTerminalStillRendersContent checks the clamp produces a usable
// surface rather than an empty or broken one.
func TestSmallTerminalStillRendersContent(t *testing.T) {
	anim, err := CreateAnimation("fireworks", 1, 1, "rama")
	if err != nil {
		t.Fatalf("CreateAnimation at 1x1: %v", err)
	}
	anim.Update(0)
	out := anim.Render()
	if strings.TrimSpace(out) == "" {
		t.Error("fireworks rendered nothing at a clamped 1x1 terminal")
	}
}

// TestEffectsDoNotPanicOnResize covers the live path a user reaches by
// dragging their window smaller. The SIGWINCH handler passes the raw new size
// straight to Resize, so the guard has to hold there as well as at
// construction.
func TestEffectsDoNotPanicOnResize(t *testing.T) {
	sizes := [][2]int{
		{0, 0}, {1, 1}, {1, 40}, {80, 1}, {80, 2}, {80, 3},
		{19, 39}, {20, 40}, {10, 5}, {10, 10}, {-5, -5}, {-1, 40}, {80, -1},
	}

	for _, effect := range allEffectNames {
		func() {
			anim, err := CreateAnimation(effect, 120, 40, "rama")
			if err != nil {
				return
			}
			anim.Update(0)
			_ = anim.Render()

			for _, size := range sizes {
				w, h := size[0], size[1]
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("effect %q panicked on Resize to %dx%d: %v", effect, w, h, r)
						}
					}()
					anim.Resize(w, h)
					anim.Update(1)
					_ = anim.Render()
				}()
			}
		}()
	}
}
