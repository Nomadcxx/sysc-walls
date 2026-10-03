package daemonize

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// reexecHelper is the stand-in for the real daemon: it daemonizes, and the
// re-executed child writes the PID file and reports what it found.
//
// The loop in #37 lived in this shape — a launcher that re-executes itself
// with the same arguments, and a child that re-enters the same branch — so the
// test reproduces that shape rather than calling the functions directly.
const reexecHelper = `
package main

import (
	"os"
	"time"

	"github.com/Nomadcxx/sysc-walls/pkg/daemonize"
)

func main() {
	if !daemonize.IsChild() {
		d := daemonize.NewDaemon("reexec-probe")
		if err := d.Daemonize(); err != nil {
			os.Exit(1)
		}
		// Daemonize exits the process, so reaching here means the loop is
		// still broken.
		os.Exit(3)
	}

	// The child. Its stdio is /dev/null, so the PID file is the only channel
	// back to the test.
	d := daemonize.NewDaemon("reexec-probe")
	if err := d.WritePidFile(); err != nil {
		os.Exit(2)
	}

	time.Sleep(3 * time.Second)
	d.CleanupPidFile()
}
`

// buildReexecHelper compiles a tiny program that exercises the launcher and
// child halves of Daemonize in separate processes.
func buildReexecHelper(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(reexecHelper), 0o644); err != nil {
		t.Fatalf("writing helper source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module syscwtest\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Point the helper's module at the repository under test, so it compiles
	// the real package rather than a copy.
	pkgDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolving package dir: %v", err)
	}
	modFile := "module syscwtest\n\ngo 1.24\n\n" +
		"require github.com/Nomadcxx/sysc-walls v0.0.0\n\n" +
		"replace github.com/Nomadcxx/sysc-walls => " + pkgDir + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modFile), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// The helper only needs the package under test, but the module graph still
	// has to be resolved before building.
	bin := filepath.Join(dir, "probe")
	build := exec.Command("go", "mod", "tidy")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}

	// nosetsid: the helper links this package, so the tag is how it picks up
	// the SysProcAttr without Setsid. Everything under test — the env marker,
	// the launcher/child split, and which process writes the PID file — is
	// independent of session detachment.
	build = exec.Command("go", "build", "-tags", "nosetsid", "-o", bin, ".")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building helper: %v\n%s", err, out)
	}

	return bin
}

// TestDaemonizeChildRunsOnceAndOwnsThePidFile is the end-to-end shape of
// #37: the launcher re-executes, the child must not daemonize again, and the
// PID file must name the child rather than the launcher.
func TestDaemonizeChildRunsOnceAndOwnsThePidFile(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a helper binary")
	}

	bin := buildReexecHelper(t)
	runtimeDir := t.TempDir()

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "XDG_RUNTIME_DIR="+runtimeDir)

	var stderr strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the launcher: %v", err)
	}

	// The launcher must exit promptly; the daemon has moved on to the child.
	launcherPID := cmd.Process.Pid
	launcherDone := make(chan error, 1)
	go func() { launcherDone <- cmd.Wait() }()

	select {
	case err := <-launcherDone:
		if err != nil {
			t.Fatalf("launcher exited with %v", err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("the launcher never exited; Daemonize did not take over")
	}

	// The PID file must name the child, not the launcher. Recording the
	// launcher's PID is the original defect: it exits on the next line, so the
	// file would name a process that is already gone.
	pidFile := filepath.Join(runtimeDir, "reexec-probe.pid")

	deadline := time.Now().Add(10 * time.Second)
	var recorded int
	for {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			recorded, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the child never wrote a usable PID file: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	if recorded == launcherPID {
		t.Fatalf("the PID file records the launcher (%d), which exits immediately; it must name the child", recorded)
	}
	if !isProcessRunning(recorded) {
		t.Fatalf("the PID file records %d, which is not a live process", recorded)
	}

	// And the child removes it on the way out, so a graceful stop does not
	// leave a stale file for the next start to argue with.
	deadline = time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(pidFile); os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the child left its PID file behind after exiting")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
