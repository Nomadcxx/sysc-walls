// multi_display.go - Multi-display handling for the screensaver
package multi_display

import (
	"fmt"
	"os"
	"os/exec"
)

// Display represents a display/monitor
type Display struct {
	Name   string
	Width  int
	Height int
}

// MultiDisplay manages multiple displays
type MultiDisplay struct {
	displays      []Display
	activeDisplay int
}

// NewMultiDisplay creates a new MultiDisplay instance
func NewMultiDisplay() (*MultiDisplay, error) {
	// Detect available displays
	displays, err := detectDisplays()
	if err != nil {
		return nil, fmt.Errorf("failed to detect displays: %w", err)
	}

	// If no displays were detected, assume a single default display
	if len(displays) == 0 {
		displays = []Display{
			{
				Name:   "default",
				Width:  1920,
				Height: 1080,
			},
		}
	}

	return &MultiDisplay{
		displays:      displays,
		activeDisplay: 0,
	}, nil
}

// GetActiveDisplay returns the currently active display
func (m *MultiDisplay) GetActiveDisplay() *Display {
	if m.activeDisplay < 0 || m.activeDisplay >= len(m.displays) {
		return nil
	}
	return &m.displays[m.activeDisplay]
}

// SetActiveDisplay sets the active display by index
func (m *MultiDisplay) SetActiveDisplay(index int) error {
	if index < 0 || index >= len(m.displays) {
		return fmt.Errorf("display index out of range")
	}

	m.activeDisplay = index
	return nil
}

// GetAllDisplays returns all detected displays
func (m *MultiDisplay) GetAllDisplays() []Display {
	return m.displays
}

// detectDisplays detects available displays using appropriate tools
func detectDisplays() ([]Display, error) {
	displays := []Display{}

	// Try Wayland first
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		// Try to detect displays using wlr-randr
		cmd := exec.Command("wlr-randr")
		if err := cmd.Run(); err == nil {
			output, err := cmd.Output()
			if err != nil {
				return displays, fmt.Errorf("failed to get wlr-randr output: %w", err)
			}

			displays, err = parseWlrRandrOutput(string(output))
			if err != nil {
				return displays, fmt.Errorf("failed to parse wlr-randr output: %w", err)
			}

			return displays, nil
		}

		// Try to detect displays using gammastep
		cmd = exec.Command("gammastep -l")
		if err := cmd.Run(); err == nil {
			output, err := cmd.Output()
			if err != nil {
				return displays, fmt.Errorf("failed to get gammastep output: %w", err)
			}

			displays, err = parseGammastepOutput(string(output))
			if err != nil {
				return displays, fmt.Errorf("failed to parse gammastep output: %w", err)
			}

			return displays, nil
		}

		// Try xrandr as fallback
		cmd = exec.Command("xrandr")
		if err := cmd.Run(); err == nil {
			output, err := cmd.Output()
			if err != nil {
				return displays, fmt.Errorf("failed to get xrandr output: %w", err)
			}

			displays, err = parseXrandrOutput(string(output))
			if err != nil {
				return displays, fmt.Errorf("failed to parse xrandr output: %w", err)
			}

			return displays, nil
		}
	}

	// Try X11
	if os.Getenv("DISPLAY") != "" {
		// Try to detect displays using xrandr
		cmd := exec.Command("xrandr")
		if err := cmd.Run(); err == nil {
			output, err := cmd.Output()
			if err != nil {
				return displays, fmt.Errorf("failed to get xrandr output: %w", err)
			}

			displays, err = parseXrandrOutput(string(output))
			if err != nil {
				return displays, fmt.Errorf("failed to parse xrandr output: %w", err)
			}

			return displays, nil
		}
	}

	return displays, nil
}

// parseWlrRandrOutput parses the output of wlr-randr
func parseWlrRandrOutput(output string) ([]Display, error) {
	displays := []Display{}

	lines := splitLines(output)

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Example wlr-randr output line:
		// HDMI-A-1 1920x1080@120.000Hz 1919x1079+0+0
		parts := splitBySpace(line)

		if len(parts) >= 2 {
			display := Display{
				Name: parts[0],
			}

			// Parse dimensions from the resolution string
			resParts := splitByX(parts[1])
			if len(resParts) >= 2 {
				if w, err := toInt(resParts[0]); err == nil {
					display.Width = w
				}

				if h, err := toInt(resParts[1]); err == nil {
					display.Height = h
				}
			}

			displays = append(displays, display)
		}
	}

	return displays, nil
}

// parseGammastepOutput parses the output of gammastep -l
func parseGammastepOutput(output string) ([]Display, error) {
	displays := []Display{}

	lines := splitLines(output)

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Example gammastep output line:
		//   0: +1920x1080+0+0 1919x1079 (0x46)
		parts := splitBySpace(line)

		if len(parts) >= 2 {
			display := Display{}

			// Parse the name from the first part
			if parts[0] != "" && parts[0] != "0:" {
				display.Name = parts[0]
			} else {
				display.Name = "display" + parts[0]
			}

			// Parse dimensions from the position string
			posParts := splitByPlus(parts[1])
			if len(posParts) >= 1 {
				resParts := splitByX(posParts[0])
				if len(resParts) >= 2 {
					if w, err := toInt(resParts[0]); err == nil {
						display.Width = w
					}

					if h, err := toInt(resParts[1]); err == nil {
						display.Height = h
					}
				}
			}

			displays = append(displays, display)
		}
	}

	return displays, nil
}

// parseXrandrOutput parses the output of xrandr
func parseXrandrOutput(output string) ([]Display, error) {
	displays := []Display{}

	lines := splitLines(output)

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Example xrandr output line:
		// HDMI1 connected 1920x1080+0+0 (0x46) 476mm x 268mm
		if line[0] != ' ' {
			parts := splitBySpace(line)

			if len(parts) >= 3 {
				display := Display{
					Name: parts[0],
				}

				// Skip connected/disconnected status
				// Parse dimensions from the resolution string
				resParts := splitByX(parts[2])
				if len(resParts) >= 2 {
					if w, err := toInt(resParts[0]); err == nil {
						display.Width = w
					}

					if h, err := toInt(resParts[1]); err == nil {
						display.Height = h
					}
				}

				displays = append(displays, display)
			}
		}
	}

	return displays, nil
}

// Helper functions for string parsing

func splitLines(s string) []string {
	lines := []string{}

	for _, line := range s {
		if line == '\n' {
			continue
		}
	}

	// Simple split by newline
	parts := splitBy(s, '\n')

	for _, part := range parts {
		if part != "" {
			lines = append(lines, part)
		}
	}

	return lines
}

func splitBySpace(s string) []string {
	return splitBy(s, ' ')
}

func splitByX(s string) []string {
	return splitBy(s, 'x')
}

func splitByPlus(s string) []string {
	return splitBy(s, '+')
}

func splitBy(s string, delim rune) []string {
	parts := []string{}

	current := ""
	for _, char := range s {
		if char == delim {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func toInt(s string) (int, error) {
	result := 0
	mul := 1

	// Handle negative numbers
	if s[0] == '-' {
		mul = -1
		s = s[1:]
	}

	for _, char := range s {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}

		result = result*10 + int(char-'0')
	}

	return result * mul, nil
}
