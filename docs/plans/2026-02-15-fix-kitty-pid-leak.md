# Fix Kitty PID Leak on Screensaver Dismiss

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix kitty terminal processes leaking (not being properly killed/reaped) when the screensaver is dismissed via mouse/keyboard activity.

**Architecture:** Three bugs in `internal/systemd/systemd.go` compound to cause PID leaks: (1) `Wait()` is fire-and-forget in a goroutine, so killed processes are never reaped, (2) no process group management so child processes of kitty survive, (3) `IsRunning()` uses `os.Signal(nil)` instead of `syscall.Signal(0)`. Fix all three, add SIGTERM-before-SIGKILL graceful shutdown, and add unit tests.

**Tech Stack:** Go 1.24.2, `os/exec`, `syscall`

**Files involved:**
- `internal/systemd/systemd.go` — the only file with bugs to fix
- `internal/systemd/systemd_test.go` — new test file (none exists today)
- `cmd/daemon/main.go` — minor: remove redundant pkill fallback (systemd.go handles it)

---

### Task 1: Create test file with failing tests for StopScreensaver

**Files:**
- Create: `internal/systemd/systemd_test.go`

**Context for the implementing agent:**
- `internal/systemd/systemd.go` contains `SystemD` struct managing `[]ScreensaverProcess`
- `SystemD` is created via `NewSystemD(cfg *config.Config)` — it needs a config
- `config.NewConfig()` returns a default config (no file needed)
- `ScreensaverProcess` has `PID int`, `Cmd *exec.Cmd`, `Output string`
- Current `StopScreensaver()` calls `Process.Kill()` then `go func() { cmd.Wait() }()` — fire-and-forget
- Current `LaunchScreensaver()` does NOT set `cmd.SysProcAttr` — no process group isolation
- Current `IsRunning()` uses `os.Signal(nil)` — wrong, should use `syscall.Signal(0)`

**Step 1: Write failing tests**

Create `internal/systemd/systemd_test.go` with these tests:

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/nomadx/Documents/sysc-screen && go test -v ./internal/systemd/ -count=1`
Expected: Multiple test failures — `TestStopScreensaver_ReapsProcesses` (zombie PID survives), `TestLaunchScreensaver_SetsProcessGroup` (nil SysProcAttr), `TestStopScreensaver_GracefulBeforeForce` (no SIGTERM sent). `TestStopScreensaver_SynchronousReturn` may pass coincidentally since `Kill()` is fast.

---

### Task 2: Fix `LaunchScreensaver` — Add process group isolation

**Files:**
- Modify: `internal/systemd/systemd.go` — `LaunchScreensaver()` method

**Context:**
- Add `import "syscall"` if not present
- Set `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` before `cmd.Start()`
- This creates a new process group for kitty, so killing the group kills all children

**Step 1: Add syscall import and SysProcAttr**

In `LaunchScreensaver()`, after `cmd := exec.Command(terminal, args...)` and before `cmd.Start()`, add:

```go
	// Create new process group so we can kill all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
```

Add `"syscall"` to the import block if not already present.

**Step 2: Run the process group test**

Run: `cd /home/nomadx/Documents/sysc-screen && go test -v ./internal/systemd/ -run TestLaunchScreensaver_SetsProcessGroup -count=1`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/systemd/systemd.go
git commit -m "fix: set process group on kitty launch for proper child cleanup"
```

---

### Task 3: Fix `StopScreensaver` — Synchronous SIGTERM→SIGKILL with process group kill

**Files:**
- Modify: `internal/systemd/systemd.go` — `StopScreensaver()` method

**Context:**
- Current code: `process.Cmd.Process.Kill()` (SIGKILL, no process group, no wait)
- New approach:
  1. Send SIGTERM to the **process group** (negative PID): `syscall.Kill(-pid, syscall.SIGTERM)`
  2. Wait up to 2 seconds for graceful exit
  3. If still alive, SIGKILL the process group: `syscall.Kill(-pid, syscall.SIGKILL)`
  4. Call `cmd.Wait()` **synchronously** (not in goroutine) to reap the zombie
  5. THEN clear the process list
- Add `"time"` to imports if not present

**Step 1: Replace StopScreensaver implementation**

Replace the entire `StopScreensaver()` method with:

```go
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
		// Fallback: try pkill
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
```

**Step 2: Ensure `"time"` is in imports**

Add `"time"` to the import block if not already present.

