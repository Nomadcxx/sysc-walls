// compositor.go - Compositor abstraction for multi-monitor support
package compositor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrNoFocusControl reports a compositor that offers clients no way to move
// keyboard focus between outputs. labwc has no IPC at all, so the daemon
// cannot place a window on a chosen output and has to settle for the one
// that already holds focus.
var ErrNoFocusControl = errors.New("compositor cannot move focus between outputs")

// Output represents a display output/monitor
type Output struct {
	Name    string // Connector name (e.g., "DP-1", "HDMI-A-0")
	Width   int
	Height  int
	Focused bool
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

	// PrepareFullscreen sets up any compositor-specific rules for fullscreen
	// windows with the given class. This is called before launching the screensaver.
	// Returns nil if the compositor doesn't need special handling.
	PrepareFullscreen(windowClass string) error

	// CleanupFullscreen removes any compositor-specific rules added by PrepareFullscreen.
	// This is called after stopping the screensaver.
	CleanupFullscreen(windowClass string) error
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

	// Try labwc. It exports LABWC_PID to every client it starts and sets
	// XDG_CURRENT_DESKTOP to "labwc:wlroots"; wlr-randr then reports the
	// outputs. It comes last because wlr-randr answers on any wlroots
	// compositor, and the environment is what tells labwc apart.
	if os.Getenv("LABWC_PID") != "" || strings.HasPrefix(os.Getenv("XDG_CURRENT_DESKTOP"), "labwc") {
		if _, err := exec.LookPath("wlr-randr"); err == nil {
			return NewLabwcCompositor(), nil
		}
	}

	return nil, fmt.Errorf("no supported compositor detected (tried niri, hyprland, sway, labwc)")
}
