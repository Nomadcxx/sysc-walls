// sway.go - Sway compositor implementation
package compositor

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// SwayCompositor implements the Compositor interface for Sway
type SwayCompositor struct{}

// swayOutput represents an output in swaymsg's JSON output
type swayOutput struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Focused bool   `json:"focused"`
	Rect    struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"rect"`
}

// NewSwayCompositor creates a new Sway compositor instance
func NewSwayCompositor() *SwayCompositor {
	return &SwayCompositor{}
}

// Name returns the compositor name
func (s *SwayCompositor) Name() string {
	return "sway"
}

// ListOutputs returns all available outputs
func (s *SwayCompositor) ListOutputs() ([]Output, error) {
	cmd := exec.Command("swaymsg", "-t", "get_outputs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run 'swaymsg -t get_outputs': %w", err)
	}

	return s.parseOutputs(output)
}

// parseOutputs parses swaymsg's JSON output
func (s *SwayCompositor) parseOutputs(data []byte) ([]Output, error) {
	var swayOutputs []swayOutput
	if err := json.Unmarshal(data, &swayOutputs); err != nil {
		return nil, fmt.Errorf("failed to parse swaymsg JSON: %w", err)
	}

	outputs := make([]Output, 0)
	for _, sout := range swayOutputs {
		// Only include active outputs
		if !sout.Active {
			continue
		}

		outputs = append(outputs, Output{
			Name:    sout.Name,
			Width:   sout.Rect.Width,
			Height:  sout.Rect.Height,
			Focused: sout.Focused,
		})
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no active outputs found in swaymsg output")
	}

	return outputs, nil
}

// GetFocusedOutput returns the currently focused output
func (s *SwayCompositor) GetFocusedOutput() (string, error) {
	outputs, err := s.ListOutputs()
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
func (s *SwayCompositor) FocusOutput(name string) error {
	cmd := exec.Command("swaymsg", "focus", "output", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to focus output %s: %w", name, err)
	}
	return nil
}
