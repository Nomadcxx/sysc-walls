package animations

import "testing"

func TestTextEffectsImplementTextUpdatable(t *testing.T) {
	effects := []string{
		"fire-text",
		"matrix-art",
		"rain-art",
		"beam-text",
		"decrypt",
		"pour",
		"print",
		"blackhole",
		"ring-text",
	}

	for _, effect := range effects {
		t.Run(effect, func(t *testing.T) {
			anim, err := CreateAnimationWithText(effect, 80, 24, "rama", "TEST")
			if err != nil {
				t.Fatalf("CreateAnimationWithText(%s) error: %v", effect, err)
			}
			tu, ok := anim.(TextUpdatable)
			if !ok {
				t.Fatalf("%s does not implement TextUpdatable", effect)
			}

			// Should not panic
			tu.SetText("HELLO WORLD")
		})
	}
}

func TestNonTextEffectsDoNotImplementTextUpdatable(t *testing.T) {
	effects := []string{"matrix", "fire", "fireworks", "rain", "beams", "aquarium"}

	for _, effect := range effects {
		t.Run(effect, func(t *testing.T) {
			anim, err := CreateAnimation(effect, 80, 24, "rama")
			if err != nil {
				t.Fatalf("CreateAnimation(%s) error: %v", effect, err)
			}
			if _, ok := anim.(TextUpdatable); ok {
				t.Fatalf("%s should not implement TextUpdatable", effect)
			}
		})
	}
}

func TestIsTextUpdatable(t *testing.T) {
	textAnim, err := CreateAnimationWithText("matrix-art", 80, 24, "rama", "TEST")
	if err != nil {
		t.Fatalf("CreateAnimationWithText(matrix-art) error: %v", err)
	}
	plainAnim, err := CreateAnimation("matrix", 80, 24, "rama")
	if err != nil {
		t.Fatalf("CreateAnimation(matrix) error: %v", err)
	}

	if !IsTextUpdatable(textAnim) {
		t.Fatal("matrix-art should be TextUpdatable")
	}
	if IsTextUpdatable(plainAnim) {
		t.Fatal("matrix should not be TextUpdatable")
	}
}

func TestSetTextSurvivesResizeForStatefulWrappers(t *testing.T) {
	anim, err := CreateAnimationWithText("matrix-art", 80, 24, "rama", "INIT")
	if err != nil {
		t.Fatalf("CreateAnimationWithText(matrix-art) error: %v", err)
	}
	tu, ok := anim.(TextUpdatable)
	if !ok {
		t.Fatalf("matrix-art should be TextUpdatable, got %T", anim)
	}

	tu.SetText("UPDATED")
	anim.Resize(100, 30)

	matrixArt, ok := anim.(*optimizedMatrixArt)
	if !ok {
		t.Fatalf("expected *optimizedMatrixArt, got %T", anim)
	}
	if matrixArt.text != "UPDATED" {
		t.Fatalf("text state not updated, got %q", matrixArt.text)
	}
}
