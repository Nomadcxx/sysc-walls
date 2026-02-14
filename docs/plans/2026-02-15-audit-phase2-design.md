# Phase 2 Audit Fixes — Design & Implementation Plan

**Date**: 2026-02-15
**Scope**: 3 items from gap analysis not covered by Plan 1

---

## Design Decisions

### Decision 1: wayland_cgo.go Data Race — `atomic.Bool`

**Problem**: `initialized` field is written under `w.mu` in `Stop()` but read without locking in `Start()`. C global state (`wayland_cgo_init`/`wayland_cgo_cleanup`) races between test runs.

**Design**: Use `atomic.Bool` for `initialized`.

**Rationale**:
- Go 1.24.2 supports `sync/atomic.Bool` (available since 1.19)
- `initialized` is a simple flag — atomic is lighter than mutex for single-field access
- Keeps `w.mu` for compound operations in `Stop()` (cleanup + state reset)
- For C global state: the `globalDetectorMu` already exists and protects Go-side state. C functions themselves are single-threaded (Wayland protocol is inherently single-threaded per connection). The race is Go-side access to the `initialized` field, not C-side.
- Test isolation: ensure no `t.Parallel()` on wayland tests — C globals are process-level singletons

**Changes**:
- `initialized bool` → `initialized atomic.Bool`
- `Start()`: change `if w.initialized` → `if w.initialized.Load()`
- `Stop()`: keep `w.mu.Lock()` for compound operation, change `w.initialized = false` → `w.initialized.Store(false)`
- `NewWaylandCGODetector()`: after successful init, `w.initialized.Store(true)`
- Test file: remove any `t.Parallel()` calls if present

### Decision 2: Compositor Tests — Parse-Level + Interface Mock

**Problem**: Compositor implementations untested. All three parse CLI output.

**Design**: Two-tier testing.

**Tier 1 (this plan)**: Test `parseOutputs()` directly with sample data.
- Hyprland: sample `hyprctl monitors -j` JSON (single monitor, multi-monitor, focused/unfocused)
- Sway: sample `swaymsg -t get_outputs` JSON (active/inactive, focused, rect parsing)
- Niri: sample `niri msg outputs` text (regex parsing of connector names)
- Edge cases: empty input, malformed JSON, missing fields, zero-dimension outputs

**Tier 2 (future — when user loads Hyprland)**: Integration tests with real compositor.
- Live `hyprctl monitors -j` output capture
- Verify fullscreen window rules work
- Verify FocusOutput + launch sequence

**File structure**: `internal/compositor/compositor_test.go` — single test file covering all three parsers (they share the same Output struct).

### Decision 3: Animation Cycling — Display-Internal with Config

**Problem**: AnimationCycler scaffolded but not wired. User wants toggle + configurable duration.

**Design**: Display binary manages cycling internally.

**Rationale for display-internal (not daemon-managed)**:
- Daemon-managed cycling would require kill + relaunch kitty → visual flash, PID churn
- Display binary already runs a bubbletea event loop with `tea.Tick` — can add a cycle timer
- Hot-swap within bubbletea: the `Model` wraps an animation — swap the animation, call `Resize()`, continue
- No need to restart the TUI program — bubbletea supports dynamic model changes via `Update()`

**Config additions** (in `internal/config/config.go`):
```ini
[animation]
cycle = true              # existing, already parsed
cycle_interval = 5m       # NEW — how often to switch animations
```

**Implementation flow**:
1. Config: add `cycleInterval time.Duration` field, `GetCycleInterval()` getter, parse from `animation.cycle_interval`
2. Display binary `main()`:
   - If `cycle` enabled, get list of available effects, create `AnimationCycler`
   - Add a `tea.Tick` with `cycleInterval` duration
   - On tick: call `cycler.SwitchAnimation()`, get new effect name, create new animation model, hot-swap into TUI
3. The existing `AnimationCycler` API is sufficient — `GetCurrentAnimation()` returns effect name, display creates new animation from that name

**Risk mitigation**:
- Hot-swap is the riskier path but avoids the visual flash. If it causes issues, fallback is daemon-managed restart.
- The display binary already handles `Resize()` dynamically — hot-swap is similar.
- AnimationCycler already handles random/sequential ordering.

---

## Implementation Plan

### Prerequisites
- Plan 1 must be applied first (PID leak fixes, dead code removal, backup cleanup)
- Project must build clean: `go build ./...`
- Existing tests must pass: `go test ./...`

---

### Task 1: Fix wayland_cgo.go data race
**File**: `pkg/idle/wayland_cgo.go`
**Risk**: Low — single field type change, atomic operations are simpler than mutex
**Tests**: Run `go test -race ./pkg/idle/ -count=3` — race must not reproduce

