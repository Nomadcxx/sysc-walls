// systemd.go - Systemd integration for screensaver management
package systemd

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"

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
func (s *SystemD) LaunchScreensaver(terminal string, args []string, outputName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create the command with validated arguments
	cmd := exec.Command(terminal, args...)

	// Create new process group so we can kill all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
		// Edge case fallback: Process tracking may be empty if:
		// 1. Daemon crashed and restarted, losing track of existing processes
		// 2. Processes were started by a different mechanism
		// 3. Service stopped but processes lingered
		// Use pkill as best-effort cleanup for orphaned screensaver instances
		killCmd := exec.Command("pkill", "-f", "kitty.*--class.*sysc-walls-screensaver")
		_ = killCmd.Run() // best-effort, ignore error
		return nil
	}

	// Kill all tracked processes
	var lastError error
	for i, process := range s.processes {
		if s.config.IsDebug() {
			log.Printf("Stopping process %d/%d: PID %d (output: %s)",
				i+1, len(s.processes), process.PID, process.Output)
		}

		pid := process.PID

		// Step 1: Send SIGTERM to process group for graceful shutdown
		// Negative PID targets the entire process group
		if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
			if s.config.IsDebug() {
				log.Printf("SIGTERM to process group -%d failed: %v, trying direct kill", pid, err)
			}
			// Fallback: direct kill if process group kill fails
			_ = process.Cmd.Process.Kill()
			process.Cmd.Wait()
			continue
		}

		// Step 2: Wait up to 2s for graceful exit
		done := make(chan error, 1)
		go func() {
			done <- process.Cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
			if s.config.IsDebug() {
				log.Printf("PID %d exited gracefully via SIGTERM", pid)
			}
		case <-time.After(2 * time.Second):
			// Step 3: Force kill the process group
			if s.config.IsDebug() {
				log.Printf("PID %d did not exit after SIGTERM, sending SIGKILL", pid)
			}
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				if s.config.IsDebug() {
					log.Printf("SIGKILL to process group -%d failed: %v", pid, err)
				}
				// Last resort: direct process kill
				_ = process.Cmd.Process.Kill()
			}
			// Wait for process to be reaped (with timeout)
			select {
			case <-done:
				// Reaped
			case <-time.After(1 * time.Second):
				log.Printf("WARNING: PID %d could not be reaped after SIGKILL", pid)
				lastError = fmt.Errorf("failed to reap PID %d", pid)
			}
		}
	}

	// Clear all processes AFTER all are reaped
	s.processes = []ScreensaverProcess{}

	if s.config.IsDebug() {
		log.Println("All screensaver processes stopped and reaped")
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

	// Check if at least one process is still running via signal 0
	stillRunning := []ScreensaverProcess{}
	for _, process := range s.processes {
		if err := syscall.Kill(process.PID, 0); err == nil {
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


