package systemd

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// spawnFakeScreensaver starts a process whose command line looks like a real
// screensaver instance: "kitty ... --class ... sysc-walls-screensaver".
//
// The class name is assembled from two halves inside the shell so that this
// helper's own command line cannot match the sweep pattern — otherwise pkill
// would match the helper and the test would pass for the wrong reason.
func spawnFakeScreensaver(t *testing.T) *exec.Cmd {
	t.Helper()

	script := `exec -a "kitty --start-as=fullscreen --class sysc-walls-scr""eensaver /usr/local/bin/sysc-walls-display" sleep 300`

	cmd := exec.Command("bash", "-c", script)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to spawn fake screensaver: %v", err)
	}

	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// Give the shell time to exec, so the spoofed argv is in place.
	time.Sleep(150 * time.Millisecond)

	return cmd
}

// processRunning reports whether pid is a live process.
//
// It deliberately does not use syscall.Kill(pid, 0): that succeeds for a
// zombie, so it would report a killed-but-unreaped child as still running.
// These helpers spawn children that nothing reaps until t.Cleanup, so the
// distinction matters. A zombie has exited, which is what the sweep produces.
func processRunning(pid int) bool {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false // gone entirely
	}

	// Fields after the (comm) field: state is the first one. comm can contain
	// spaces and parentheses, so find the closing paren first.
	s := string(data)
	i := strings.LastIndex(s, ")")
	if i < 0 {
		return false
	}
	fields := strings.Fields(s[i+1:])
	if len(fields) == 0 {
		return false
	}
	return fields[0] != "Z"
}

// TestCleanupOrphans_KillsUntrackedScreensaver covers the reported bug: a
// screensaver that outlived its daemon is invisible to a fresh daemon's process
// list, so without an explicit sweep it survives and stacks up behind a second
// screensaver.
func TestCleanupOrphans_KillsUntrackedScreensaver(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	orphan := spawnFakeScreensaver(t)
	orphanPID := orphan.Process.Pid

	if !processRunning(orphanPID) {
		t.Fatalf("precondition failed: fake screensaver %d is not running", orphanPID)
	}

	// A fresh daemon has tracked nothing, which is exactly why it cannot
	// discover this process on its own.
	s := NewSystemD(config.NewConfig())
	if n := s.GetProcessCount(); n != 0 {
		t.Fatalf("precondition failed: fresh daemon tracks %d processes, want 0", n)
	}

	s.CleanupOrphans()

	// pkill is best-effort; give it a moment to take effect.
	deadline := time.Now().Add(2 * time.Second)
	for processRunning(orphanPID) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if processRunning(orphanPID) {
		t.Errorf("CleanupOrphans() left an untracked screensaver (PID %d) running — it would stay fullscreen on the user's screen", orphanPID)
	}
}

// TestCleanupOrphans_LeavesUnrelatedProcessesAlone guards the sweep against
// becoming a foot-gun: it must only match the screensaver class, not arbitrary
// kitty windows the user happens to have open.
func TestCleanupOrphans_LeavesUnrelatedProcessesAlone(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	// A kitty window with a different class: same terminal, different app.
	bystander := exec.Command("bash", "-c", `exec -a "kitty --class my-regular-term" sleep 300`)
	if err := bystander.Start(); err != nil {
		t.Fatalf("failed to spawn bystander: %v", err)
	}
	t.Cleanup(func() {
		_ = bystander.Process.Kill()
		_ = bystander.Wait()
	})
	time.Sleep(150 * time.Millisecond)

	if !processRunning(bystander.Process.Pid) {
		t.Fatalf("precondition failed: bystander is not running")
	}

	NewSystemD(config.NewConfig()).CleanupOrphans()
	time.Sleep(300 * time.Millisecond)

	if !processRunning(bystander.Process.Pid) {
		t.Errorf("CleanupOrphans() killed PID %d, an unrelated kitty window", bystander.Process.Pid)
	}
}

// TestStopScreensaver_TrackedPathIsUnchanged confirms the refactor kept the
// existing behaviour: killing tracked processes still goes through the process
// group, and a healthy tracked child is reaped rather than left behind.
func TestStopScreensaver_TrackedPathIsUnchanged(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	s := NewSystemD(config.NewConfig())
	if err := s.LaunchScreensaver("sleep", []string{"300"}, "DP-1"); err != nil {
		t.Fatalf("LaunchScreensaver: %v", err)
	}
	if n := s.GetProcessCount(); n != 1 {
		t.Fatalf("tracked %d processes, want 1", n)
	}

	if err := s.StopScreensaver(); err != nil {
		t.Errorf("StopScreensaver returned an error: %v", err)
	}

	if n := s.GetProcessCount(); n != 0 {
		t.Errorf("GetProcessCount() = %d after stop, want 0", n)
	}
	if s.IsRunning() {
		t.Error("IsRunning() = true after stop, want false")
	}
}
