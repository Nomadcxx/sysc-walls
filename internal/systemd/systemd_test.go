package systemd

import (
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// helper: create a SystemD with default config
func newTestSystemD() *SystemD {
	cfg := config.NewConfig()
	return NewSystemD(cfg)
}

// helper: launch a long-running subprocess that we can kill
// Uses "sleep 300" as a stand-in for kitty
func launchSleepProcess(t *testing.T, s *SystemD) {
	t.Helper()
	err := s.LaunchScreensaver("sleep", []string{"300"}, "test-output")
	if err != nil {
		t.Fatalf("Failed to launch test process: %v", err)
	}
}

func TestStopScreensaver_ReapsProcesses(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := newTestSystemD()
	launchSleepProcess(t, s)

	// Get the PID before stopping
	pids, err := s.GetPIDs()
	if err != nil {
		t.Fatalf("Failed to get PIDs: %v", err)
	}
	if len(pids) != 1 {
		t.Fatalf("Expected 1 PID, got %d", len(pids))
	}
	pid := pids[0]

	// Stop should kill AND reap
	err = s.StopScreensaver()
	if err != nil {
		t.Fatalf("StopScreensaver failed: %v", err)
	}

	// After StopScreensaver returns, the process must be fully dead
	// (not a zombie). Give a brief moment for cleanup.
	time.Sleep(100 * time.Millisecond)

	// signal(0) should fail on a reaped process
	err = syscall.Kill(pid, 0)
	if err == nil {
		t.Errorf("PID %d still exists after StopScreensaver — process was not reaped", pid)
	}
}

func TestStopScreensaver_SynchronousReturn(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := newTestSystemD()
	launchSleepProcess(t, s)

	err := s.StopScreensaver()
	if err != nil {
		t.Fatalf("StopScreensaver failed: %v", err)
	}

	// Process list must be empty AFTER wait completes
	if s.GetProcessCount() != 0 {
		t.Errorf("Process list not cleared after StopScreensaver")
	}

	// Calling StopScreensaver again should be a no-op (not error)
	err = s.StopScreensaver()
	if err != nil {
		t.Errorf("Second StopScreensaver call should not error, got: %v", err)
	}
}

func TestLaunchScreensaver_SetsProcessGroup(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := newTestSystemD()
	launchSleepProcess(t, s)

	// Verify process group was set (Setpgid)
	s.mu.Lock()
	proc := s.processes[0]
	s.mu.Unlock()

	if proc.Cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr not set — process group isolation missing")
	}
	if !proc.Cmd.SysProcAttr.Setpgid {
		t.Error("Setpgid not set — child processes won't be killed with parent")
	}

	// Cleanup
	s.StopScreensaver()
}

func TestIsRunning_AccurateAfterKill(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := newTestSystemD()
	launchSleepProcess(t, s)

	if !s.IsRunning() {
		t.Fatal("Expected IsRunning=true after launch")
	}

	s.StopScreensaver()

	// After synchronous stop, IsRunning must be false
	if s.IsRunning() {
		t.Error("Expected IsRunning=false after StopScreensaver")
	}
}

func TestStopScreensaver_GracefulBeforeForce(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := newTestSystemD()

	// Launch a process that handles SIGTERM (bash trap)
	cmd := exec.Command("bash", "-c", "trap 'exit 0' TERM; sleep 300")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	s.mu.Lock()
	s.processes = append(s.processes, ScreensaverProcess{
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Output: "test-graceful",
	})
	s.mu.Unlock()

	start := time.Now()
	err := s.StopScreensaver()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("StopScreensaver failed: %v", err)
	}

	// Should complete quickly (SIGTERM handled) — not wait for full timeout
	if elapsed > 3*time.Second {
		t.Errorf("StopScreensaver took too long (%v) — SIGTERM may not have been sent", elapsed)
	}
}

func TestMultipleLaunchStopCycles(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := newTestSystemD()

	// Simulate 5 idle→active cycles
	for i := 0; i < 5; i++ {
		launchSleepProcess(t, s)

		if !s.IsRunning() {
			t.Fatalf("Cycle %d: Expected IsRunning=true after launch", i)
		}

		err := s.StopScreensaver()
		if err != nil {
			t.Fatalf("Cycle %d: StopScreensaver failed: %v", i, err)
		}

		if s.IsRunning() {
			t.Fatalf("Cycle %d: Expected IsRunning=false after stop", i)
		}
	}

	// No zombie processes should exist — verified by the reap test
}