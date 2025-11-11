<div align="center">
  <img src="assets/logo.png" alt="sysc-walls">

  **A terminal screensaver, designed in Go and built for Wayland**

  [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
  [![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](go.mod)
</div>

---

<!-- Placeholder for showcase animation GIF -->
<div align="center">
  <img src="assets/showcase.gif" alt="sysc-walls showcase" width="800">
  <p><em>Showcase of available animations - coming soon</em></p>
</div>

---

Watches for idle activity on Wayland (or X11), then displays animations from [sysc-Go](https://github.com/Nomadcxx/sysc-Go) when you step away. Runs as a systemd service and supports multi-monitor setups across Niri, Hyprland, Sway, and other Wayland compositors.

## Install

**One-line install:**
```bash
curl -fsSL https://raw.githubusercontent.com/Nomadcxx/sysc-walls/master/install.sh | sudo bash
```

Or **manual install:**
```bash
git clone https://github.com/Nomadcxx/sysc-walls.git
cd sysc-walls
go run cmd/installer/main.go
```

The installer automatically:
- Clones the [sysc-Go](https://github.com/Nomadcxx/sysc-Go) animation library
- Builds all binaries (daemon, display, client)
- Installs to `/usr/local/bin`
- Sets up the systemd user service
- Imports `WAYLAND_DISPLAY` for compositor detection
- Backs up and updates your config with current defaults

**Test your installation first:**
```bash
sysc-walls-daemon -test              # Quick test
sysc-walls-daemon -test -debug       # Test with diagnostics
```

Then enable the service:
```bash
systemctl --user enable sysc-walls.service
systemctl --user start sysc-walls.service
```

The default idle timeout is 5 minutes.

## Configuration

Config lives at `~/.config/sysc-walls/daemon.conf`:

```ini
[idle]
timeout = 5m          # How long before screensaver kicks in
min_duration = 30s    # Minimum time screensaver runs

[animation]
effect = matrix-art   # Which animation to show
theme = rama          # Color scheme
cycle = false         # Rotate through effects

[daemon]
debug = false         # Enable detailed logging

[terminal]
kitty = true          # Use Kitty terminal (required)
fullscreen = true     # Launch fullscreen
```

**Available effects:**
`matrix`, `matrix-art`, `fire`, `fireworks`, `rain`, `rain-art`, `beams`, `beam-text`, `decrypt`, `pour`, `aquarium`, `print`, `ring-text`, `blackhole`

**Available themes:**
`rama`, `nord`, `dracula`, `gruvbox`, `tokyo-night`, `catppuccin`, `material`, `solarized`, `monochrome`, `eldritch`, `dark`, `trainsishardjob`

For detailed configuration options and troubleshooting, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## Building

**Quick build with installer (recommended):**
```bash
git clone https://github.com/Nomadcxx/sysc-walls.git
cd sysc-walls
go run cmd/installer/main.go
```

**Manual build:**
```bash
# Clone and build
git clone https://github.com/Nomadcxx/sysc-walls.git
cd sysc-walls
git clone https://github.com/Nomadcxx/sysc-Go.git

go build -o daemon ./cmd/daemon/
go build -o display ./cmd/display/
go build -o client ./cmd/client/

# Install
sudo cp daemon /usr/local/bin/sysc-walls-daemon
sudo cp display /usr/local/bin/sysc-walls-display
sudo cp client /usr/local/bin/sysc-walls-client
sudo chmod +x /usr/local/bin/sysc-walls-*

# Setup systemd
mkdir -p ~/.config/systemd/user
cp systemd/sysc-walls-user.service ~/.config/systemd/user/sysc-walls.service
systemctl --user daemon-reload
systemctl --user enable sysc-walls.service
systemctl --user start sysc-walls.service
```

**Dependencies:**
- Go 1.24+ ([install](https://go.dev/doc/install))
- Wayland development libraries for [CGO bindings](pkg/idle/)
- Kitty terminal

```bash
# Arch/Manjaro
sudo pacman -S go wayland kitty

# Debian/Ubuntu
sudo apt install golang libwayland-dev kitty

# Fedora
sudo dnf install golang wayland-devel kitty
```

## Architecture

sysc-walls consists of three components working together:

### 1. Daemon ([cmd/daemon/](cmd/daemon/))

The core idle detection service that runs continuously via systemd.

**Key responsibilities:**
- Monitors system idle time using Wayland protocols (or X11 fallback)
- Detects your compositor (Niri, Hyprland, Sway) for multi-monitor support
- Launches screensaver instances on all connected displays
- Terminates screensaver on keyboard/mouse activity

**Idle detection** ([pkg/idle/](pkg/idle/))
Uses [CGO bindings to libwayland-client](pkg/idle/) following the same approach as swayidle. This provides native integration with the `ext-idle-notify-v1` Wayland protocol for reliable idle detection across compositors. For X11 systems, falls back to `xprintidle`.

**Multi-monitor support** ([internal/compositor/](internal/compositor/))
Automatically detects your compositor and enumerates all displays. For each monitor, it focuses the output and launches a screensaver instance, ensuring wall-to-wall coverage across your entire setup.

### 2. Display ([cmd/display/](cmd/display/))

The animation renderer that runs in fullscreen Kitty terminal instances.

**Key responsibilities:**
- Renders animations from [sysc-Go](https://github.com/Nomadcxx/sysc-Go)
- Handles terminal sizing and fullscreen mode
- Applies color themes from configuration
- Supports text-based effects with ASCII art loading

**Animation integration** ([internal/animations/](internal/animations/))
Wraps sysc-Go effects with optimized rendering for terminal output. Implements text loading from `sysc-Go/assets/` for crystallization effects like matrix-art and rain-art.

### 3. Client ([cmd/client/](cmd/client/))

Optional CLI tool for managing the daemon and testing configurations.

**Not required for normal operation** - the daemon runs autonomously once started. Useful for quick tests and config validation.

### Configuration ([internal/config/](internal/config/))

Manages all settings from `~/.config/sysc-walls/daemon.conf`. Validates effects, themes, and timeouts. The installer updates this file on each run (with automatic backup to `daemon.conf.backup`).

### Build System

The project uses Go modules with a local `replace` directive for sysc-Go during development:

```go
replace github.com/Nomadcxx/sysc-Go => ./sysc-Go
```

The installer automatically clones sysc-Go to maintain this structure. For production releases, this will be replaced with a versioned GitHub module reference.

## Testing & Debugging

**Quick test:**
```bash
sysc-walls-daemon -test
```

**Test with full diagnostics:**
```bash
sysc-walls-daemon -test -debug
```

**Check service status:**
```bash
systemctl --user status sysc-walls.service
```

**View live logs:**
```bash
journalctl --user -u sysc-walls.service -f
```

**Enable debug logging permanently:**

Edit `~/.config/sysc-walls/daemon.conf`:
```ini
[daemon]
debug = true
```

For detailed troubleshooting, compositor-specific setup, and common issues, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## License

MIT - Do whatever you want with it.

## Credits

Built on top of [sysc-Go](https://github.com/Nomadcxx/sysc-Go) for animations. Inspired by classic terminal screensavers but designed for modern Wayland systems.
