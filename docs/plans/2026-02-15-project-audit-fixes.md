# Plan: Project Audit Fixes

**Date**: 2026-02-15
**Branch**: `fix/project-audit` (from `fix/kitty-pid-leak`)
**Scope**: Bug fixes, dead code cleanup, CLI client completion
**Principle**: Conservative — no unnecessary refactoring, minimal blast radius

---

## Task 1: Fix Data Race in `pkg/idle/idle.go`

**Problem**: `lastActive` field is written from multiple goroutines without synchronization. Race between `MarkActive()` (line ~451), `Start()` (line ~56), and Wayland `onResume` callback (line ~118).

**File**: `pkg/idle/idle.go`

**Changes**:
1. Add `mu sync.Mutex` field to `IdleDetector` struct
2. Lock/unlock around ALL writes to `lastActive`:
   - `Start()` initial assignment
   - `MarkActive()` 
   - `startWaylandIdleDetection` onResume callback (if it writes `lastActive` directly)
3. Lock/unlock around ALL reads of `lastActive` (check if any exist)
4. Do NOT change any logic — mutex only

**Verification**:
- `go test -race ./pkg/idle/` passes
- `go test -race ./...` passes (no regressions)

**Must NOT**: Change any idle detection logic, timings, or channel behavior.

---

## Task 2: Fix Hyprland Fullscreen Tiling

**Problem**: Kitty windows tile instead of fullscreening on Hyprland. Used to work, likely broke due to Hyprland update changing how `--start-as=fullscreen` is handled.

**File**: `internal/compositor/hyprland.go`

**Changes** (PENDING — awaiting librarian research):
- Most likely fix: Add dynamic window rule via `hyprctl keyword windowrule` before launching kitty, targeting `class:sysc-walls-screensaver`
- Alternative: Dispatch `fullscreen 1` after each kitty launch with a short delay
- Ensure rules are cleaned up in a `RemoveWindowRules()` method called during StopScreensaver

**Verification**:
- Unit test: verify `hyprctl keyword windowrule` command is constructed correctly
- Integration: requires Hyprland environment (manual test)

**Must NOT**: Change Niri or Sway compositor implementations. Changes scoped to Hyprland only.

---

## Task 3: Remove Dead Code

**Files**: `internal/systemd/systemd.go`, `cmd/daemon/main.go`

**Changes**:
1. **Delete `parseCommand()`** from `internal/systemd/systemd.go` — never called
2. **Delete `startActivityMonitoring()`** from `cmd/daemon/main.go` — no-op goroutine (100ms ticker doing nothing)
3. **Remove the call to `startActivityMonitoring()`** from wherever it's invoked in daemon

**Verification**:
- `go build ./cmd/daemon/` succeeds
- `go vet ./cmd/daemon/` clean
- `go test ./internal/systemd/` passes

**Must NOT**: Remove any code that IS referenced, even if it looks unused at first glance.

---

## Task 4: Simplify Daemon StopScreensaver pkill Fallback

**Problem**: Daemon's `StopScreensaver()` has a redundant pkill fallback that duplicates logic already in `systemd.StopScreensaver()`.

**File**: `cmd/daemon/main.go`

**Changes**:
1. Review whether daemon-level pkill is truly redundant now that systemd.StopScreensaver() does synchronous SIGTERM→SIGKILL with process group kill
2. If redundant: remove daemon pkill fallback, rely on systemd layer
3. If NOT redundant (edge cases): add comment explaining when it fires and why

**Verification**:
- `go build ./cmd/daemon/` succeeds
- Existing tests pass

**Must NOT**: Change `internal/systemd/systemd.go` — it was already fixed.

---

## Task 5: Implement CLI Client Commands

**File**: `cmd/client/main.go`

**Changes**:
1. **`run` command**: Use `os/exec` to launch `sysc-walls-display` with args from config. Foreground execution, Ctrl+C to stop.
2. **`test` command**: Same as `run` but with a 10-second timeout (preview mode).
3. **`start` command**: `exec.Command("systemctl", "--user", "start", "sysc-walls.service").CombinedOutput()` — report result.
4. **`stop` command**: `exec.Command("systemctl", "--user", "stop", "sysc-walls.service").CombinedOutput()` — report result.
5. **`status` command**: Enhance to also run `systemctl --user status sysc-walls.service` and show daemon PID.

**Verification**:
- `go build ./cmd/client/` succeeds
- `go vet ./cmd/client/` clean
- Manual: `sysc-walls-client status` shows service state

**Must NOT**: Add complex daemon communication (IPC, sockets). Keep it simple — exec systemctl commands.

---

## Task 6: Clean Up Stale Backups

**Changes**:
1. Delete `src/sysc-walls-1.0.0/` directory
2. Delete `sysc-Go.bak/` directory
3. Add to `.gitignore`:
   ```
   src/
   *.bak/
   sysc-Go.bak/
   ```
4. Verify `pkg/multi_display/` — if truly unused, leave for now (not breaking anything)

**Verification**:
- `go build ./...` still works
- `go test ./...` still passes
- `du -sh .` shows significant size reduction

**Must NOT**: Delete anything tracked by git without confirming it's not imported. Only delete untracked directories.

---

## Execution Order

1. **Task 1** (data race) — independent, no risk to other code
2. **Task 2** (Hyprland) — independent, compositor-scoped
3. **Task 3** (dead code) — safe removal
4. **Task 4** (pkill simplification) — depends on understanding Task 3 results
5. **Task 5** (CLI client) — independent, additive
6. **Task 6** (cleanup) — last, purely subtractive

Tasks 1, 2, 5 are fully independent and can be parallelized.

---

## Risk Assessment

| Task | Risk | Mitigation |
|------|------|------------|
| 1 (data race) | Low — adding mutex only | Verify no deadlock with -race |
| 2 (Hyprland) | Medium — compositor-specific, can't test without Hyprland | Window rules are well-documented, add cleanup |
| 3 (dead code) | Low — grep-verified unused | Build verification |
| 4 (pkill) | Low-Medium — fallback removal | Keep if any edge case found |
| 5 (CLI) | Low — additive only | Build verification |
| 6 (cleanup) | Low — untracked files only | Verify build after deletion |
