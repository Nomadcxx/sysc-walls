// animations.go - Animation handling
package animations

// Animation interface for all animations
type Animation interface {
	Update(frame int)
	Render() string
	Resize(width, height int)
}

// TextUpdatable allows changing animation text at runtime.
type TextUpdatable interface {
	SetText(text string)
}

// IsTextUpdatable checks whether an animation supports runtime text updates.
func IsTextUpdatable(anim Animation) bool {
	_, ok := anim.(TextUpdatable)
	return ok
}

// CreateAnimation creates an animation using direct library integration
func CreateAnimation(effect string, width, height int, theme string) (Animation, error) {
	// Use optimized implementation that directly calls sysc-Go library
	return CreateOptimizedAnimation(effect, width, height, theme)
}

// CreateAnimationWithText creates an animation with custom text content for text-based effects
func CreateAnimationWithText(effect string, width, height int, theme string, text string) (Animation, error) {
	// Use optimized implementation with text support
	return CreateOptimizedAnimationWithText(effect, width, height, theme, text)
}
