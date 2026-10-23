package animations

import (
	"slices"
	"testing"

	syscGo "github.com/Nomadcxx/sysc-Go/animations"
)

func TestSyscGoRegistryEffectsCanBeConstructed(t *testing.T) {
	expectedEffects := []string{
		"skull", "sonar", "cracktro", "sysc-logo", "cross-logo", "logo-morph",
		"ak-47", "ak-47-text", "great-wave", "wasteland",
		"f-14-tomcat", "f-15e-strike-eagle", "f-22-raptor", "fa-18e-super-hornet",
	}
	effectNames := syscGo.GetEffectNames()
	for _, effect := range expectedEffects {
		if !slices.Contains(effectNames, effect) {
			t.Fatalf("sysc-Go registry is missing effect %q", effect)
		}
	}

	for _, effect := range AvailableEffectNames() {
		t.Run(effect, func(t *testing.T) {
			var anim Animation
			var err error
			if syscGo.IsTextBasedEffect(effect) {
				anim, err = CreateAnimationWithText(effect, 80, 24, "nord", "SYSC")
			} else {
				anim, err = CreateAnimation(effect, 80, 24, "nord")
			}
			if err != nil {
				t.Fatalf("construct: %v", err)
			}
			anim.Update(0)
			if frame := anim.Render(); frame == "" {
				t.Fatal("rendered an empty frame")
			}
			anim.Resize(100, 30)
			anim.Update(1)
			if frame := anim.Render(); frame == "" {
				t.Fatal("rendered an empty frame after resize")
			}
		})
	}
}

func TestBrokenEffectsAreUnavailable(t *testing.T) {
	for _, effect := range []string{"ak-47-rama", "justice-cross"} {
		if _, err := CreateAnimation(effect, 80, 24, "nord"); err == nil {
			t.Errorf("unfinished effect %q can be constructed", effect)
		}
	}
}

func TestAvailableEffectNamesMatchesRegistry(t *testing.T) {
	want := make([]string, 0, len(syscGo.GetEffectNames()))
	for _, effect := range syscGo.GetEffectNames() {
		if effect == "ak-47-rama" || effect == "justice-cross" {
			continue
		}
		want = append(want, effect)
	}
	if got := AvailableEffectNames(); !slices.Equal(got, want) {
		t.Fatalf("AvailableEffectNames() = %v, want %v", got, want)
	}
}

func TestSyscShellThemesUseSyscGoScreensaverPalettes(t *testing.T) {
	for _, theme := range []string{"ayu", "blue", "eldritch-abyss"} {
		if got, want := getThemePalette(theme), syscGo.GetScreensaverPalette(theme); !slices.Equal(got, want) {
			t.Errorf("getThemePalette(%q) = %v, want sysc-Go palette %v", theme, got, want)
		}
	}
}

type cellAspectEffect struct {
	width, height int
}

func (e *cellAspectEffect) Update()                {}
func (e *cellAspectEffect) Render() string         { return "frame" }
func (e *cellAspectEffect) Resize(int, int)        {}
func (e *cellAspectEffect) SetCellAspect(w, h int) { e.width, e.height = w, h }

func TestOptimizedSyscGoForwardsCellAspect(t *testing.T) {
	effect := &cellAspectEffect{}
	newOptimizedSyscGo(effect).SetCellAspect(9, 20)
	if effect.width != 9 || effect.height != 20 {
		t.Fatalf("cell aspect = %dx%d, want 9x20", effect.width, effect.height)
	}
}

// TestCreateOptimizedAnimation tests animation creation
func TestCreateOptimizedAnimation(t *testing.T) {
	tests := []struct {
		effect string
		hasErr bool
	}{
		{"matrix", false},
		{"fire", false},
		{"fireworks", false},
		{"rain", false},
		{"beams", false},
		{"beam-text", false},
		{"decrypt", false},
		{"pour", false},
		{"aquarium", false},
		{"print", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.effect, func(t *testing.T) {
			anim, err := CreateOptimizedAnimation(tt.effect, 80, 24, "nord")

			if tt.hasErr {
				if err == nil {
					t.Errorf("CreateOptimizedAnimation(%q) expected error, got nil", tt.effect)
				}
				return
			}

			if err != nil {
				t.Fatalf("CreateOptimizedAnimation(%q) unexpected error: %v", tt.effect, err)
			}

			if anim == nil {
				t.Fatal("Animation is nil")
			}

			// Test Update doesn't panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Update() panicked: %v", r)
					}
				}()
				anim.Update(1)
			}()

			// Test Render doesn't panic and returns something
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Render() panicked: %v", r)
					}
				}()
				output := anim.Render()
				if len(output) == 0 {
					t.Error("Render() returned empty string")
				}
			}()

			// Test Resize doesn't panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Resize() panicked: %v", r)
					}
				}()
				anim.Resize(100, 30)
			}()
		})
	}
}

// TestGetThemePalette tests theme palette lookup
func TestGetThemePalette(t *testing.T) {
	tests := []struct {
		theme       string
		expectEmpty bool
	}{
		{"nord", false},
		{"dracula", false},
		{"gruvbox", false},
		{"tokyo-night", false},
		{"catppuccin", false},
		{"material", false},
		{"solarized", false},
		{"monochrome", false},
		{"transishardjob", false},
		{"invalid-theme", false}, // Should return default (nord)
	}

	for _, tt := range tests {
		t.Run(tt.theme, func(t *testing.T) {
			palette := getThemePalette(tt.theme)

			if tt.expectEmpty {
				if len(palette) != 0 {
					t.Errorf("getThemePalette(%q) expected empty, got %d colors", tt.theme, len(palette))
				}
			} else {
				if len(palette) == 0 {
					t.Errorf("getThemePalette(%q) returned empty palette", tt.theme)
				}
				// All palettes should have valid hex colors
				for i, color := range palette {
					if len(color) < 4 || color[0] != '#' {
						t.Errorf("getThemePalette(%q) color[%d] = %q is invalid", tt.theme, i, color)
					}
				}
			}
		})
	}
}

// TestMinInt tests helper function
func TestMinInt(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, 5, -1},
		{-5, -1, -5},
	}

	for _, tt := range tests {
		result := minInt(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// TestCreateAnimation tests factory wrapper
func TestCreateAnimation(t *testing.T) {
	// Test that CreateAnimation properly delegates to CreateOptimizedAnimation
	anim, err := CreateAnimation("matrix", 80, 24, "nord")
	if err != nil {
		t.Fatalf("CreateAnimation() error = %v", err)
	}
	if anim == nil {
		t.Fatal("CreateAnimation() returned nil")
	}

	// Test invalid effect
	_, err = CreateAnimation("invalid", 80, 24, "nord")
	if err == nil {
		t.Error("CreateAnimation(invalid) expected error, got nil")
	}
}
