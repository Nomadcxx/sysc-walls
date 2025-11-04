// systemd.go - Systemd integration for screensaver management
package systemd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// SystemD handles systemd integration
type SystemD struct {
	config    *config.Config
	processID *int
	cmd       *exec.Cmd
}

// NewSystemD creates a new SystemD instance
func NewSystemD(cfg *config.Config) *SystemD {
	return &SystemD{
		config:    cfg,
		processID: nil,
		cmd:       nil,
	}
}

// LaunchScreensaver starts the screensaver
func (s *SystemD) LaunchScreensaver(command string) error {
	// Parse the command string
	args, err := parseCommand(command)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	// Create the command
	cmd := exec.Command(args[0], args[1:]...)

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start screensaver: %w", err)
	}

	// Store the process ID
	pid := cmd.Process.Pid
	s.processID = &pid
	s.cmd = cmd

	if s.config.IsDebug() {
		fmt.Printf("Launched screensaver with PID: %d\n", pid)
	}

	return nil
}

// StopScreensaver stops the screensaver
func (s *SystemD) StopScreensaver() error {
	fmt.Printf("SystemD.StopScreensaver called - processID=%v cmd=%v\n", s.processID, s.cmd)
	
	if s.processID == nil || s.cmd == nil {
		fmt.Println("SystemD has no tracked process, trying pkill anyway")
		// Don't return error, try pkill as fallback
		killCmd := exec.Command("pkill", "-f", "kitty.*--class.*sysc-walls-screensaver")
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("pkill failed and no tracked process: %w", err)
		}
		fmt.Println("Killed via pkill despite no tracked process")
		return nil
	}

	fmt.Printf("Attempting to kill process with PID: %d\n", *s.processID)

	// First, try to kill just the screensaver kitty window by class name
	// This prevents killing all kitty instances
	killCmd := exec.Command("pkill", "-f", "kitty.*--class.*sysc-walls-screensaver")
	if err := killCmd.Run(); err != nil {
		fmt.Printf("pkill by class failed: %v, falling back to PID kill\n", err)
		
		// Fallback: kill the process tree starting from our PID
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop screensaver: %w", err)
		}
		fmt.Println("Killed via Process.Kill()")
	} else {
		fmt.Println("Killed via pkill by class")
	}

	// Wait for the process to finish
	if err := s.cmd.Wait(); err != nil {
		// Process might have already been killed
		if exitErr, ok := err.(*exec.ExitError); ok {
			// This is expected when a process is killed with SIGKILL
			if exitErr.ExitCode() == -1 {
				// This is fine, it means the process was killed
				// Clear the process information
				s.processID = nil
				s.cmd = nil
				return nil
			}
		}
		return fmt.Errorf("error waiting for screensaver process: %w", err)
	}

	// Clear the process information
	s.processID = nil
	s.cmd = nil

	if s.config.IsDebug() {
		fmt.Println("Stopped screensaver")
	}

	return nil
}

// IsRunning checks if the screensaver is running
func (s *SystemD) IsRunning() bool {
	if s.processID == nil || s.cmd == nil {
		return false
	}

	// Check if the process is still running
	if err := s.cmd.Process.Signal(os.Signal(nil)); err != nil {
		// Process is not running anymore
		s.processID = nil
		s.cmd = nil
		return false
	}

	return true
}

// GetPID returns the process ID of the screensaver if it's running
func (s *SystemD) GetPID() (*int, error) {
	if s.processID == nil {
		return nil, fmt.Errorf("screensaver is not running")
	}

	return s.processID, nil
}

// parseCommand parses a command string into arguments
func parseCommand(command string) ([]string, error) {
	// A very simple command parser that splits by spaces
	// For production, consider using a more robust parser like shlex or go-shlex
	if command == "" {
		return nil, fmt.Errorf("empty command string")
	}

	// Split by spaces, respecting quotes
	// This is a simple implementation, for a more robust solution use shlex or similar
	parts := []string{}
	current := ""
	inQuotes := false
	quoteChar := ""

	for _, char := range command {
		switch char {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = string(char)
			} else if string(char) == quoteChar {
				inQuotes = false
				quoteChar = ""
			} else {
				current += string(char)
			}
		case ' ':
			if !inQuotes {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	// Check if the command exists
	if len(parts) == 0 {
		return nil, fmt.Errorf("no command found")
	}

	return parts, nil
}
