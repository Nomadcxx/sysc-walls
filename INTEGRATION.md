# sysc-Go ↔ sysc-walls Integration Architecture

This document describes the integration architecture between sysc-Go (animation library) and sysc-walls (screensaver daemon).

## Overview

sysc-walls integrates sysc-Go using **direct library integration** for optimal performance:
- **Zero IPC overhead** - Direct Go function calls
- **Type-safe** - Full compile-time checking
- **Automatic sync** - Registry-based effect management

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       sysc-Go                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  animations/registry.go                               │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │  EffectRegistry []EffectMetadata                │  │  │
│  │  │  - GetEffectNames()                             │  │  │
│  │  │  - GetTextBasedEffects()                        │  │  │
│  │  │  - IsTextBasedEffect(name)                      │  │  │
│  │  │  - GetLibraryVersion()                          │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│  animations/*.go (fire.go, matrix.go, etc.)                │
└─────────────────────────────────────────────────────────────┘
                             │
                             │ Direct import
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                     sysc-walls                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  internal/config/config.go                            │  │
│  │  - AvailableEffects = syscGo.GetEffectNames()        │  │
│  │  - CheckSyscGoVersion()                              │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  internal/animations/optimized.go                     │  │
│  │  - CreateOptimizedAnimationWithText()                │  │
│  │  - Maps effect name → sysc-Go constructor            │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  cmd/display/main.go                                  │  │
│  │  - isTextBasedEffect() = syscGo.IsTextBasedEffect()  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Key Features

### 1. Unified Effect Registry

**Location**: `sysc-Go/animations/registry.go`

The registry provides a single source of truth for all effects:

```go
type EffectMetadata struct {
    Name          string // e.g., "fire", "matrix"
    RequiresText  bool   // Whether effect needs text input
    Description   string // Brief description
    VersionAdded  string // Version when added
    Category      string // "particle", "text", "abstract", "scene"
}

var EffectRegistry = []EffectMetadata{
    {
        Name:         "fire",
        RequiresText: false,
        Description:  "Doom-style fire effect",
        VersionAdded: "1.0.0",
        Category:     "particle",
    },
    // ... more effects
}
```

**Benefits**:
- **Auto-sync**: sysc-walls automatically gets new effects
- **Type safety**: Compile-time verification
- **Metadata**: Rich information about each effect
- **No duplication**: Single source of truth

### 2. Version Compatibility

**Location**: `internal/config/config.go`

```go
const MinimumSyscGoVersion = "1.0.1"

func CheckSyscGoVersion() error {
    actualVersion := syscGo.GetLibraryVersion()
    if actualVersion < MinimumSyscGoVersion {
        return fmt.Errorf("version mismatch: found %s, requires >= %s",
            actualVersion, MinimumSyscGoVersion)
    }
    return nil
}
```

**Checked at**:
- Daemon startup (`cmd/daemon/main.go:94`)
- Automatically prevents incompatible versions

### 3. Automated Sync

**Location**: `scripts/sync-sysc-go.sh`

Syncs all animation code from main sysc-Go repository:

```bash
./scripts/sync-sysc-go.sh
```

**What it syncs**:
- `animations/*.go` - All effect implementations
- `cmd/` - Version information
- `tui/` - TUI components
- `go.mod`/`go.sum` - Dependencies
- Updates `MinimumSyscGoVersion` automatically

**Features**:
- ✅ Version checking
- ✅ Uncommitted changes warning
- ✅ Colorized output
- ✅ Summary statistics

## Workflow

### Adding a New Effect to sysc-Go

1. **Create effect in sysc-Go**:
   ```go
   // animations/myeffect.go
   type MyEffect struct { ... }
   func NewMyEffect(...) *MyEffect { ... }
   func (e *MyEffect) Update() { ... }
   func (e *MyEffect) Render() string { ... }
   func (e *MyEffect) Resize(w, h int) { ... }
   ```

2. **Add to registry**:
   ```go
   // animations/registry.go
   {
       Name:         "my-effect",
       RequiresText: false,
       Description:  "My awesome effect",
       VersionAdded: "1.0.2",
       Category:     "abstract",
   },
   ```

3. **Sync to sysc-walls**:
   ```bash
   cd /path/to/sysc-walls
   make sync
   ```

4. **Add wrapper in sysc-walls**:
   ```go
   // internal/animations/optimized.go
   case "my-effect":
       return newOptimizedMyEffect(width, height, palette)

   // ... implement wrapper
   type optimizedMyEffect struct {
       effect *syscGo.MyEffect
   }
   ```

5. **Build and test**:
   ```bash
   make build
   ./bin/sysc-walls-display --effect my-effect --theme nord
   ```

### Keeping sysc-walls Up to Date

#### Manual Sync
```bash
make sync     # Sync from main repo
make build    # Rebuild
make test     # Test effects
```

#### Pre-build Check
```bash
make safe-build  # Checks sync status before building
```

#### Check Version
```bash
make version     # Show version info
```

## Directory Structure

```
sysc-walls/
├── sysc-Go/               # Local copy of sysc-Go (managed by sync)
│   ├── animations/
│   │   ├── registry.go    # Effect registry (SOURCE OF TRUTH)
│   │   ├── fire.go        # Effect implementations
│   │   ├── matrix.go
│   │   └── ...
│   ├── cmd/
│   ├── tui/
│   └── go.mod
├── internal/
│   ├── config/
│   │   └── config.go      # Auto-generates from registry
│   └── animations/
│       └── optimized.go   # Wrappers for sysc-Go effects
├── cmd/
│   ├── display/
│   │   └── main.go        # Display component
│   └── daemon/
│       └── main.go        # Daemon with version checking
├── scripts/
│   └── sync-sysc-go.sh    # Automated sync script
├── Makefile               # Build targets
└── INTEGRATION.md         # This file
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build display and daemon binaries |
| `make clean` | Remove build artifacts |
| `make install` | Install to `/usr/local/bin` |
| `make sync` | Sync from main sysc-Go repo |
| `make check-sync` | Check if sync needed (fails if outdated) |
| `make test` | Run basic effect tests |
| `make version` | Show version information |
| `make safe-build` | Check sync + build |
| `make help` | Show help message |

## Version Compatibility Matrix

| sysc-walls | Required sysc-Go | Notes |
|-----------|------------------|-------|
| current   | >= 1.0.1        | fire-text, updated effects |

## Integration Points

1. **Effect Registry** (`sysc-Go/animations/registry.go`)
   - Single source of truth for effects
   - Metadata for each effect

2. **Config Auto-generation** (`internal/config/config.go`)
   - `AvailableEffects = syscGo.GetEffectNames()`
   - Auto-updates when registry changes

3. **Text-based Detection** (`cmd/display/main.go`)
   - `isTextBasedEffect() = syscGo.IsTextBasedEffect()`
   - No hardcoded lists

4. **Version Checking** (`cmd/daemon/main.go`)
   - Checked at startup
   - Prevents incompatible versions

5. **Effect Wrappers** (`internal/animations/optimized.go`)
   - Maps effect name → constructor
   - Implements Animation interface

## Testing

### Test Individual Effect
```bash
./bin/sysc-walls-display --effect fire-text \
    --theme nord \
    --file ~/art.txt
```

### Test All Effects
```bash
for effect in matrix fire rain fireworks beams aquarium; do
    echo "Testing: $effect"
    ./bin/sysc-walls-display --effect $effect --theme nord --no-clear &
    sleep 2
    pkill -f sysc-walls-display
done
```

### Verify Version
```bash
./bin/sysc-walls-daemon --help  # Check startup messages
make version                     # Show version info
```

## Benefits of This Architecture

### ✅ Automatic Synchronization
- New effects appear automatically after sync
- No manual list updates required
- Compile-time verification

### ✅ Version Safety
- Incompatible versions detected at startup
- Clear error messages
- Prevents runtime failures

### ✅ Maintainability
- Single source of truth (registry)
- No duplicate effect lists
- Easy to add new effects

### ✅ Performance
- Direct function calls (no IPC)
- Zero serialization overhead
- Native Go performance

### ✅ Developer Experience
- Simple sync workflow
- Makefile convenience targets
- Clear error messages

## Troubleshooting

### Version Mismatch Error
```
Error: sysc-Go version mismatch: found 1.0.0, requires >= 1.0.1
```
**Solution**: Run `make sync` to update sysc-Go

### Effect Not Found
```
Error: unknown animation effect: my-effect
```
**Solution**:
1. Check effect exists in `sysc-Go/animations/`
2. Verify effect in registry: `sysc-Go/animations/registry.go`
3. Add wrapper in `internal/animations/optimized.go`
4. Rebuild: `make build`

### Sync Script Fails
```
Error: Main sysc-Go repository not found
```
**Solution**: Update path in `scripts/sync-sysc-go.sh:13`

## Future Improvements

### Potential Enhancements
1. **CI/CD Integration**
   - Automated sync checks in GitHub Actions
   - Version compatibility tests
   - Effect regression tests

2. **Effect Hot-reloading**
   - Watch sysc-Go for changes
   - Auto-rebuild and reload

3. **Plugin System**
   - External effect plugins
   - Dynamic loading

4. **Performance Profiling**
   - Effect benchmarks
   - Memory usage tracking

5. **Effect Marketplace**
   - Share custom effects
   - Community contributions

## References

- **sysc-Go Repository**: `/home/nomadx/Documents/sysc-Go`
- **sysc-walls Repository**: `/home/nomadx/Documents/sysc-screen`
- **Effect Registry**: `sysc-Go/animations/registry.go`
- **Sync Script**: `scripts/sync-sysc-go.sh`
- **Makefile**: `Makefile`

---

**Last Updated**: 2025-01-13
**sysc-Go Version**: 1.0.1
**sysc-walls Version**: 1.0.0
