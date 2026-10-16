package animations

import (
	"testing"

	syscGo "github.com/Nomadcxx/sysc-Go/animations"
)

func TestTextEffectsRespectRuntimeUpdateSupport(t *testing.T) {
	for _, effect := range syscGo.GetTextBasedEffects() {
		t.Run(effect, func(t *testing.T) {
			anim, err := CreateAnimationWithText(effect, 80, 24, "rama", "TEST")
			if err != nil {
				t.Fatalf("CreateAnimationWithText(%s) error: %v", effect, err)
			}
			tu, ok := anim.(TextUpdatable)
			wantUpdatable := effect != "ak-47-text"
			if ok != wantUpdatable {
				t.Fatalf("%s TextUpdatable = %v, want %v", effect, ok, wantUpdatable)
			}

			if ok {
				// Should not panic
				tu.SetText("HELLO WORLD")
			}
		})
	}
}

func TestNonTextEffectsDoNotImplementTextUpdatable(t *testing.T) {
	for _, effect := range AvailableEffectNames() {
		if syscGo.IsTextBasedEffect(effect) {
			continue
		}
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
