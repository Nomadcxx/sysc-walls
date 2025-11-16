<div align="center">
  <img src="assets/logo.png" alt="sysc-walls">

  **A terminal screensaver, designed in Go and built for Wayland**
</div>

---

<div align="center">
  <img src="assets/showcase.gif" alt="sysc-walls showcase" width="800">
</div>

---

Watches for idle activity on Wayland (or X11), then displays animations from [sysc-Go](https://github.com/Nomadcxx/sysc-Go) when you step away. Runs as a systemd service and supports multi-monitor setups across Niri, Hyprland, Sway, and other Wayland compositors.

## Install

**One-line install:**
```bash
curl -fsSL https://raw.githubusercontent.com/Nomadcxx/sysc-walls/master/install.sh | sudo bash
```
### Arch Linux (AUR)

```bash
yay -S sysc-walls
# or
paru -S sysc-walls
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
`matrix`, `matrix-art`, `fire`, `fire-text`, `fireworks`, `rain`, `rain-art`, `beams`, `beam-text`, `aquarium`, `ring-text`, `blackhole`

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

Three components:

### 1. Daemon ([cmd/daemon/](cmd/daemon/))

Systemd service that monitors idle time via Wayland's `ext-idle-notify-v1` protocol (X11 via xprintidle). Detects compositor (Niri/Hyprland/Sway), launches screensaver on all monitors, kills on activity. See [pkg/idle/](pkg/idle/) for CGO bindings and [internal/compositor/](internal/compositor/) for multi-monitor logic.

### 2. Display ([cmd/display/](cmd/display/))

Renders [sysc-Go](https://github.com/Nomadcxx/sysc-Go) animations in fullscreen Kitty terminals. Wraps effects with terminal sizing, theme application, and ASCII art loading. See [internal/animations/](internal/animations/).

### 3. Client ([cmd/client/](cmd/client/))

Optional CLI for testing. Not needed for normal operation.

Config lives in `~/.config/sysc-walls/daemon.conf` (see [internal/config/](internal/config/)). Build uses local sysc-Go clone via `replace` directive - will switch to GitHub module for releases.

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

## Roadmap

**Work in Progress:**

- [ ] **DateTime Effects** - Render time/date as negative space with effects filling around glyphs (fire-datetime, matrix-datetime, etc.)
- [ ] **VOID Theme** - New dark theme with deep blacks and subtle accents
- [ ] **Better X11 Support** - Improved compatibility beyond xprintidle, multi-monitor X11, hybrid Wayland/X11
- [ ] **Auto-Updating** - Self-updating daemon that checks for new versions and animations
- [ ] **More Font Options** - Additional ASCII fonts for text effects (KABEL, YES styles)
- [ ] **Effect Cycling Improvements** - Smoother transitions, configurable cycle timing
- [ ] **Custom Animation Parameters** - Per-effect configuration (speed, density, colors)
- [ ] **Lock Screen Integration** - Optional integration with swaylock/hyprlock

Have a feature request? Open an issue on [GitHub](https://github.com/Nomadcxx/sysc-walls/issues).

## Sister Projects

- **[sysc-Go](https://github.com/Nomadcxx/sysc-Go)** - Terminal animation library that powers sysc-walls effects. Includes CLI tools and interactive TUI for creating ASCII art.
- **[sysc-greet](https://github.com/Nomadcxx/sysc-greet)** - Console greeter for greetd with sysc-Go animations and video wallpapers
- **[moonbit](https://github.com/Nomadcxx/moonbit)** - Modern system cleaner with TUI for reclaiming disk space

## Acknowledgements

- [terminaltexteffects](https://github.com/ChrisBuilds/terminaltexteffects) - Inspiration for terminal visual effects
- [sysc-Go](https://github.com/Nomadcxx/sysc-Go) - Core animation library used for all effects

## License

MIT - Do whatever you want with it.
