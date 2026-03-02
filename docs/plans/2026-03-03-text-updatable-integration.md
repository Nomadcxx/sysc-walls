# Plan: TextUpdatable Integration in sysc-walls

> **Sub-skill**: `superpowers/writing-plans` -> `superpowers/executing-plans`
> **Date**: 2026-03-03
> **Branch**: `feat/text-updatable-integration` (from `master`)
> **Upstream dependency**: sysc-Go v1.0.3 (PRs #48 + #49 merged)

## Goal

Wire up sysc-Go's `TextUpdatable` support so text-based effects render a live clock as animation negative space, while preserving existing overlay behavior for non-text effects.

User requirements:
- User can toggle datetime display (`animation.datetime = true/false`)
- User can set datetime position (`datetime.position = top/center/bottom`)
- Clock updates every 1 second by default and is configurable (`datetime.interval`)

## Architecture Decision

Use a local interface in `internal/animations`:
- Add `TextUpdatable` in `internal/animations/animations.go`
- Implement `SetText(string)` on text-capable wrappers in `internal/animations/optimized.go`
- In display loop, type-assert against local interface (no sysc-Go interface leakage)

Display behavior:
- Text-updatable effects: update rendered text every `datetime.interval`
- Non-text effects: keep current `overlayDateTime()` path (no behavior regression)

---

## Step 1: Update dependency and minimum version

**Files**: `go.mod`, `internal/config/config.go`

### Changes
1. Bump `github.com/Nomadcxx/sysc-Go` from `v1.0.2` -> `v1.0.3`
2. Change `MinimumSyscGoVersion` from `"1.0.1"` -> `"1.0.3"`

### Verify
```bash
go build ./...
```

---

## Step 2: Add TextUpdatable interface and compile-correct tests (TDD red)

**Files**: `internal/animations/animations.go`, `internal/animations/text_updatable_test.go`

### Step 2a: Add interface in animations package

```go
type TextUpdatable interface {
    SetText(text string)
}

func IsTextUpdatable(anim Animation) bool {
    _, ok := anim.(TextUpdatable)
    return ok
}
```

### Step 2b: Add failing tests (must compile and fail before implementation)

Create `internal/animations/text_updatable_test.go` with factory calls that handle errors correctly:

```go
func TestTextEffectsImplementTextUpdatable(t *testing.T) {
    effects := []string{"fire-text", "matrix-art", "rain-art", "beam-text", "decrypt", "pour", "print", "blackhole", "ring-text"}
    for _, effect := range effects {
        t.Run(effect, func(t *testing.T) {
            anim, err := CreateAnimationWithText(effect, 80, 24, "rama", "TEST")
            if err != nil {
                t.Fatalf("CreateAnimationWithText(%s) error: %v", effect, err)
            }
            tu, ok := anim.(TextUpdatable)
            if !ok {
                t.Fatalf("%s does not implement TextUpdatable", effect)
            }
            tu.SetText("HELLO WORLD")
        })
    }
}

func TestNonTextEffectsDoNotImplementTextUpdatable(t *testing.T) {
    effects := []string{"matrix", "fire", "fireworks", "rain", "beams", "aquarium"}
    for _, effect := range effects {
        t.Run(effect, func(t *testing.T) {
            anim, err := CreateAnimation(effect, 80, 24, "rama")
            if err != nil {
                t.Fatalf("CreateAnimation(%s) error: %v", effect, err)
            }
            if _, ok := anim.(TextUpdatable); ok {
                t.Fatalf("%s should not implement TextUpdatable", effect)
            }
        })
    }
}
```

Add a resize-safety test to catch text rollback:

```go
func TestSetTextSurvivesResizeForStatefulWrappers(t *testing.T) {
    anim, err := CreateAnimationWithText("matrix-art", 80, 24, "rama", "INIT")
    if err != nil {
        t.Fatal(err)
    }
    tu := anim.(TextUpdatable)
    tu.SetText("UPDATED")
    anim.Resize(100, 30)

    m, ok := anim.(*optimizedMatrixArt)
    if !ok {
        t.Fatalf("expected *optimizedMatrixArt, got %T", anim)
    }
    if m.text != "UPDATED" {
        t.Fatalf("text state not updated, got %q", m.text)
    }
}
```

### Run (expect failures before Step 3)
```bash
go test -v ./internal/animations/ -run TextUpdatable
go test -v ./internal/animations/ -run TestSetTextSurvivesResizeForStatefulWrappers
```

---

## Step 3: Implement SetText on wrappers (green)

**File**: `internal/animations/optimized.go`

### Changes
1. Add `SetText(string)` to all text-capable wrappers.
2. For wrappers that store `text` and rebuild effect on resize, update both state and inner effect.

Required state updates:
- `optimizedBeamText`
- `optimizedMatrixArt`
- `optimizedRainArt`
- `optimizedBlackhole`
- `optimizedRingText`

Example pattern for stateful wrappers:

```go
func (m *optimizedMatrixArt) SetText(text string) {
    m.text = text
    m.effect.SetText(text)
}
```

Example pattern for stateless wrappers:

```go
func (f *optimizedFireText) SetText(text string) { f.effect.SetText(text) }
```

### Run
```bash
go test -v ./internal/animations/ -run TextUpdatable
go test -v ./internal/animations/ -run TestSetTextSurvivesResizeForStatefulWrappers
```

Expected: pass.

---

## Step 4: Wire display loop without regressing non-text datetime

**File**: `cmd/display/main.go`

### Step 4a: Add datetime interval flag

Add:
- `-datetime-interval` (default `1s`) using `flag.Duration`
- Validation: `> 0`, otherwise exit with clear error

### Step 4b: Add helper to push clock text into TextUpdatable animation

Add helper like:

```go
func setAnimationClockText(anim animations.Animation, position string, width, height int) bool
```

Behavior:
- Type-assert `anim.(animations.TextUpdatable)`
- Build clock text from `clock.GetDateTime()` + `clock.RenderClock()`
- Apply vertical placement (`top`/`center`/`bottom`) by adding newline padding before/after clock block relative to terminal height
- `SetText(clockText)`
- Return `true` if applied, else `false`

### Step 4c: Runtime flow

- Keep existing render ticker (`50ms`)
- Add `clockTicker := time.NewTicker(*datetimeInterval)`
- On startup: if `--datetime`, call `setAnimationClockText(...)` once so first frame is populated
- On `clockTicker`: refresh clock text for text-updatable animations
- On `SIGWINCH`: after `anim.Resize(...)`, refresh clock text immediately when datetime enabled

### Step 4d: Preserve overlay fallback for non-text effects

Do **not** remove `overlayDateTime()` in this change.
- If animation is not `TextUpdatable`, continue applying `overlayDateTime(...)` in render path
- This avoids removing existing datetime capability for non-text effects

### Manual verification
```bash
go build -o bin/sysc-walls-display ./cmd/display/
./bin/sysc-walls-display -effect fire-text -theme rama -datetime -datetime-position top -datetime-interval 1s
./bin/sysc-walls-display -effect matrix -theme rama -datetime -datetime-position bottom -datetime-interval 1s
```

Expected:
- `fire-text`: live clock in animation text region, updates every second
- `matrix`: existing overlay behavior still works

---

## Step 5: Config and command wiring

**Files**: `internal/config/config.go`, `internal/config/config_test.go`

### Step 5a: Add interval to config model

Add field in `Config`:
- `datetimeInterval time.Duration` (default `1 * time.Second`)

Add parser key:
- `datetime.interval` via `parseDuration`
- Require `> 0`, else warn and keep default

Add getter:
- `GetDatetimeInterval() time.Duration`

### Step 5b: Ensure default and saved config include datetime settings

In both `createDefaultConfig()` and `SaveToFile()` write:

```ini
[animation]
datetime = false

[datetime]
position = bottom
interval = 1s
```

### Step 5c: Update `GetScreensaverCommand()` behavior

If datetime enabled:
- always append `--datetime`
- append `--datetime-position <value>`
- append `--datetime-interval <duration>`

Remove old compatibility gate that disabled datetime on text-based effects.

### Step 5d: Tests

Add/adjust tests for:
- parsing `datetime.position` and `datetime.interval`
- invalid interval fallback (`0s`, negative, malformed)
- saved config contains `[datetime]` section and `animation.datetime`
- command generation includes datetime flags for both text and non-text effects when enabled

### Run
```bash
go test -v ./internal/config/
```

---

## Step 6: Documentation updates

**Files**: `README.md`, optionally `TROUBLESHOOTING.md`

Update docs for:
- `animation.datetime`
- `datetime.position`
- `datetime.interval`
- CLI flag `-datetime-interval`
- Behavior split: text-updatable effects use animation text path; non-text effects use overlay fallback

Note: commit only files relevant to this feature; do not blindly bundle unrelated untracked paths.

---

## Step 7: Final verification

```bash
go fmt ./...
go vet ./...
go test -v -count=1 ./...
go test -race -count=1 ./...
go build ./cmd/display/ ./cmd/daemon/ ./cmd/client/
```

Manual:
```bash
./bin/sysc-walls-display -effect matrix-art -theme rama -datetime -datetime-position center -datetime-interval 1s
./bin/sysc-walls-display -effect fire -theme rama -datetime -datetime-position top -datetime-interval 2s
```

---

## File Change Summary

| File | Action | Step |
|------|--------|------|
| `go.mod` | Bump sysc-Go to v1.0.3 | 1 |
| `internal/config/config.go` | Minimum version, datetime interval field/parser/writer/command flags | 1, 5 |
| `internal/config/config_test.go` | Add datetime parsing and command-generation coverage | 5 |
| `internal/animations/animations.go` | Add `TextUpdatable` + helper | 2 |
| `internal/animations/text_updatable_test.go` | New compile-correct TDD tests | 2 |
| `internal/animations/optimized.go` | Add `SetText`; keep state sync for resize-safe wrappers | 3 |
| `cmd/display/main.go` | Add text-clock updater, ticker, interval flag, fallback overlay preservation | 4 |
| `README.md` | Document datetime behavior and interval | 6 |

## Execution Notes

- Step 2 tests must fail before Step 3 changes (red -> green)
- Do not remove overlay path in this PR; keep regression risk low
- Handle resize explicitly for text-updatable mode to avoid stale placement
- Keep commits small and bisectable
