<div align="center">
  <img src="assets/logo.png" alt="sysc-walls">
  
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

The installer builds everything and sets up the systemd service.

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

- **multi-monitor support** - automatically launches on all displays (Niri, Sway, Hyprland)
- **native Wayland idle detection** via CGO bindings to libwayland-client
- **X11 support** using xprintidle as fallback
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

## Testing & Debugging

**Quick test:**
```bash
sysc-walls-daemon -test
```

**Test with diagnostics (shows compositor, outputs, timing):**
```bash
sysc-walls-daemon -test -debug
```

**Check service logs:**
```bash
journalctl --user -u sysc-walls.service -f
```

**Test idle detection separately:**
```bash
./test-idle-detector --timeout 10
```

### Multi-Monitor Troubleshooting

If screensavers only appear on one monitor:

1. **Verify compositor detection:**
   ```bash
   sysc-walls-daemon -test -debug
   ```
   Should show your compositor (niri/hyprland/sway) and list all outputs.

2. **Check compositor is supported:**
   - Niri: Requires `niri msg` command
   - Hyprland: Requires `hyprctl` command
   - Sway: Requires `swaymsg` command

3. **Enable debug logging:**
   Edit `~/.config/sysc-walls/daemon.conf`:
   ```ini
   [daemon]
   debug = true
   ```
   Then check logs to see focus and launch sequence.

4. **If windows don't follow focus:**
   This is usually a timing issue. The daemon waits 100ms after focusing each monitor before launching. Some compositors may need more time.

## License

MIT - Do whatever you want with it.

## Credits

Built on top of [sysc-Go](https://github.com/Nomadcxx/sysc-Go) for animations. Inspired by classic terminal screensavers but actually functional on modern Wayland systems.