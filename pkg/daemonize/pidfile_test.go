package daemonize

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// withRuntimeDir points the package at a temporary XDG_RUNTIME_DIR so the
// tests never touch a real /run/user entry.
func withRuntimeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	return dir
}

// TestWritePidFileRecordsTheRunningProcess is the core of #37: the file has to
// name the process that holds the idle loop.
//
// It used to be written by the launcher, which exits on the next line, so the
// recorded PID was always dead by the time anyone read it. That defeated both
// the singleton check and Stop.
func TestWritePidFileRecordsTheRunningProcess(t *testing.T) {
	runtimeDir := withRuntimeDir(t)

	d := NewDaemon("test-write-pid")
	if err := d.WritePidFile(); err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}

	want := filepath.Join(runtimeDir, "test-write-pid.pid")
	if d.PidFile() != want {
		t.Errorf("PidFile() = %q, want %q", d.PidFile(), want)
	}

	got := readPid(t, d.PidFile())
	if got != os.Getpid() {
		t.Errorf("PID file contains %d, want %d (the running process)", got, os.Getpid())
	}

	// A recorded PID has to look alive, or the singleton check never fires.
	if !isProcessRunning(got) {
		t.Errorf("isProcessRunning(%d) = false for the process that just wrote it", got)
	}
}

// TestWritePidFileRefusesASecondDaemon: the singleton guarantee only works if
// a live holder of the file blocks the next start.
func TestWritePidFileRefusesASecondDaemon(t *testing.T) {
	withRuntimeDir(t)

	first := NewDaemon("test-singleton")
	if err := first.WritePidFile(); err != nil {
		t.Fatalf("first WritePidFile() error = %v", err)
	}

	second := NewDaemon("test-singleton")
	err := second.WritePidFile()
	if err == nil {
		// Our own PID is in the file and is alive, so this must be refused.
		t.Fatal("second WritePidFile() error = nil, want a refusal while the first is running")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("second WritePidFile() error = %v, want it to mention that a process is already running", err)
	}
}

// TestWritePidFileTakesOverFromAStaleFile: a file left by a crashed daemon
// must not block the next start.
func TestWritePidFileTakesOverFromAStaleFile(t *testing.T) {
	runtimeDir := withRuntimeDir(t)

	// A PID that is not running. 1 is init under most pid namespaces, so use a
	// high number that nothing will be using.
	stale := filepath.Join(runtimeDir, "test-stale.pid")
	if err := os.WriteFile(stale, []byte("4194303"), 0644); err != nil {
		t.Fatalf("seeding stale PID file: %v", err)
	}

	d := NewDaemon("test-stale")
	if err := d.WritePidFile(); err != nil {
		t.Errorf("WritePidFile() over a stale file error = %v, want it to take over", err)
	}

	if got := readPid(t, d.PidFile()); got != os.Getpid() {
		t.Errorf("PID file contains %d, want %d after taking over", got, os.Getpid())
	}
}

// TestWritePidFileRejectsAGarbagePIDFile: a truncated or corrupt file should be
// reported, not silently treated as stale.
func TestWritePidFileRejectsAGarbagePIDFile(t *testing.T) {
	runtimeDir := withRuntimeDir(t)

	bad := filepath.Join(runtimeDir, "test-garbage.pid")
	if err := os.WriteFile(bad, []byte("not-a-pid"), 0644); err != nil {
		t.Fatalf("seeding PID file: %v", err)
	}

	d := NewDaemon("test-garbage")
	if err := d.WritePidFile(); err == nil {
		t.Error("WritePidFile() error = nil for a corrupt PID file, want an error")
	}
}

// TestCleanupPidFileRemovesIt covers the other half of #37: the file used to
// survive every exit, making the stale-file path the normal case.
func TestCleanupPidFileRemovesIt(t *testing.T) {
	withRuntimeDir(t)

	d := NewDaemon("test-cleanup")
	if err := d.WritePidFile(); err != nil {
		t.Fatalf("WritePidFile() error = %v", err)
	}
	if _, err := os.Stat(d.PidFile()); err != nil {
		t.Fatalf("PID file missing before cleanup: %v", err)
	}

	if err := d.CleanupPidFile(); err != nil {
		t.Errorf("CleanupPidFile() error = %v", err)
	}
	if _, err := os.Stat(d.PidFile()); !os.IsNotExist(err) {
		t.Error("PID file still present after CleanupPidFile()")
	}
}

