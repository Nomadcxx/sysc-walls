// utils.go - Utility functions for terminal handling
package utils

import (
	"fmt"
	"os"
)

// GetTerminalSize returns the current terminal dimensions
func GetTerminalSize() (int, int, error) {
	// Simple heuristic for terminal dimensions
	// This doesn't rely on external libraries
	return 80, 24, nil // Default to standard terminal dimensions
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
