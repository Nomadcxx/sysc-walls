// compositor.go - Compositor abstraction for multi-monitor support
package compositor

import (
	"fmt"
	"os"
	"os/exec"
)

// Output represents a display output/monitor
type Output struct {
	Name       string // Connector name (e.g., "DP-1", "HDMI-A-0")
	Width      int
	Height     int
	Focused    bool
}

// Compositor interface for compositor-specific operations
type Compositor interface {
	// ListOutputs returns all available outputs
	ListOutputs() ([]Output, error)

	// GetFocusedOutput returns the currently focused output
	GetFocusedOutput() (string, error)

	// FocusOutput focuses a specific output by name
	FocusOutput(name string) error

	// Name returns the compositor name
	Name() string
}

// DetectCompositor detects and returns the appropriate compositor implementation
func DetectCompositor() (Compositor, error) {
	// Check environment variables to determine compositor
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil, fmt.Errorf("not running on Wayland")
	}

	// Try Niri first
	if _, err := exec.LookPath("niri"); err == nil {
		// Verify niri is actually running by checking if we can execute a command
		cmd := exec.Command("niri", "msg", "version")
		if err := cmd.Run(); err == nil {
			return NewNiriCompositor(), nil
		}
	}

	// Try Hyprland
	if _, err := exec.LookPath("hyprctl"); err == nil {
		// Verify hyprland is running
		cmd := exec.Command("hyprctl", "version")
		if err := cmd.Run(); err == nil {
			return NewHyprlandCompositor(), nil
		}
	}

	// Try Sway
	if _, err := exec.LookPath("swaymsg"); err == nil {
		// Verify sway is running
		cmd := exec.Command("swaymsg", "-t", "get_version")
		if err := cmd.Run(); err == nil {
			return NewSwayCompositor(), nil
		}
	}

	return nil, fmt.Errorf("no supported compositor detected (tried niri, hyprland, sway)")
}