// TestIsChild reflects the environment marker that breaks the re-exec loop.
func TestIsChild(t *testing.T) {
	t.Setenv(ChildEnvVar, "")
	if IsChild() {
		t.Error("IsChild() = true with no marker set")
	}

	t.Setenv(ChildEnvVar, "1")
	if !IsChild() {
		t.Error("IsChild() = false with the marker set")
	}
}

// TestDaemonizeRefusesToRunInTheChild is the loop this all exists to break.
// Without the guard the re-executed child calls Daemonize again.
func TestDaemonizeRefusesToRunInTheChild(t *testing.T) {
	withRuntimeDir(t)
	t.Setenv(ChildEnvVar, "1")

	d := NewDaemon("test-child-refusal")
	err := d.Daemonize()
	if err == nil {
		t.Fatal("Daemonize() error = nil in the re-executed child; it would fork another generation")
	}
	if !strings.Contains(err.Error(), ChildEnvVar) {
		t.Errorf("Daemonize() error = %v, want it to name %s so the cause is obvious", err, ChildEnvVar)
	}
}

// TestIsSameExecutable covers the recycled-PID guard: signalling whatever
// happens to hold the recorded PID could hit an unrelated process.
func TestIsSameExecutable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test: reads /proc/<pid>/exe")
	}

	same, checkable := isSameExecutable(os.Getpid())
	if !checkable {
		t.Skip("/proc/self/exe not readable")
	}
	if !same {
		t.Error("isSameExecutable(self) = false, want true")
	}

	// PID 1 exists but is not this executable.
	if other := findOtherProcess(); other > 0 {
		same, checkable := isSameExecutable(other)
		if checkable && same {
			t.Errorf("isSameExecutable(%d) = true for a different process, want false", other)
		}
	}
}

// TestIsSameExecutableIsNotCheckableForAMissingProcess: a PID that is gone has
// no /proc entry, and that must not be reported as "not ours" — the caller
// handles liveness separately.
func TestIsSameExecutableIsNotCheckableForAMissingProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	_, checkable := isSameExecutable(4194303)
	if checkable {
		t.Error("isSameExecutable(dead PID) reported checkable, want it to decline")
	}
}

// TestStopRefusesAnUnrelatedPID is the user-visible consequence: a recycled PID
// must not be signalled.
func TestStopRefusesAnUnrelatedPID(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test: reads /proc/<pid>/exe")
	}

	runtimeDir := withRuntimeDir(t)

	// A long-lived process that is definitely not our binary.
	victim := exec.Command("sleep", "30")
	if err := victim.Start(); err != nil {
		t.Fatalf("starting the unrelated process: %v", err)
	}
	t.Cleanup(func() {
		_ = victim.Process.Kill()
		_ = victim.Wait()
	})

	pidPath := filepath.Join(runtimeDir, "test-stop-refuse.pid")
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(victim.Process.Pid)), 0644); err != nil {
		t.Fatalf("seeding PID file: %v", err)
	}

	d := NewDaemon("test-stop-refuse")
	d.pidFile = pidPath

	if err := d.Stop(); err == nil {
		t.Error("Stop() error = nil for a PID belonging to another program, want a refusal")
	}

	// The unrelated process must still be alive.
	time.Sleep(200 * time.Millisecond)
	if victim.ProcessState != nil {
		t.Error("Stop() killed an unrelated process")
	}
	if !isProcessRunning(victim.Process.Pid) {
		t.Error("the unrelated process is gone; Stop() signalled something it should not have")
	}
}

func readPid(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading PID file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parsing PID file %q: %v", data, err)
	}
	return pid
}

// findOtherProcess returns the PID of some running process that is not this
// one, or -1.
func findOtherProcess() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return -1
	}
	self := os.Getpid()
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}
		if isProcessRunning(pid) {
			return pid
		}
	}
	return -1
}
