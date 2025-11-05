// systemd.go - Systemd integration for screensaver management
package systemd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// ScreensaverProcess represents a single screensaver process
type ScreensaverProcess struct {
	PID    int
	Cmd    *exec.Cmd
	Output string // Monitor identifier (e.g., "DP-1", "HDMI-A-0")
}

// SystemD handles systemd integration
type SystemD struct {
	config    *config.Config
	processes []ScreensaverProcess
	mu        sync.Mutex // Protects processes slice
}

// NewSystemD creates a new SystemD instance
func NewSystemD(cfg *config.Config) *SystemD {
	return &SystemD{
		config:    cfg,
		processes: []ScreensaverProcess{},
	}
}

// LaunchScreensaver starts the screensaver on a specific output
func (s *SystemD) LaunchScreensaver(command string, outputName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	// Store the process
	process := ScreensaverProcess{
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Output: outputName,
	}
	s.processes = append(s.processes, process)

	if s.config.IsDebug() {
		log.Printf("Launched screensaver on %s with PID: %d", outputName, process.PID)
	}

	return nil
}

// StopScreensaver stops all screensaver processes
func (s *SystemD) StopScreensaver() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.IsDebug() {
		log.Printf("StopScreensaver called - %d processes tracked", len(s.processes))
	}

	if len(s.processes) == 0 {
		if s.config.IsDebug() {
			log.Println("No tracked processes, trying pkill anyway")
		}
		// Fallback: try pkill
		killCmd := exec.Command("pkill", "-f", "kitty.*--class.*sysc-walls-screensaver")
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("pkill failed and no tracked processes: %w", err)
		}
		if s.config.IsDebug() {
			log.Println("Killed via pkill despite no tracked processes")
		}
		return nil
	}

	// Kill all tracked processes
	var lastError error
	for i, process := range s.processes {
		if s.config.IsDebug() {
			log.Printf("Killing process %d/%d: PID %d (output: %s)",
				i+1, len(s.processes), process.PID, process.Output)
		}

		// Try to kill the process
		if err := process.Cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill PID %d: %v", process.PID, err)
			lastError = err
			continue
		}

		// Wait for process to finish (non-blocking check)
		go func(cmd *exec.Cmd) {
			cmd.Wait()
		}(process.Cmd)
	}

	// Clear all processes
	s.processes = []ScreensaverProcess{}

	if s.config.IsDebug() {
		log.Println("All screensaver processes stopped")
	}

	return lastError
}

// IsRunning checks if any screensaver processes are running
func (s *SystemD) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.processes) == 0 {
		return false
	}

	// Check if at least one process is still running
	stillRunning := []ScreensaverProcess{}
	for _, process := range s.processes {
		if err := process.Cmd.Process.Signal(os.Signal(nil)); err == nil {
			// Process is still running
			stillRunning = append(stillRunning, process)
		}
	}

	// Update processes list to only include running processes
	s.processes = stillRunning

	return len(s.processes) > 0
}

// GetPIDs returns the process IDs of all running screensavers
func (s *SystemD) GetPIDs() ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.processes) == 0 {
		return nil, fmt.Errorf("no screensaver processes running")
	}

	pids := make([]int, len(s.processes))
	for i, process := range s.processes {
		pids[i] = process.PID
	}

	return pids, nil
}

// GetProcessCount returns the number of running screensaver processes
func (s *SystemD) GetProcessCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.processes)
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
