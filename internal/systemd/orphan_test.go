package systemd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// spawnAs starts sh under a symlink named name, so its argv[0] is that name,
// with extra as the arguments after the script. The script does not end in a
// bare command, so the shell stays the process instead of exec'ing sleep.
func spawnAs(t *testing.T, name string, extra ...string) *exec.Cmd {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	link := filepath.Join(t.TempDir(), name)
	if err := os.Symlink(sh, link); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(link, append([]string{"-c", "sleep 300; :"}, extra...)...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to spawn %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return cmd
}

// spawnFakeScreensaver starts a process whose argv is laid out like a real
// screensaver instance: kitty, then --class and the class, then the display.
func spawnFakeScreensaver(t *testing.T) *exec.Cmd {
	t.Helper()
	return spawnAs(t, "kitty", "--class", "sysc-walls-screensaver", "/usr/local/bin/sysc-walls-display")
}

// readCmdline returns a process's command line with NUL separators turned into
// spaces.
func readCmdline(t *testing.T, pid int) string {
	t.Helper()
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(strings.TrimRight(string(data), "\x00"), "\x00", " ")
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
	// And a shell or grep whose command line mentions the screensaver, which
	// the old pkill -f pattern matched and killed (audit sysc-2365).
	bystanders := []*exec.Cmd{
		spawnAs(t, "kitty", "--class", "my-regular-term"),
		spawnAs(t, "grep", "kitty --start-as=fullscreen --class sysc-walls-screensaver"),
		spawnAs(t, "zsh", "--class", "sysc-walls-screensaver"),
	}
	for _, b := range bystanders {
		if !processRunning(b.Process.Pid) {
			t.Fatalf("precondition failed: bystander %d is not running", b.Process.Pid)
		}
	}

	NewSystemD(config.NewConfig()).CleanupOrphans()
	time.Sleep(300 * time.Millisecond)

	for _, b := range bystanders {
		if !processRunning(b.Process.Pid) {
			t.Errorf("CleanupOrphans() killed PID %d (%q), an unrelated process", b.Process.Pid, readCmdline(t, b.Process.Pid))
		}
	}
}

func TestIsScreensaverArgv(t *testing.T) {
	for _, tc := range []struct {
		argv []string
		want bool
	}{
		{[]string{"kitty", "--start-as=fullscreen", "--class", "sysc-walls-screensaver", "/usr/local/bin/sysc-walls-display"}, true},
		{[]string{"/usr/bin/kitty", "--class=sysc-walls-screensaver", "x"}, true},
		{[]string{"xterm", "--class", "sysc-walls-screensaver", "x"}, true},
		{[]string{"kitty", "--class", "my-regular-term", "x"}, false},
		{[]string{"kitty", "--class"}, false},
		{[]string{"grep", "kitty.*--class.*sysc-walls-screensaver"}, false},
		{[]string{"zsh", "-c", "pkill -f kitty --class sysc-walls-screensaver"}, false},
		{[]string{"kitty --class sysc-walls-screensaver /usr/local/bin/sysc-walls-display"}, false},
		{[]string{"vim", "--class", "sysc-walls-screensaver", "x"}, false},
	} {
		if got := isScreensaverArgv(tc.argv); got != tc.want {
			t.Errorf("isScreensaverArgv(%q) = %v, want %v", tc.argv, got, tc.want)
		}
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
