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

	// done is closed once the process has exited and been reaped. A nil
	// channel means the process was not started through LaunchScreensaver,
	// so nothing is reaping it.
	done chan struct{}
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
		done:   make(chan struct{}),
	}
	s.processes = append(s.processes, process)

	// Reap the child as soon as it exits. Without this the daemon keeps a
	// zombie for every screensaver that exits on its own, and kill(pid, 0)
	// still reports a zombie as alive, so IsRunning() could never tell a dead
	// screensaver from a live one.
	go func(p ScreensaverProcess) {
		err := p.Cmd.Wait()
		if s.config.IsDebug() {
			log.Printf("Screensaver on %s (PID %d) exited: %v", p.Output, p.PID, err)
		}
		close(p.done)
	}(process)

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
		}

		// Step 2: Wait up to 2s for graceful exit. The reaper goroutine owns
		// the Cmd's Wait, so exit is observed through the done channel.
		if process.waitExit(2 * time.Second) {
			if s.config.IsDebug() {
				log.Printf("PID %d exited gracefully via SIGTERM", pid)
			}
			continue
		}

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
		if !process.waitExit(1 * time.Second) {
			log.Printf("WARNING: PID %d could not be reaped after SIGKILL", pid)
			lastError = fmt.Errorf("failed to reap PID %d", pid)
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

	// Drop processes that have exited. kill(pid, 0) cannot be used for this:
	// it still succeeds for a zombie, which is exactly the state a
	// screensaver that exited on its own is left in.
	stillRunning := make([]ScreensaverProcess, 0, len(s.processes))
	for _, process := range s.processes {
		if process.exited() {
			if s.config.IsDebug() {
				log.Printf("Screensaver on %s (PID %d) is no longer running, dropping it",
					process.Output, process.PID)
			}
			continue
		}
		stillRunning = append(stillRunning, process)
	}

	// Update processes list to only include running processes
	s.processes = stillRunning

	return len(s.processes) > 0
}

// exited reports whether the process has exited and been reaped already.
func (p ScreensaverProcess) exited() bool {
	if p.done == nil {
		// No reaper goroutine, so fall back to a signal 0 probe.
		return syscall.Kill(p.PID, 0) != nil
	}

	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// waitExit waits up to timeout for the process to exit. It returns false if
// the process is still running when the timeout elapses.
func (p ScreensaverProcess) waitExit(timeout time.Duration) bool {
	if p.done == nil {
		// No reaper goroutine: reap here, once.
		reaped := make(chan struct{})
		go func() {
			p.Cmd.Wait()
			close(reaped)
		}()

		select {
		case <-reaped:
			return true
		case <-time.After(timeout):
			return false
		}
	}

	select {
	case <-p.done:
		return true
	case <-time.After(timeout):
		return false
	}
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


