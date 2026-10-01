// hyprland.go - Hyprland compositor implementation
package compositor

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
)

// HyprlandCompositor implements the Compositor interface for Hyprland
type HyprlandCompositor struct {
	mu sync.Mutex
	// applied records window classes whose fullscreen rule has been added.
	// Rules persist in Hyprland until a reload and there is no way to remove
	// an individual rule, so re-adding on every launch only duplicates it.
	applied map[string]struct{}
	// runHyprctl executes hyprctl with the given arguments; nil uses the real
	// binary. Overridable in tests.
	runHyprctl func(args ...string) error
}

// hyprlandMonitor represents a monitor in hyprctl's JSON output
type hyprlandMonitor struct {
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Focused bool   `json:"focused"`
}

// NewHyprlandCompositor creates a new Hyprland compositor instance
func NewHyprlandCompositor() *HyprlandCompositor {
	return &HyprlandCompositor{}
}

// Name returns the compositor name
func (h *HyprlandCompositor) Name() string {
	return "hyprland"
}

// ListOutputs returns all available outputs
func (h *HyprlandCompositor) ListOutputs() ([]Output, error) {
	cmd := exec.Command("hyprctl", "monitors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run 'hyprctl monitors -j': %w", err)
	}

	return h.parseOutputs(output)
}

// parseOutputs parses hyprctl's JSON output
func (h *HyprlandCompositor) parseOutputs(data []byte) ([]Output, error) {
	var monitors []hyprlandMonitor
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, fmt.Errorf("failed to parse hyprctl JSON: %w", err)
	}

	outputs := make([]Output, 0, len(monitors))
	for _, mon := range monitors {
		outputs = append(outputs, Output{
			Name:    mon.Name,
			Width:   mon.Width,
			Height:  mon.Height,
			Focused: mon.Focused,
		})
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no outputs found in hyprctl output")
	}

	return outputs, nil
}

// GetFocusedOutput returns the currently focused output
func (h *HyprlandCompositor) GetFocusedOutput() (string, error) {
	outputs, err := h.ListOutputs()
	if err != nil {
		return "", err
	}

	for _, output := range outputs {
		if output.Focused {
			return output.Name, nil
		}
	}

	// If no focused output found, return first output as fallback
	if len(outputs) > 0 {
		return outputs[0].Name, nil
	}

	return "", fmt.Errorf("no focused output found")
}

// FocusOutput focuses a specific output by name
func (h *HyprlandCompositor) FocusOutput(name string) error {
	cmd := exec.Command("hyprctl", "dispatch", "focusmonitor", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to focus output %s: %w", name, err)
	}
	return nil
}

// PrepareFullscreen adds a window rule to force fullscreen for the given window class.
// This is needed because Hyprland may tile windows instead of respecting the terminal's
// fullscreen request (--start-as=fullscreen in kitty).
//
// The rule is added once per compositor instance: Hyprland rules persist until
// a reload, so re-applying on every screensaver launch would only accumulate
// duplicates. A failed application is not recorded, so the next launch retries.
func (h *HyprlandCompositor) PrepareFullscreen(windowClass string) error {
	h.mu.Lock()
	_, alreadyApplied := h.applied[windowClass]
	h.mu.Unlock()
	if alreadyApplied {
		return nil
	}

	// Add window rule to force fullscreen for windows with this class
	// Using windowrulev2 with class matching for precise targeting
	rule := fmt.Sprintf("fullscreen, class:(%s)", windowClass)
	run := h.runHyprctl
	if run == nil {
		run = func(args ...string) error {
			return exec.Command("hyprctl", args...).Run()
		}
	}
	if err := run("keyword", "windowrulev2", rule); err != nil {
		return fmt.Errorf("failed to add fullscreen window rule: %w", err)
	}

	h.mu.Lock()
	if h.applied == nil {
		h.applied = make(map[string]struct{})
	}
	h.applied[windowClass] = struct{}{}
	h.mu.Unlock()

	return nil
}

// CleanupFullscreen removes the fullscreen window rule for the given class.
// Note: Hyprland doesn't have a direct "remove rule" command, but the rule
// is harmless as it only affects windows with the specific class.
func (h *HyprlandCompositor) CleanupFullscreen(windowClass string) error {
	// Hyprland doesn't support removing individual rules directly.
	// The rule we added only affects windows with the specific class,
	// so it's safe to leave it. If needed, we could use "hyprctl reload"
	// but that would affect all user rules, which is too disruptive.
	//
	// The applied mark is intentionally kept: the rule is still in effect, so
	// the next PrepareFullscreen for this class must not add it again.
	return nil
}
