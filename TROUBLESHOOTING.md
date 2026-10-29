# Troubleshooting & Configuration

This guide covers common issues, configuration options, and debugging steps for sysc-walls.

## Configuration

### Config File Location

`~/.config/sysc-walls/daemon.conf`

The installer automatically creates/updates this file with current defaults. A backup is saved to `daemon.conf.backup` when updating.

### ASCII Art Directory

`~/.config/sysc-walls/ascii/`

The installer copies all ASCII art files from sysc-Go to this directory. Text-based effects like `matrix-art` and `rain-art` automatically load `SYSC.txt` from here.

**Adding custom ASCII art:**

1. Create or download your ASCII art file (plain text)
2. Save it to `~/.config/sysc-walls/ascii/myart.txt`
3. Use it with the `--file` flag:
   ```bash
   /usr/local/bin/sysc-walls-display --effect matrix-art --theme rama --file ~/.config/sysc-walls/ascii/myart.txt
   ```

**Available ASCII art files:**
- `SYSC.txt` - Default SYSC logo (used automatically)
- `SYSC2.txt` - Alternative SYSC design
- `SYSC3.txt` - Another SYSC variant

## Creating Custom ASCII Art

Text effects such as `matrix-art`, `rain-art`, `fire-text`, `beam-text`, `ring-text`, `blackhole`, `decrypt`, `pour`, and `print` display plain-text artwork. Use sysclet to create a font-based header, or make artwork in any text editor.

### Export a header with sysclet

