// animation_cycle.go - Animation cycling functionality
package animations

import (
	"fmt"
	"math/rand"
	"time"
)

// AnimationCycler manages cycling through multiple animations
type AnimationCycler struct {
	animations     []Animation
	currentIdx     int
	lastSwitch     time.Time
	switchInterval time.Duration
	randomOrder    bool
}

// NewAnimationCycler creates a new animation cycler
func NewAnimationCycler(animations []Animation, switchInterval time.Duration, randomOrder bool) *AnimationCycler {
	return &AnimationCycler{
		animations:     animations,
		currentIdx:     0,
		lastSwitch:     time.Now(),
		switchInterval: switchInterval,
		randomOrder:    randomOrder,
	}
}

// GetCurrentAnimation returns the current animation
func (c *AnimationCycler) GetCurrentAnimation() Animation {
	if len(c.animations) == 0 {
		return nil
	}
	return c.animations[c.currentIdx]
}

// SwitchAnimation switches to the next animation based on the cycling settings
func (c *AnimationCycler) SwitchAnimation() error {
	if len(c.animations) == 0 {
		return fmt.Errorf("no animations to cycle through")
	}

	// Calculate if we should switch based on the interval
	now := time.Now()
	if now.Sub(c.lastSwitch) < c.switchInterval {
		return nil // Not time to switch yet
	}

	c.lastSwitch = now

	// Determine the next index
	if c.randomOrder {
		c.currentIdx = rand.Intn(len(c.animations))
	} else {
		c.currentIdx = (c.currentIdx + 1) % len(c.animations)
	}

	return nil
}

// SetSwitchInterval sets the time interval between animation switches
func (c *AnimationCycler) SetSwitchInterval(interval time.Duration) {
	c.switchInterval = interval
}

// GetSwitchInterval returns the current switch interval
func (c *AnimationCycler) GetSwitchInterval() time.Duration {
	return c.switchInterval
}

// AddAnimation adds a new animation to the cycle
func (c *AnimationCycler) AddAnimation(animation Animation) {
	c.animations = append(c.animations, animation)
}

// RemoveAnimation removes an animation from the cycle
func (c *AnimationCycler) RemoveAnimation(index int) error {
	if index < 0 || index >= len(c.animations) {
		return fmt.Errorf("animation index out of range")
	}

	// Don't allow removing the last animation
	if len(c.animations) <= 1 {
		return fmt.Errorf("cannot remove the last animation")
	}

	// Remove the animation
	c.animations = append(c.animations[:index], c.animations[index+1:]...)

	// Adjust current index if necessary
	if c.currentIdx >= len(c.animations) {
		c.currentIdx = len(c.animations) - 1
	}

	return nil
}

// SetRandomOrder sets whether animations should be played in random order
func (c *AnimationCycler) SetRandomOrder(random bool) {
	c.randomOrder = random
}

// GetRandomOrder returns whether animations are set to random order
func (c *AnimationCycler) GetRandomOrder() bool {
	return c.randomOrder
}

// CreateDefaultCycle creates a cycle with the default animations
func CreateDefaultCycle() (*AnimationCycler, error) {
	// Create the default animations
	matrix, err := NewMatrixAnimation(80, 24, "dracula")
	if err != nil {
		return nil, err
	}

	fire, err := NewFireAnimation(80, 24, "dracula")
	if err != nil {
		return nil, err
	}

	fireworks, err := NewFireworksAnimation(80, 24, "dracula")
	if err != nil {
		return nil, err
	}

	rain, err := NewRainAnimation(80, 24, "dracula")
	if err != nil {
		return nil, err
	}

	beams, err := NewBeamsAnimation(80, 24, "dracula")
	if err != nil {
		return nil, err
	}

	animations := []Animation{
		matrix,
		fire,
		fireworks,
		rain,
		beams,
	}

	// Create a cycler with 2 minute intervals and random order
	return NewAnimationCycler(animations, 2*time.Minute, true), nil
}
