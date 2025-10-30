// daemonize.go - Daemonization utilities
package daemonize

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Daemon represents a daemonized process
type Daemon struct {
	name    string
	pid     int
	pidFile string
}

// NewDaemon creates a new daemon instance
func NewDaemon(name string) *Daemon {
	return &Daemon{
		name: name,
		pid:  -1,
	}
}

// PidFile returns the PID file path
func (d *Daemon) PidFile() string {
	return d.pidFile
}

// Pid returns the process ID
func (d *Daemon) Pid() int {
	return d.pid
}

// isDaemon checks if the current process is already a daemon
func isDaemon() bool {
	return os.Getppid() == 1
}

// Daemonize starts the process as a daemon
func (d *Daemon) Daemonize() error {
	// Check if we're already a daemon
	if isDaemon() {
		return fmt.Errorf("process is already a daemon")
	}

	// Create PID file
	if err := d.createPidFile(); err != nil {
		return fmt.Errorf("failed to create PID file: %w", err)
	}

	// Command to re-execute ourselves with --daemon flag
	// This is the standard way to daemonize a Go program
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := os.Args
	if len(args) > 0 {
		// Remove the first argument (program name)
		args = args[1:]
	}

	// Add --daemon flag if not present
	if !containsFlag(args, "--daemon") {
		args = append([]string{"--daemon"}, args...)
	}

	// Start the process in a new session and with redirected file descriptors
	cmd := exec.Command(executable, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:     true, // Create a new session
		Setpgid:    true, // Create a new process group
		Credential: nil,  // No credentials change
	}

	// Redirect file descriptors
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Exit the parent process
	os.Exit(0)

	return nil
}

// createPidFile creates a PID file with the current process ID
func (d *Daemon) createPidFile() error {
	// Determine PID file location
	d.pidFile = filepath.Join("/var/run", fmt.Sprintf("%s.pid", d.name))

	// Try to create the PID file
	file, err := os.OpenFile(d.pidFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		// Check if the PID file already exists
		if os.IsExist(err) {
			// Read the PID from the existing file
			content, readErr := os.ReadFile(d.pidFile)
			if readErr != nil {
				return fmt.Errorf("failed to read PID file: %w", readErr)
			}

			// Parse the PID
			pid, parseErr := strconv.Atoi(string(content))
			if parseErr != nil {
				return fmt.Errorf("invalid PID in file: %w", parseErr)
			}

			// Check if the process is running
			if isProcessRunning(pid) {
				return fmt.Errorf("process already running with PID %d", pid)
			}

			// Remove the stale PID file
			os.Remove(d.pidFile)

			// Try again to create the PID file
			file, err = os.OpenFile(d.pidFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
			if err != nil {
				return fmt.Errorf("failed to create PID file: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create PID file: %w", err)
		}
	}

	// Write the PID to the file
	pid := os.Getpid()
	_, err = file.WriteString(strconv.Itoa(pid))
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Close the file
	file.Close()

	return nil
}

// removePidFile removes the PID file
func (d *Daemon) removePidFile() error {
	return os.Remove(d.pidFile)
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// Send a signal to the process to check if it's running
	// Signal 0 doesn't actually send anything, it just checks if the process exists
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// Stop stops the daemon process
func (d *Daemon) Stop() error {
	// Check if PID file exists
	if _, err := os.Stat(d.pidFile); os.IsNotExist(err) {
		return fmt.Errorf("PID file not found, daemon may not be running")
	}

	// Read the PID from the file
	content, err := os.ReadFile(d.pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Parse the PID
	pid, parseErr := strconv.Atoi(string(content))
	if parseErr != nil {
		return fmt.Errorf("invalid PID in file: %w", parseErr)
	}

	// Check if the process is running
	if !isProcessRunning(pid) {
		// Process not running, remove the PID file
		os.Remove(d.pidFile)
		return nil
	}

	// Send TERM signal to gracefully stop the process
	err = syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to send TERM signal: %w", err)
	}

	// Wait for the process to exit
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		if !isProcessRunning(pid) {
			break
		}
	}

	// If process still running, force kill
	if isProcessRunning(pid) {
		syscall.Kill(pid, syscall.SIGKILL)

		// Wait for the process to exit
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			if !isProcessRunning(pid) {
				break
			}
		}
	}

	// Remove the PID file
	os.Remove(d.pidFile)

	return nil
}

// containsFlag checks if a slice of strings contains a specific flag
func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}
