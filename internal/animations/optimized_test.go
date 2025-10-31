package animations

import (
	"testing"
)

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
