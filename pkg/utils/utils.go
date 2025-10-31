// utils.go - Utility functions for terminal handling
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// GetTerminalSize returns the current terminal dimensions
func GetTerminalSize() (int, int, error) {
	// For fullscreen usage, we need to detect actual terminal size
	// Try multiple methods to get real dimensions

	// Method 1: Use tput for terminal size (most reliable)
	if cols, lines, err := getTerminalSizeTput(); err == nil {
		return cols, lines, nil
	}

	// Method 2: Use environment variables (common for fullscreen terminals)
	cols := 200 // Large default for fullscreen
	lines := 60 // Large default for fullscreen

	if colsEnv := os.Getenv("COLUMNS"); colsEnv != "" {
		if colVal, err := strconv.Atoi(colsEnv); err == nil && colVal > 0 {
			cols = colVal
		}
	}

	if linesEnv := os.Getenv("LINES"); linesEnv != "" {
		if lineVal, err := strconv.Atoi(linesEnv); err == nil && lineVal > 0 {
			lines = lineVal
		}
	}

	return cols, lines, nil
}

// getTerminalSizeTput gets terminal size using tput
func getTerminalSizeTput() (int, int, error) {
	// Try to use tput if available
	cmd := exec.Command("tput", "cols")
	if output, err := cmd.Output(); err == nil {
		if cols, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			cmd = exec.Command("tput", "lines")
			if output, err := cmd.Output(); err == nil {
				if lines, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
					return cols, lines, nil
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("tput method failed")
}

// SetupTerminal prepares the terminal for full-screen animations
func SetupTerminal() {
	fmt.Print("\033[2J")   // Clear screen
	fmt.Print("\033[H")    // Move cursor to top
	fmt.Print("\033[?25l") // Hide cursor
}

// RestoreTerminal resets the terminal after animation
func RestoreTerminal() {
	fmt.Print("\033[2J")   // Clear screen
	fmt.Print("\033[H")    // Move cursor to top
	fmt.Print("\033[?25h") // Show cursor
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[2J")
}

// MoveCursorTop moves the cursor to the top-left corner
func MoveCursorTop() {
	fmt.Print("\033[H")
}

// HideCursor hides the terminal cursor
func HideCursor() {
	fmt.Print("\033[?25l")
}

// ShowCursor shows the terminal cursor
func ShowCursor() {
	fmt.Print("\033[?25h")
}

// EnterFullscreen attempts to put the terminal in fullscreen mode
func EnterFullscreen() {
	// This is a no-op in most implementations
	// Different terminal emulators have different ways of entering fullscreen
	fmt.Printf("Entering fullscreen mode\n")
}

// GetPID returns the process ID
func GetPID() (int, error) {
	return os.Getpid(), nil
}

// GetPPID returns the parent process ID
func GetPPID() (int, error) {
	return os.Getppid(), nil
}