Install [sysclet v0.3.0 or later](https://github.com/Nomadcxx/sysclet/releases), then export a header:

```sh
sysclet export --format sysc-walls \
  --font-name Bleach --output "$HOME/.config/sysc-walls/ascii/myart" SYSC
```

sysclet writes plain text as `myart.txt`, points the daemon config at it, and sets the effect to `beam-text` with the `dracula` theme. A bare `--output` name such as `myart` writes `~/.local/share/syscgo/walls/myart.txt` instead; any other path must resolve under `~/.local/share` or `~/.config`.

In `~/.config/sysc-walls/daemon.conf`, sysclet sets `effect`, `theme`, `file` and `cycle = false` under `[animation]` and keeps every other line. It saves the original beside it as `daemon.conf.sysclet-<timestamp>.bak`, edits the target when `daemon.conf` is a symlink, and creates the file when it is missing. `follow_shell` stays as it was: with `follow_shell = true`, the default, sysc-walls keeps using the sysc-shell theme while sysc-shell has one selected, and uses `dracula` otherwise.

The daemon reads `daemon.conf` only at start. Restart the service to apply it:

```sh
systemctl --user restart sysc-walls.service
```

### Manual ASCII Art Creation

You can also create ASCII art manually using any text editor:

**Tips for good ASCII art:**

1. **Use block characters** for solid areas:
   ```
   █ ▓ ▒ ░ ▄ ▀ ▌ ▐
   ```

2. **Keep it proportional** - Terminal characters are taller than they are wide (roughly 2:1)

3. **Test readability** - View it in your terminal at full screen to ensure it looks good

4. **Size considerations:**
   - Recommended width: 80-120 characters
   - Recommended height: 15-30 lines
   - Too large and it won't fit on smaller screens
   - Too small and the effect won't be as dramatic

5. **Character density:**
   - Matrix-art works best with solid blocks of text
   - Leave some spacing for the "crystallization" effect to show

**Example ASCII art structure:**
```
▄▀▀▀▀ █   █ ▄▀▀▀▀ ▄▀▀▀▀          ▄▀ █   █
 ▀▀▀▄ ▀▀▀▀█  ▀▀▀▄ █     ▀▀▀▀▀  ▄▀   █ █ █
▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀   ▀▀▀▀       ▀      ▀ ▀
```
### Using Custom ASCII Art with Effects

Once you have your ASCII art file in `~/.config/sysc-walls/ascii/`:

**For manual testing:**
```bash
# Test with matrix-art
/usr/local/bin/sysc-walls-display --effect matrix-art --theme rama --file ~/.config/sysc-walls/ascii/myart.txt

# Test with fire-text
/usr/local/bin/sysc-walls-display --effect fire-text --theme rama --file ~/.config/sysc-walls/ascii/myart.txt

# Test with beam-text
/usr/local/bin/sysc-walls-display --effect beam-text --theme dracula --file ~/.config/sysc-walls/ascii/myart.txt

# Test with ring-text
/usr/local/bin/sysc-walls-display --effect ring-text --theme nord --file ~/.config/sysc-walls/ascii/myart.txt
```

**For use with the daemon:**

The daemon uses `SYSC.txt` by default. For a manually created file, set `file = /path/to/myart.txt` under `[animation]` in `daemon.conf`, or replace `SYSC.txt`. Restart the service after changing the file or config:

```sh
systemctl --user restart sysc-walls.service
```

### ASCII Art Resources

**Dedicated ASCII editors:**
- [Moebius](https://blocktronics.github.io/moebius/) - Modern ANSI/ASCII art editor with live preview
- [REXPaint](https://www.gridsagegames.com/rexpaint/) - Powerful ASCII art editor for detailed work

**Online generators:**
- [patorjk.com](https://patorjk.com/software/taag/) - Text to ASCII Art Generator
- [ascii-art-generator.org](https://ascii-art-generator.org/) - Image to ASCII converter

**Creating from images:**

1. Use an online converter or tool like `jp2a`:
   ```bash
   # Install jp2a
   sudo pacman -S jp2a  # Arch
   sudo apt install jp2a  # Debian/Ubuntu

   # Convert image to ASCII
   jp2a --width=80 --height=20 image.png > myart.txt
   ```

2. Edit the output to refine it
3. Save to `~/.config/sysc-walls/ascii/`

### Config Options

```ini
[idle]
timeout = 5m          # Time before screensaver activates
min_duration = 30s    # Minimum screensaver run time

[daemon]
debug = false         # Enable detailed logging

[animation]
effect = matrix-art   # Animation to display
theme = rama          # Color palette
cycle = false         # Rotate through effects

[terminal]
kitty = true          # Use kitty terminal (required)
fullscreen = true     # Launch in fullscreen mode
```

### Available Effects

Text effects, which use custom ASCII art:

`matrix-art`, `fire-text`, `rain-art`, `beam-text`, `decrypt`, `pour`, `print`, `blackhole`, `ring-text`, `ak-47-text`

Standalone effects, which render without text:

`matrix`, `fire`, `fireworks`, `rain`, `beams`, `aquarium`, `skull`, `sonar`, `cracktro`, `sysc-logo`, `cross-logo`, `logo-morph`, `ak-47`, `great-wave`, `wasteland`

### Available Themes

`dracula`, `catppuccin`, `catppuccin-mocha`, `nord`, `tokyo-night` (`tokyonight`), `gruvbox`, `material`, `solarized`, `monochrome`, `transishardjob`, `rama`, `eldritch`, `dark`, `ayu`, `amber`, `blue`, `purple`, `green`, `orange`, `rose-pine`, `kanagawa`, `noctalia`, `eldritch-abyss`, `void`, `red`, `cyan`, `coral`, `pink`.

sysc-walls reads sysc-shell's last committed named theme from `$XDG_CONFIG_HOME/sysc-shell/shell-theme` (or `~/.config/sysc-shell/shell-theme` when `XDG_CONFIG_HOME` is unset) each time the screensaver starts. Set `follow_shell = false` in `~/.config/sysc-walls/daemon.conf` or run `sysc-walls set follow-shell false` to use the configured `theme` instead. A manual `sysc-walls set theme <name>` disables following; `sysc-walls set follow-shell true` re-enables it.

## Key Files

### Binaries
- `/usr/local/bin/sysc-walls-daemon` - Main idle detection daemon
- `/usr/local/bin/sysc-walls-display` - Animation renderer
- `/usr/local/bin/sysc-walls-client` - CLI management tool

### Configuration & Data
- `~/.config/sysc-walls/daemon.conf` - Main configuration file
- `~/.config/sysc-walls/daemon.conf.backup` - Backup created by installer
- `~/.config/sysc-walls/ascii/` - ASCII art files for text-based effects

### Service
- `~/.config/systemd/user/sysc-walls.service` - systemd user service
- Source: `systemd/sysc-walls-user.service` in repo

### Source Code Structure
- `cmd/daemon/` - Daemon implementation
- `cmd/display/` - Display renderer
- `cmd/installer/` - Interactive installer
- `pkg/idle/` - Wayland/X11 idle detection
- `internal/compositor/` - Multi-monitor compositor support
- `internal/config/` - Configuration management
- `internal/animations/` - Animation wrappers for sysc-Go
- `sysc-Go/` - External animation library (local dependency)

## Debugging

### Check Service Status

```bash
systemctl --user status sysc-walls.service
```

Should show `active (running)`. If not:

```bash
systemctl --user restart sysc-walls.service
```

### View Live Logs

```bash
journalctl --user -u sysc-walls.service -f
```

Look for:
- "Detected compositor: niri/hyprland/sway/labwc" - confirms Wayland detection
- "System idle, launching screensaver" - confirms idle detection working
- "Launched screensaver on [output]" - confirms launch success

### Enable Debug Mode

Edit `~/.config/sysc-walls/daemon.conf`:

```ini
[daemon]
debug = true
```

Then restart:

```bash
systemctl --user restart sysc-walls.service
```

Debug mode shows:
- Compositor detection details
- All monitor outputs and focus changes
- Exact commands being executed
- Timing information

### Test Without Systemd

Run the daemon directly to see all output:

```bash
# Quick test (launches immediately)
sysc-walls-daemon -test

# Test with debug output
sysc-walls-daemon -test -debug

# Run daemon manually
sysc-walls-daemon -start
```

## Common Issues

### "Compositor detection failed: not running on Wayland"

**Cause:** The systemd service doesn't have access to `WAYLAND_DISPLAY` environment variable.

**Fix:** Reinstall with latest installer, which automatically imports the environment:

```bash
cd sysc-walls
go build -o install-sysc-walls ./cmd/installer/
sudo ./install-sysc-walls
```

**Manual fix:**

```bash
systemctl --user import-environment WAYLAND_DISPLAY
systemctl --user restart sysc-walls.service
```

### Screensaver only appears on one monitor

**Symptoms:** Multi-monitor setup but screensaver only launches on a single display.

**Check compositor support:**

```bash
sysc-walls-daemon -test -debug
```

Should list all your monitors and show compositor name. If it says "launching single instance", compositor detection failed.

**Verify compositor commands:**

```bash
# For Niri:
niri msg outputs

# For Hyprland:
hyprctl monitors

# For Sway:
swaymsg -t get_outputs
```

If these commands fail, your compositor isn't properly installed or configured.

### Screensaver doesn't respond to keyboard/mouse

**Cause:** Input detection not working properly.

**Check idle detection:**

```bash
sysc-walls-daemon -test -debug
```

Should show input events being detected. If not, the daemon may not have proper access to input devices or Wayland protocols.

**Verify user permissions:**

```bash
# Check if you're in the input group
groups | grep input

# Add yourself if missing (then logout/login)
sudo usermod -a -G input $USER
```

### Config changes not taking effect

**Cause:** Service is still running with old config.

**Fix:**

```bash
systemctl --user restart sysc-walls.service
```

Or test directly:

```bash
sysc-walls-daemon -test -debug
```

### Animation looks wrong or doesn't display

**Check theme and effect are valid:**

```bash
# List available options in the error message
sysc-walls-daemon -test
```

**Common typos:**
- `matrix-art` not `matrix_art`
- `rama` not `RAMA`

**If animation file is missing:**

```bash
# Update sysc-Go dependency
cd sysc-walls/sysc-Go
git pull
cd ..
go build -o daemon ./cmd/daemon/
sudo cp daemon /usr/local/bin/sysc-walls-daemon
```

## Compositor-Specific Configuration

### Niri

Niri support requires the `niri msg` command. The daemon uses text parsing for output detection.

**Verify niri is working:**

```bash
niri msg outputs
niri msg version
```

**Import environment on startup:**

Add to your niri config to ensure systemd services have access:

```kdl
// In your niri config
spawn-at-startup "systemctl" "--user" "import-environment" "WAYLAND_DISPLAY"
```

### Hyprland

Hyprland support requires `hyprctl`. The daemon uses JSON output for monitor detection.

**Verify hyprland is working:**

```bash
hyprctl monitors -j
hyprctl version
```

**Import environment:**

Add to `~/.config/hypr/hyprland.conf`:

```conf
exec-once = systemctl --user import-environment WAYLAND_DISPLAY
```

### labwc

labwc support requires `wlr-randr` (it reads outputs through wlr-output-management).
labwc has no IPC, so the daemon cannot move focus between outputs and launches a
single fullscreen instance on whichever output already has focus.

Make sure the daemon sees the session environment:

```bash
systemctl --user import-environment WAYLAND_DISPLAY XDG_CURRENT_DESKTOP
```

### Sway

Sway support requires `swaymsg`. The daemon uses JSON output.

**Verify sway is working:**

```bash
swaymsg -t get_outputs
swaymsg -t get_version
```

**Import environment:**

Add to `~/.config/sway/config`:

```
exec systemctl --user import-environment WAYLAND_DISPLAY
```

### GNOME (Wayland)

GNOME Wayland sessions should work with the fallback single-monitor mode if compositor-specific detection fails.

**Ensure WAYLAND_DISPLAY is set:**

```bash
echo $WAYLAND_DISPLAY
```

Should output something like `wayland-0` or `wayland-1`.

### KDE Plasma (Wayland)

Similar to GNOME, KDE Plasma should work in fallback mode.

**Check session type:**

```bash
echo $XDG_SESSION_TYPE
```

Should output `wayland`.

### X11 Fallback (Work-in-progress)

If running on X11, the daemon automatically falls back to xprintidle for idle detection. Multi-monitor support is not available on X11. X11 is still very much a work in progress.

**Install xprintidle:**

```bash
# Arch/Manjaro
sudo pacman -S xprintidle

# Debian/Ubuntu
sudo apt install xprintidle

# Fedora
sudo dnf install xprintidle
```

