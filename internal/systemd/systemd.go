// systemd.go - Systemd integration for screensaver management
package systemd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// ScreensaverProcess represents a single screensaver instance
type ScreensaverProcess struct {
	PID    int
	Cmd    *exec.Cmd
	Output string
}

// SystemD handles systemd integration
type SystemD struct {
	config     *config.Config
	processes  []*ScreensaverProcess
	compositor compositor.Compositor
}

// NewSystemD creates a new SystemD instance
func NewSystemD(cfg *config.Config) *SystemD {
	// Detect compositor
	comp, err := compositor.DetectCompositor()
	if err != nil {
		if cfg.IsDebug() {
			log.Printf("Warning: Failed to detect compositor: %v (multi-monitor may not work)", err)
		}
	} else if cfg.IsDebug() {
		log.Printf("Detected compositor: %s", comp.Name())
	}

	return &SystemD{
		config:     cfg,
		processes:  make([]*ScreensaverProcess, 0),
		compositor: comp,
	}
}

// LaunchScreensaver starts the screensaver on all outputs
func (s *SystemD) LaunchScreensaver(command string) error {
	// Parse the command string
	args, err := parseCommand(command)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	// If compositor not detected, launch on current output only
	if s.compositor == nil {
		if s.config.IsDebug() {
			log.Println("No compositor detected, launching on current output only")
		}
		return s.launchSingle(args)
	}

	// Get all outputs
	outputs, err := s.compositor.GetOutputs()
	if err != nil {
		if s.config.IsDebug() {
			log.Printf("Failed to get outputs: %v, launching on current output only", err)
		}
		return s.launchSingle(args)
	}

	if len(outputs) == 0 {
		return fmt.Errorf("no outputs found")
	}

	if s.config.IsDebug() {
		log.Printf("Launching screensaver on %d outputs: %v", len(outputs), outputs)
	}

	// Save currently focused output to restore later
	originalOutput, err := s.compositor.GetFocusedOutput()
	if err != nil {
		if s.config.IsDebug() {
			log.Printf("Warning: Could not get focused output: %v", err)
		}
		originalOutput = ""
	}

	// Launch screensaver on each output
	for _, output := range outputs {
		if s.config.IsDebug() {
			log.Printf("Focusing output: %s", output)
		}

		// Focus this output
		if err := s.compositor.FocusOutput(output); err != nil {
			if s.config.IsDebug() {
				log.Printf("Warning: Failed to focus output %s: %v", output, err)
			}
			continue
		}

		// Small delay to allow focus to settle
		time.Sleep(100 * time.Millisecond)

		// Launch screensaver on this output
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Start(); err != nil {
			if s.config.IsDebug() {
				log.Printf("Warning: Failed to start screensaver on %s: %v", output, err)
			}
			continue
		}

		// Track this process
		process := &ScreensaverProcess{
			PID:    cmd.Process.Pid,
			Cmd:    cmd,
			Output: output,
		}
		s.processes = append(s.processes, process)

		if s.config.IsDebug() {
			log.Printf("Launched screensaver on %s with PID: %d", output, process.PID)
		}

		// Small delay between launches
		time.Sleep(100 * time.Millisecond)
	}

	// Restore original focus
	if originalOutput != "" {
		if err := s.compositor.FocusOutput(originalOutput); err != nil {
			if s.config.IsDebug() {
				log.Printf("Warning: Failed to restore focus to %s: %v", originalOutput, err)
			}
		}
	}

	if len(s.processes) == 0 {
		return fmt.Errorf("failed to launch screensaver on any output")
	}

	if s.config.IsDebug() {
		log.Printf("Successfully launched %d screensaver instance(s)", len(s.processes))
	}
	return nil
}

// launchSingle launches screensaver on current output only (fallback)
func (s *SystemD) launchSingle(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start screensaver: %w", err)
	}

	process := &ScreensaverProcess{
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Output: "unknown",
	}
	s.processes = append(s.processes, process)

	if s.config.IsDebug() {
		log.Printf("Launched screensaver with PID: %d", process.PID)
	}

	return nil
}

// StopScreensaver stops all screensaver instances
func (s *SystemD) StopScreensaver() error {
	if s.config.IsDebug() {
		log.Printf("SystemD.StopScreensaver called - %d process(es) tracked", len(s.processes))
	}

	if len(s.processes) == 0 {
		if s.config.IsDebug() {
			log.Println("No tracked processes, trying pkill anyway")
		}
		// Try pkill as fallback
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
	var lastErr error
	killedCount := 0

	for _, process := range s.processes {
		if process.Cmd == nil {
			continue
		}

		if s.config.IsDebug() {
			log.Printf("Killing screensaver on %s (PID: %d)", process.Output, process.PID)
		}

		// Try to kill the process
		if err := process.Cmd.Process.Kill(); err != nil {
			if s.config.IsDebug() {
				log.Printf("Failed to kill PID %d: %v", process.PID, err)
			}
			lastErr = err
			continue
		}

		// Wait for it to finish (don't block on error)
		go func(cmd *exec.Cmd) {
			cmd.Wait()
		}(process.Cmd)

		killedCount++
	}

	// Also use pkill as backup to catch any orphaned processes
	killCmd := exec.Command("pkill", "-f", "kitty.*--class.*sysc-walls-screensaver")
	if err := killCmd.Run(); err == nil && s.config.IsDebug() {
		log.Println("pkill also used as backup")
	}

	// Clear all processes
	s.processes = make([]*ScreensaverProcess, 0)

	if killedCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to stop any screensaver instances: %w", lastErr)
	}

	if s.config.IsDebug() {
		log.Printf("Stopped %d screensaver instance(s)", killedCount)
	}
	return nil
}

// IsRunning checks if any screensaver instance is running
func (s *SystemD) IsRunning() bool {
	if len(s.processes) == 0 {
		return false
	}

	// Check if at least one process is still running
	for _, process := range s.processes {
		if process.Cmd != nil && process.Cmd.Process != nil {
			if err := process.Cmd.Process.Signal(os.Signal(nil)); err == nil {
				return true
			}
		}
	}

	// No processes running, clear the list
	s.processes = make([]*ScreensaverProcess, 0)
	return false
}

// GetPID returns the process ID of the first screensaver instance if running
func (s *SystemD) GetPID() (*int, error) {
	if len(s.processes) == 0 {
		return nil, fmt.Errorf("screensaver is not running")
	}

	pid := s.processes[0].PID
	return &pid, nil
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
