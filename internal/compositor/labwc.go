// labwc.go - labwc compositor implementation
package compositor

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// LabwcCompositor implements the Compositor interface for labwc.
//
// labwc has no IPC of its own, so outputs are read through wlr-randr, which
// speaks wlr-output-management. Nothing outside the compositor can move
// keyboard focus, so FocusOutput reports ErrNoFocusControl and the daemon
// falls back to a single instance on whichever output already has focus.
type LabwcCompositor struct {
	// runWlrRandr runs wlr-randr and returns its stdout. A nil value means
	// the real command.
	runWlrRandr func(args ...string) ([]byte, error)
}

// labwcHead is one element of `wlr-randr --json`. The array holds one object
// per head, so an output with two heads appears twice under the same name.
type labwcHead struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Modes   []struct {
		Width   int  `json:"width"`
		Height  int  `json:"height"`
		Current bool `json:"current"`
	} `json:"modes"`
}

// NewLabwcCompositor creates a new labwc compositor instance
func NewLabwcCompositor() *LabwcCompositor {
	return &LabwcCompositor{}
}

// Name returns the compositor name
func (l *LabwcCompositor) Name() string {
	return "labwc"
}

// ListOutputs returns all enabled outputs
func (l *LabwcCompositor) ListOutputs() ([]Output, error) {
	run := l.runWlrRandr
	if run == nil {
		run = func(args ...string) ([]byte, error) {
			return exec.Command("wlr-randr", args...).Output()
		}
	}

	data, err := run("--json")
	if err != nil {
		return nil, fmt.Errorf("failed to run 'wlr-randr --json': %w", err)
	}

	return l.parseOutputs(data)
}

// parseOutputs parses the head array printed by wlr-randr --json
func (l *LabwcCompositor) parseOutputs(data []byte) ([]Output, error) {
	var heads []labwcHead
	if err := json.Unmarshal(data, &heads); err != nil {
		return nil, fmt.Errorf("failed to parse wlr-randr JSON: %w", err)
	}

	outputs := make([]Output, 0, len(heads))
	seen := make(map[string]bool, len(heads))
	for _, head := range heads {
		if !head.Enabled || seen[head.Name] {
			continue
		}
		seen[head.Name] = true

		output := Output{Name: head.Name}
		for _, mode := range head.Modes {
			if mode.Current {
				output.Width = mode.Width
				output.Height = mode.Height
				break
			}
		}
		outputs = append(outputs, output)
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no enabled outputs found in wlr-randr output")
	}

	return outputs, nil
}

// GetFocusedOutput cannot be answered: labwc exposes no state to clients.
// Returning ErrNoFocusControl leaves the caller without a focus to restore,
// which costs nothing here because focus is never moved.
func (l *LabwcCompositor) GetFocusedOutput() (string, error) {
	return "", ErrNoFocusControl
}

// FocusOutput reports that focus cannot be moved from outside labwc
func (l *LabwcCompositor) FocusOutput(name string) error {
	return ErrNoFocusControl
}

// PrepareFullscreen is a no-op for labwc as it properly respects the
// terminal's fullscreen request.
func (l *LabwcCompositor) PrepareFullscreen(windowClass string) error {
	return nil
}

// CleanupFullscreen is a no-op for labwc.
func (l *LabwcCompositor) CleanupFullscreen(windowClass string) error {
	return nil
}
