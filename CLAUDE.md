# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Critical Rules

### Git Workflow
- **CRITICAL: NEVER add Co-Authored-By or attribute contributions to Claude/AI**
- **NEVER add "Generated with Claude Code" or similar attribution**
- Commits should be clean, simple, and human-like
- Brief commit messages focused on what changed
- Do NOT push automatically - always ask first
- Main branch: `master`

### Communication Style
- Do NOT make comments sound AI-produced
- No excessive validation or praise
- Be direct and technical
- Write documentation naturally, as a human developer would

## Memory Management with OpenMemory

**IMPORTANT**: Use OpenMemory MCP tools for persistent semantic memory to reduce token usage.

### When to Use OpenMemory

**DO use for:**
- ✅ Retrieving project architecture and patterns
- ✅ Querying user preferences and workflow rules
- ✅ Looking up technical constraints
- ✅ Storing discovered bugs/patterns during development

**DO NOT use for:**
- ❌ Reading actual source code (use Read tool)
- ❌ Checking git status (use Bash tool)

### OpenMemory Tools

```bash
# Query memory
mcp__openmemory__openmemory_query(query="idle detection", k=5)

# Store new knowledge
mcp__openmemory__openmemory_store(content="...", tags=["architecture", "sysc-walls"])

# Reinforce important memories
mcp__openmemory__openmemory_reinforce(id="...", boost=0.1)
```

## Project Overview

sysc-walls is a terminal screensaver daemon for Wayland compositors, written in Go. It monitors idle time via Wayland's ext-idle-notify-v1 protocol and launches animations from sysc-Go when the system is idle.

## Build Commands

```bash
# Build all binaries
make build

# Build specific component
go build -o bin/sysc-walls-daemon ./cmd/daemon/
go build -o bin/sysc-walls-display ./cmd/display/
go build -o bin/sysc-walls-client ./cmd/client/

# Run installer
go run cmd/installer/main.go

# Install system-wide
sudo cp bin/* /usr/local/bin/
```

## Architecture

### Core Components

1. **Daemon** (`cmd/daemon/`)
   - Systemd user service monitoring idle time
   - Wayland idle detection via CGO bindings (`pkg/idle/`)
   - Compositor detection (Niri, Hyprland, Sway)
   - Multi-monitor support (`internal/compositor/`)

2. **Display** (`cmd/display/`)
   - Renders sysc-Go animations in fullscreen Kitty
   - Effect wrapper with terminal sizing
   - Theme application and ASCII art loading
   - DateTime overlay support

3. **Client** (`cmd/client/`)
   - Optional CLI for testing (not needed for normal operation)

4. **Installer** (`cmd/installer/`)
   - Bubble Tea TUI for installation
   - Config validation and backup
   - Systemd service setup

### Key Packages

- **internal/config/** - INI config parsing, validation, settings
- **internal/compositor/** - Multi-monitor detection and management
- **internal/animations/** - Effect wrappers for sysc-Go
- **internal/systemd/** - Service management and process control
- **internal/clock/** - DateTime overlay rendering
- **internal/version/** - Version tracking
- **pkg/idle/** - CGO bindings for Wayland idle detection
- **pkg/utils/** - Terminal utilities

## Configuration

Config file: `~/.config/sysc-walls/daemon.conf`

```ini
[idle]
timeout = 5m
min_duration = 30s

[daemon]
debug = false

[animation]
effect = matrix-art
theme = rama
cycle = false
file = ~/.config/sysc-walls/ascii/SYSC.txt
datetime = false

[datetime]
position = bottom

[terminal]
kitty = true
fullscreen = true
```

### Effects

**Text-based** (require `-file` flag):
- fire-text, matrix-art, beam-text, rain-art, ring-text, blackhole

**Non-text** (standalone):
- matrix, fire, rain, fireworks, beams, aquarium

## Development Guidelines

### Testing

```bash
# Quick test
sysc-walls-daemon -test

# Test with debug output
sysc-walls-daemon -test -debug

# Demo mode (cycles through all effects)
sysc-walls-daemon -demo

# Direct display test
sysc-walls-display --effect fire --theme dracula --fullscreen
```

### Demo Mode

Demo mode uses `sysc-walls-demo` window class (not `sysc-walls-screensaver`) to avoid conflict with running service's pkill.

### Security

- Config validation prevents command injection
- Safe path validation for file arguments
- Effect and theme names validated against registry

### File Locations

- Config: `~/.config/sysc-walls/daemon.conf`
- ASCII art: `~/.config/sysc-walls/ascii/`
- Binaries: `/usr/local/bin/sysc-walls-*`
- Service: `~/.config/systemd/user/sysc-walls.service`
- sysc-Go: `./sysc-Go/` (local clone, replace directive in go.mod)

## Common Tasks

### Adding a New Effect

1. Implement in sysc-Go (`animations/` package)
2. Register in sysc-Go's effect registry
3. Display automatically picks it up via registry

### Adding a New Theme

1. Add to sysc-Go's theme registry
2. Display automatically picks it up

### Updating Config Options

1. Add field to Config struct in `internal/config/config.go`
2. Add getter/setter methods
3. Update `LoadFromFile()` to parse new option
4. Update default config template in installer

## Important Notes

### Dependencies

- Go 1.24+
- sysc-Go (local clone via replace directive)
- Wayland development libraries (for CGO)
- Kitty terminal

### Service Management

```bash
# Enable and start
systemctl --user enable sysc-walls.service
systemctl --user start sysc-walls.service

# Check status
systemctl --user status sysc-walls.service

# View logs
journalctl --user -u sysc-walls.service -f
```

### Troubleshooting

- See TROUBLESHOOTING.md for common issues
- Debug mode: `debug = true` in config or `-debug` flag
- Test mode verifies idle detection before enabling service
