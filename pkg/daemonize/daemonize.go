// daemonize.go - Daemonization utilities
package daemonize

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ChildEnvVar marks the process Daemonize re-executed.
//
// The re-executed child re-enters main with the same arguments, so without a
// marker it would call Daemonize again: the second generation either sees
// itself already detached and fatals, or forks a third. The flag on the
// command line cannot express this, because the child legitimately needs the
// same flags the launcher was given.
const ChildEnvVar = "SYSC_WALLS_DAEMON_CHILD"

// IsChild reports whether this process is the daemon that Daemonize
// re-executed, rather than the process that started it. The caller uses it to
// skip daemonising a second time and to know that it is the process that
// should own the PID file.
func IsChild() bool {
	return os.Getenv(ChildEnvVar) == "1"
}

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

// Daemonize re-executes this program as a detached daemon and exits the
// launching process. It never returns on success.
//
// It deliberately does not write the PID file. The PID that would be recorded
// is the launcher's, and the launcher exits on the very next line, so the file
// would name a process that is already gone. The child calls WritePidFile
// instead, once it is the daemon.
func (d *Daemon) Daemonize() error {
	if IsChild() {
		return fmt.Errorf("refusing to daemonize: this process is already the re-executed daemon child (%s is set)", ChildEnvVar)
	}

	// Check if we're already a daemon
	if isDaemon() {
		return fmt.Errorf("process is already a daemon")
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
	cmd.SysProcAttr = daemonProcAttr

	// The environment marker is what tells the child not to daemonize again.
	cmd.Env = append(os.Environ(), ChildEnvVar+"=1")

	// Detach from the launcher's working directory so the daemon is not tied
	// to a directory the user may later remove.
	cmd.Dir = "/"

	// Redirect file descriptors. Leaving them nil makes os/exec open
	// /dev/null, so the daemon never holds the launching terminal open.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Exit the parent process. There is no return: callers must treat a nil
	// error from Daemonize as unreachable.
	os.Exit(0)

	// Unreachable; present only because the compiler cannot see that
	// os.Exit does not return.
	return nil
}

// WritePidFile records this process as the running daemon.
//
// The caller must be the re-executed child — see IsChild. Recording the
// launcher's PID would leave a file naming a process that has already exited,
// which makes the singleton check never fire and makes Stop signal nothing.
func (d *Daemon) WritePidFile() error {
	return d.createPidFile()
}

// CleanupPidFile removes the PID file. The daemon should defer this so a
// graceful exit does not leave a stale file behind for the next start to
// argue with.
func (d *Daemon) CleanupPidFile() error {
	return d.removePidFile()
}

// createPidFile creates a PID file with the current process ID
func (d *Daemon) createPidFile() error {
	// Determine PID file location - use user runtime directory
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	d.pidFile = filepath.Join(runtimeDir, fmt.Sprintf("%s.pid", d.name))

	// /run/user/<uid> is created by logind, not by us, and XDG_RUNTIME_DIR may
	// point somewhere that has not been made yet. Without this the failure
	// surfaces as a bare "no such file or directory" from OpenFile.
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		return fmt.Errorf("failed to create runtime directory %s: %w", runtimeDir, err)
	}

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

			// Remove the stale PID file. A failure here is worth reporting:
			// the recreate below would otherwise fail with a confusing
			// "file exists" that looks like a second daemon.
			if rmErr := os.Remove(d.pidFile); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("failed to remove stale PID file %s: %w", d.pidFile, rmErr)
			}

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

	// A PID file records a number, and the operating system recycles those.
	// Between a crash and the next start the recorded PID can name a
	// completely unrelated process, and signalling it would be worse than
	// failing to stop anything. Confirm it is still our binary.
	if ours, checkable := isSameExecutable(pid); checkable && !ours {
		return fmt.Errorf("PID %d from %s is no longer %s; refusing to signal it", pid, d.pidFile, d.name)
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

// isSameExecutable reports whether pid is running the same executable as this
// process. The second return value is false when the question cannot be
// answered — a non-Linux system, or a process we may not inspect — in which
// case the caller should not treat a negative answer as a refusal.
func isSameExecutable(pid int) (same bool, checkable bool) {
	readLink := func(p string) (string, error) {
		target, err := os.Readlink(p)
		if err != nil {
			return "", err
		}
		// A deleted binary is reported as "<path> (deleted)".
		return strings.TrimSuffix(target, " (deleted)"), nil
	}

	self, err := readLink("/proc/self/exe")
	if err != nil {
		return false, false
	}
	other, err := readLink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return false, false
	}

	return self == other, true
}
