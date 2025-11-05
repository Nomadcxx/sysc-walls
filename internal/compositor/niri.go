// niri.go - Niri compositor implementation
package compositor

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// NiriCompositor implements the Compositor interface for Niri
type NiriCompositor struct {
	outputRegex *regexp.Regexp
}

// NewNiriCompositor creates a new Niri compositor instance
func NewNiriCompositor() *NiriCompositor {
	// Regex to parse: Output "name" (CONNECTOR)
	// We want to extract CONNECTOR (the actual display name)
	return &NiriCompositor{
		outputRegex: regexp.MustCompile(`Output "([^"]+)" \(([^)]+)\)`),
	}
}

// Name returns the compositor name
func (n *NiriCompositor) Name() string {
	return "niri"
}

// ListOutputs returns all available outputs
func (n *NiriCompositor) ListOutputs() ([]Output, error) {
	cmd := exec.Command("niri", "msg", "outputs")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run 'niri msg outputs': %w", err)
	}

	return n.parseOutputs(string(output))
}

// parseOutputs parses niri's text output format
// Example output:
//   Output "eDP-1" (eDP-1)
//   Output "HDMI-A-0" (HDMI-A-0)
func (n *NiriCompositor) parseOutputs(output string) ([]Output, error) {
	outputs := []Output{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Match: Output "name" (CONNECTOR)
		matches := n.outputRegex.FindStringSubmatch(line)
		if len(matches) >= 3 {
			// matches[1] = name (often same as connector)
			// matches[2] = CONNECTOR (the actual display identifier)
			outputs = append(outputs, Output{
				Name: matches[2], // Use CONNECTOR as the name
			})
		}
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("no outputs found in niri output")
	}

	return outputs, nil
}

// GetFocusedOutput returns the currently focused output
func (n *NiriCompositor) GetFocusedOutput() (string, error) {
	cmd := exec.Command("niri", "msg", "focused-output")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'niri msg focused-output': %w", err)
	}

	// Parse the output - should contain the focused output name
	// Example: Output "eDP-1" (eDP-1)
	outputStr := strings.TrimSpace(string(output))
	matches := n.outputRegex.FindStringSubmatch(outputStr)
	if len(matches) >= 3 {
		return matches[2], nil // Return CONNECTOR
	}

	// Fallback: just return the trimmed string if regex doesn't match
	return outputStr, nil
}

// FocusOutput focuses a specific output by name
func (n *NiriCompositor) FocusOutput(name string) error {
	cmd := exec.Command("niri", "msg", "action", "focus-monitor", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to focus output %s: %w", name, err)
	}
	return nil
}

// FullscreenFocusedWindow fullscreens the currently focused window
func (n *NiriCompositor) FullscreenFocusedWindow() error {
	cmd := exec.Command("niri", "msg", "action", "fullscreen-window")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fullscreen focused window: %w", err)
	}
	return nil
}
