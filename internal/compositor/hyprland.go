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
	// fullscreenMu guards fullscreenApplied. Hyprland has no way to remove one
	// window rule at a time, so the rule is added once and stays for the rest of
	// the Hyprland session; re-adding it on every screensaver launch would
	// append a duplicate each time (#38).
	fullscreenMu      sync.Mutex
	fullscreenApplied bool

	// runHyprctl runs one hyprctl invocation. nil means the real binary, so
	// production is unchanged; a test substitutes it.
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
// The rule is applied once per compositor instance. CleanupFullscreen cannot
// remove it, so applying it again on the next launch would only append a
// duplicate rule for the same class. A failed application is not recorded, so
// the next launch retries exactly as before.
func (h *HyprlandCompositor) PrepareFullscreen(windowClass string) error {
	h.fullscreenMu.Lock()
	defer h.fullscreenMu.Unlock()
	if h.fullscreenApplied {
		return nil
	}

	// windowrulev2 was removed in Hyprland 0.53; the current parser only
	// accepts "windowrulev2 is deprecated" and does nothing with it, so the
	// old call was a silent no-op and the screensaver stayed tiled. The
	// replacement is a windowrule value of "match:<prop> <value>, <effect>
	// <value>"; the fields are comma separated and each is trimmed.
	rule := fmt.Sprintf("match:class %s, fullscreen on", windowClass)
	if err := h.hyprctl("keyword", "windowrule", rule); err != nil {
		return fmt.Errorf("failed to add fullscreen window rule: %w", err)
	}
	h.fullscreenApplied = true
	return nil
}

// hyprctl runs hyprctl with the given arguments, or the substituted runner
// when a test installed one.
func (h *HyprlandCompositor) hyprctl(args ...string) error {
	if h.runHyprctl != nil {
		return h.runHyprctl(args...)
	}
	return exec.Command("hyprctl", args...).Run()
}

// CleanupFullscreen removes the fullscreen window rule for the given class.
// Note: Hyprland doesn't have a direct "remove rule" command, but the rule
// is harmless as it only affects windows with the specific class.
func (h *HyprlandCompositor) CleanupFullscreen(windowClass string) error {
	// Hyprland doesn't support removing individual rules directly.
	// The rule we added only affects windows with the specific class,
	// so it's safe to leave it. If needed, we could use "hyprctl reload"
	// but that would affect all user rules, which is too disruptive.
	return nil
}
