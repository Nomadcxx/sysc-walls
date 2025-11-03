<div align="center">
  <img src="assets/logo.png" alt="sysc-walls" width="600">
  
  **Terminal screensaver with animations and idle detection**
  
  [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
  [![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](go.mod)
</div>

---

A screensaver for your terminal. Watches for idle activity on Wayland (or X11), then displays animations from [sysc-Go](https://github.com/Nomadcxx/sysc-Go) when you step away. Runs as a systemd service so it starts with your session.

## Getting Started

**Clone and install:**
```bash
git clone https://github.com/Nomadcxx/sysc-walls.git
cd sysc-walls
sudo ./installer
```

The installer builds everything and sets up the systemd service. After that:

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
effect = matrix       # Which animation to show
theme = nord          # Color scheme
cycle = false         # Rotate through effects

[daemon]
debug = false         # Enable logging
```

To test immediately without waiting:
```bash
/usr/local/bin/sysc-walls-daemon -test
```

## What's Inside

- **native Wayland idle detection** via CGO bindings to libwayland-client (works with Niri, Sway, Hyprland, etc.)
- **X11 support** using xprintidle
- **systemd integration** so it starts with your session
- **multiple animations**: matrix, fire, fireworks, rain, beams, decrypt, pour, aquarium, and more
- **color themes**: dracula, gruvbox, nord, tokyo-night, catppuccin, material, solarized
- **fullscreen terminal rendering** that actually uses your entire screen
- **resource efficient** - sleeps when you're active

## How It Works

Three simple components:

1. **daemon** - Watches for idle activity using Wayland protocols (or X11 as fallback), launches screensaver when you go idle
2. **display** - Renders sysc-Go animations in fullscreen Kitty terminal
3. **client** - CLI tool for managing settings (not required, just convenient)

The daemon uses CGO bindings to native Wayland libraries, following the same approach as swayidle. This means it works reliably across different compositors without depending on archived Go libraries.

## Debugging

If something's wonky, check the logs:
```bash
journalctl --user -u sysc-walls.service -f
```

Or test idle detection directly:
```bash
./test-idle-detector --timeout 10
```

This will show you when idle/resume events are detected without launching the full screensaver.

## License

MIT - Do whatever you want with it.

## Credits

Built on top of [sysc-Go](https://github.com/Nomadcxx/sysc-Go) for animations. Inspired by classic terminal screensavers but actually functional on modern Wayland systems.