**Step 3: Run reap and synchronous tests**

Run: `cd /home/nomadx/Documents/sysc-screen && go test -v ./internal/systemd/ -run "TestStopScreensaver|TestMultipleLaunchStopCycles" -count=1`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/systemd/systemd.go
git commit -m "fix: synchronous process group kill with SIGTERM→SIGKILL escalation"
```

---

### Task 4: Fix `IsRunning` — Use proper signal-0 check

**Files:**
- Modify: `internal/systemd/systemd.go` — `IsRunning()` method

**Context:**
- Current: `process.Cmd.Process.Signal(os.Signal(nil))` — broken
- Fix: `syscall.Kill(process.PID, 0)` — proper POSIX liveness check via signal 0
- Also need to handle the case where process was Wait()'d — after Wait(), the PID is reaped and Signal() panics
- Remove `"os"` from imports if no longer used elsewhere

**Step 1: Replace IsRunning implementation**

```go
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
```

**Step 2: Clean up imports**

Check if `"os"` is still used elsewhere in the file. If not, remove it. `"syscall"` should already be present from Task 2.

**Step 3: Run IsRunning test**

Run: `cd /home/nomadx/Documents/sysc-screen && go test -v ./internal/systemd/ -run TestIsRunning -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/systemd/systemd.go
git commit -m "fix: use syscall.Signal(0) for accurate process liveness check"
```

---

### Task 5: Clean up daemon's redundant pkill fallback

**Files:**
- Modify: `cmd/daemon/main.go` — `StopScreensaver()` method on `Daemon` struct

**Context:**
- `systemd.StopScreensaver()` now handles pkill fallback internally
- The daemon's `StopScreensaver()` has its OWN pkill fallback that duplicates this
- Simplify: just call `d.systemD.StopScreensaver()` and log errors
- This also prevents double-pkill calls

**Step 1: Simplify daemon StopScreensaver**

Replace the `Daemon.StopScreensaver()` method (around line 400+ in main.go) with:

```go
// StopScreensaver stops the screensaver
func (d *Daemon) StopScreensaver() {
	if d.debug {
		log.Println("StopScreensaver called")
	}

	if err := d.systemD.StopScreensaver(); err != nil {
		log.Printf("StopScreensaver error: %v", err)
	}

	if d.debug {
		log.Println("StopScreensaver finished")
	}
}
```

**Step 2: Verify compilation**

Run: `cd /home/nomadx/Documents/sysc-screen && go vet ./cmd/daemon/ 2>&1; go build ./cmd/daemon/ 2>&1`
Note: This may fail due to missing `pkg/` packages — that's a pre-existing issue, not caused by our changes. Verify our specific file has no syntax/type errors via:
Run: `cd /home/nomadx/Documents/sysc-screen && go vet ./internal/systemd/`
Expected: Clean

**Step 3: Commit**

```bash
git add cmd/daemon/main.go
git commit -m "refactor: remove redundant pkill fallback from daemon, systemd handles it"
```

---

### Task 6: Run full test suite and verify

**Step 1: Run all systemd tests**

Run: `cd /home/nomadx/Documents/sysc-screen && go test -v -race ./internal/systemd/ -count=1`
Expected: All 6 tests PASS, no race conditions detected

**Step 2: Run all project tests that compile**

Run: `cd /home/nomadx/Documents/sysc-screen && go test ./internal/... -count=1`
Expected: All PASS (config, animations, systemd packages)

**Step 3: Run go vet on changed files**

Run: `cd /home/nomadx/Documents/sysc-screen && go vet ./internal/systemd/ ./cmd/daemon/`
Expected: Clean

**Step 4: Final commit if any fixups needed, otherwise done**

---

## Summary of Changes

| Bug | Root Cause | Fix | File |
|-----|-----------|-----|------|
| Zombie PIDs | `Wait()` in fire-and-forget goroutine | Synchronous `Wait()` after kill | systemd.go |
| Orphaned children | No `Setpgid` on kitty process | `SysProcAttr{Setpgid: true}` + process group kill | systemd.go |
| False `IsRunning()` | `os.Signal(nil)` not valid | `syscall.Kill(pid, 0)` | systemd.go |
| No graceful shutdown | Direct `SIGKILL` | SIGTERM → 2s wait → SIGKILL | systemd.go |
| Double pkill | Daemon + SystemD both pkill | Remove daemon's redundant fallback | main.go |