#### Step 1a: Change `initialized` to `atomic.Bool`
```go
// BEFORE
type WaylandCGODetector struct {
    mu          sync.Mutex
    initialized bool
    // ...
}

// AFTER
import "sync/atomic"

type WaylandCGODetector struct {
    mu          sync.Mutex
    initialized atomic.Bool
    // ...
}
```

#### Step 1b: Update all read/write sites
Every `w.initialized = true` → `w.initialized.Store(true)`
Every `w.initialized = false` → `w.initialized.Store(false)`
Every `if w.initialized` → `if w.initialized.Load()`
Every `if !w.initialized` → `if !w.initialized.Load()`

#### Step 1c: Remove t.Parallel() from wayland tests if present
Check `pkg/idle/idle_test.go` and any wayland-specific test files.

#### Verification
```bash
go build ./pkg/idle/
go test -race ./pkg/idle/ -count=3
go vet ./pkg/idle/
```

---

### Task 2: Compositor parseOutputs() tests
**File**: `internal/compositor/compositor_test.go` (NEW)
**Risk**: Low — read-only tests, no production code changes
**Depends on**: Nothing

#### Step 2a: Hyprland parseOutputs tests
Table-driven tests with:
- Single monitor focused: `[{"name":"DP-1","width":2560,"height":1440,"x":0,"y":0,"focused":true}]`
- Multi-monitor, one focused: 2+ monitors, verify correct one marked focused
- No monitor focused: all `focused: false` — verify error or empty
- Empty JSON array: `[]`
- Malformed JSON: `invalid{json`
- Missing fields: `[{"name":"DP-1"}]` — verify defaults or error

#### Step 2b: Sway parseOutputs tests
Table-driven tests with:
- Active output with rect: `[{"name":"eDP-1","active":true,"focused":true,"rect":{"width":1920,"height":1080}}]`
- Inactive output filtered: verify Active:false outputs excluded
- Multi-monitor with one focused
- Empty/malformed JSON

#### Step 2c: Niri parseOutputs tests
Table-driven tests with:
- Sample `niri msg outputs` text (need to capture real format)
- Multiple outputs with connector names (DP-1, HDMI-A-1, eDP-1)
- Empty output
- Unexpected format

#### Step 2d: GetFocusedOutput logic tests
Each compositor's GetFocusedOutput iterates ListOutputs — test the selection logic with pre-built Output slices.

#### Verification
```bash
go test -v ./internal/compositor/
go vet ./internal/compositor/
```

---

### Task 3: Wire animation cycling into display binary
**Files**: `internal/config/config.go`, `cmd/display/main.go`, `internal/animations/animation_cycle.go`
**Risk**: Medium — modifies display binary's main loop
**Depends on**: Task 1, Task 2 (for clean test baseline)

#### Step 3a: Add cycleInterval to config
- Add `cycleInterval time.Duration` field to Config struct
- Default: 5 minutes (`5m`)
- Parse from `animation.cycle_interval` in LoadFromFile
- Add `GetCycleInterval() time.Duration` getter
- Add `SetCycleInterval(d time.Duration)` setter
- Add to SaveToFile output
- Write tests in config_test.go

#### Step 3b: Add available effects list function
The display binary needs to know which effects exist for cycling.
- Check if `internal/animations` already exposes an effect list
- If not, add `GetAvailableEffects() []string` 
- This should return all valid effect names that CreateAnimation accepts

#### Step 3c: Wire AnimationCycler into display binary
- In main(): if config.ShouldCycleAnimations() is true:
  - Get available effects list
  - Create AnimationCycler with config.GetCycleInterval()
  - Add a cycle tick message type to the bubbletea model
  - In Update(): on cycle tick, call SwitchAnimation(), create new animation, swap into model
  - Return next tick command to continue cycling
- If cycle is false: existing behavior unchanged (single effect)

#### Step 3d: Add test for config cycleInterval
- TestParseCycleInterval — valid durations (30s, 5m, 1h)
- TestParseCycleInterval — invalid input falls back to default
- TestParseCycleInterval — zero/negative treated as default

#### Verification
```bash
go build ./cmd/display/
go test -v ./internal/config/
go test -v ./internal/animations/
go vet ./...
```

---

### Task 4: Final verification
**Risk**: None — read-only

```bash
go build ./...
go test -race ./... -count=1
go vet ./...
```

All must pass. Pre-existing failures (if any) documented but not fixed.

---

## Out of Scope (deferred)
- cmd/daemon and cmd/display integration tests (user decision: skip for now)
- Hyprland fullscreen fix (covered in Plan 1 Task 2)
- CLI client commands (covered in Plan 1 Task 5)
- Remove pkg/multi_display (only in stale src/, covered by Plan 1 Task 6 backup deletion)
- Niri FocusOutput documentation (not needed — fully implemented)
