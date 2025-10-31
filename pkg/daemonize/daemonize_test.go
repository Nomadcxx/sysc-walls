package daemonize

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewDaemon tests daemon creation
func TestNewDaemon(t *testing.T) {
	d := NewDaemon("test-daemon")

	if d == nil {
		t.Fatal("NewDaemon() returned nil")
	}

	if d.name != "test-daemon" {
		t.Errorf("Daemon name = %q, want test-daemon", d.name)
	}

	if d.pid != -1 {
		t.Errorf("Daemon PID = %d, want -1", d.pid)
	}
}

// TestPidFile tests PID file path
func TestPidFile(t *testing.T) {
	d := NewDaemon("test-daemon")
	d.pidFile = "/var/run/test-daemon.pid"

	pidFile := d.PidFile()
	if pidFile != "/var/run/test-daemon.pid" {
		t.Errorf("PidFile() = %q, want /var/run/test-daemon.pid", pidFile)
	}
}

// TestPid tests PID accessor
func TestPid(t *testing.T) {
	d := NewDaemon("test-daemon")

	if d.Pid() != -1 {
		t.Errorf("Pid() = %d, want -1", d.Pid())
	}

	d.pid = 1234
	if d.Pid() != 1234 {
		t.Errorf("Pid() = %d, want 1234", d.Pid())
	}
}

// TestCreatePidFile tests PID file creation
func TestCreatePidFile(t *testing.T) {
	// Skip if not running as root - createPidFile hardcodes /var/run
	if os.Geteuid() != 0 {
		t.Skip("Skipping test - requires root to write to /var/run")
	}

	d := NewDaemon("test-daemon-unit")

	err := d.createPidFile()
	if err != nil {
		t.Fatalf("createPidFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(d.pidFile); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// Verify content
	content, err := os.ReadFile(d.pidFile)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	// Just verify it's not empty
	if len(content) == 0 {
		t.Error("PID file is empty")
	}

	// Clean up
	os.Remove(d.pidFile)
}

// TestCreatePidFileExists tests handling of existing PID file
func TestCreatePidFileExists(t *testing.T) {
	// Skip if not running as root
	if os.Geteuid() != 0 {
		t.Skip("Skipping test - requires root to write to /var/run")
	}

	d := NewDaemon("test-daemon-exist")

	// Create a stale PID file (for non-existent process)
	pidPath := "/var/run/test-daemon-exist.pid"
	err := os.WriteFile(pidPath, []byte("99999"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}
	defer os.Remove(pidPath)

	d.pidFile = pidPath

	// Should succeed by removing stale PID file
	err = d.createPidFile()
	if err != nil {
		t.Errorf("createPidFile() should succeed with stale PID, got error: %v", err)
	}
}

// TestIsProcessRunning tests process existence check
func TestIsProcessRunning(t *testing.T) {
	// Current process should be running
	currentPID := os.Getpid()
	if !isProcessRunning(currentPID) {
		t.Error("isProcessRunning(currentPID) = false, want true")
	}

	// PID 99999 is very unlikely to exist
	if isProcessRunning(99999) {
		t.Log("Warning: PID 99999 exists on this system (test may be unreliable)")
	}
}

// TestContainsFlag tests flag detection
func TestContainsFlag(t *testing.T) {
	tests := []struct {
		args     []string
		flag     string
		expected bool
	}{
		{[]string{"-a", "-b", "-c"}, "-b", true},
		{[]string{"-a", "-b", "-c"}, "-d", false},
		{[]string{"--daemon"}, "--daemon", true},
		{[]string{"--daemon"}, "-daemon", false},
		{[]string{}, "--daemon", false},
		{[]string{"-start", "--daemon"}, "--daemon", true},
	}

	for _, tt := range tests {
		result := containsFlag(tt.args, tt.flag)
		if result != tt.expected {
			t.Errorf("containsFlag(%v, %q) = %v, want %v", tt.args, tt.flag, result, tt.expected)
		}
	}
}

// TestIsDaemon tests daemon detection
func TestIsDaemon(t *testing.T) {
	// This is difficult to test properly without actually daemonizing
	// Just verify the function doesn't panic
	result := isDaemon()
	t.Logf("isDaemon() = %v", result)
}

// TestRemovePidFile tests PID file removal
func TestRemovePidFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Create PID file
	err := os.WriteFile(pidPath, []byte("1234"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	d := NewDaemon("test-daemon")
	d.pidFile = pidPath

	// Remove it
	err = d.removePidFile()
	if err != nil {
		t.Errorf("removePidFile() error = %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file still exists after removePidFile()")
	}
}
