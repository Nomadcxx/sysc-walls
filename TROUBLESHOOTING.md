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

Text-based effects like `matrix-art`, `rain-art`, `fire-text`, `beam-text`, `ring-text`, and `blackhole` can display custom ASCII art. You can create your own art using the sysc-Go TUI tool or manually.

### Using the sysc-Go ASCII Creator TUI

The sysc-Go library includes an interactive TUI for creating and previewing ASCII art:

**Install sysc-Go (one-line):**
```bash
curl -fsSL https://raw.githubusercontent.com/Nomadcxx/sysc-Go/master/install.sh | bash
```

**Or build manually:**
```bash
git clone https://github.com/Nomadcxx/sysc-Go.git
cd sysc-Go
go build -o syscgo ./cmd/syscgo/
./syscgo
```

**TUI Features:**
- **Live preview** - See your ASCII art with matrix-art, rain-art, or other text effects in real-time
- **Theme selection** - Test different color palettes
- **Animation preview** - Watch how the effect animates with your text
- **Built-in editor** - Create ASCII art directly in the terminal
- **File browser** - Load existing .txt files from assets directory
- **Export options** - Save to sysc-Go assets or sysc-walls config directory

**Workflow:**

1. **Start the TUI:**
   ```bash
   ./syscgo-tui
   ```

2. **Choose creation method:**
   - Select "Custom text" to open the built-in editor
   - Or select an existing .txt file as a starting point

3. **Create your ASCII art:**
   - Type or paste ASCII art in the editor
   - Use the preview pane to see it animated in real-time
   - Test different effects (matrix-art, rain-art) and themes

4. **Save your creation:**
   - Press the export key (shown in help bar)
   - Choose export target:
     - **sysc-walls config** - Saves to `~/.config/sysc-walls/ascii/` (for immediate use with sysc-walls daemon)
     - **sysc-Go assets** - Saves to `~/.local/share/syscgo/assets/` (for use with standalone sysc-Go)
   - Enter a filename (e.g., `myart.txt`)

5. **Use your ASCII art:**
   - If exported to sysc-walls config, it's ready to use
   - Reinstall sysc-walls to pick up the new file:
     ```bash
     cd sysc-walls
     go run cmd/installer/main.go
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
████████████████  ████  ████  ████████████████
████        ████  ████  ████  ████        ████
████████████████  ████  ████  ████████████████
▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒  ▒▒▒▒  ▒▒▒▒  ▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒
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

The daemon currently uses `SYSC.txt` by default for all text-based effects. To use a different file:

1. Rename your file to `SYSC.txt`:
   ```bash
   mv ~/.config/sysc-walls/ascii/myart.txt ~/.config/sysc-walls/ascii/SYSC.txt
   ```

2. Or keep the original and create a symlink:
   ```bash
   ln -sf ~/.config/sysc-walls/ascii/myart.txt ~/.config/sysc-walls/ascii/SYSC.txt
   ```

3. Restart the daemon:
   ```bash
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

- `matrix` - Classic Matrix digital rain
- `matrix-art` - Matrix rain that crystallizes into ASCII art
- `fire` - Animated flames
- `fire-text` - Fire effect with ASCII art overlay
- `fireworks` - Firework explosions
- `rain` - Falling rain droplets
- `rain-art` - Rain crystallizing into ASCII art
- `beams` - Horizontal light beams
- `beam-text` - Text revealed by scanning beams
- `decrypt` - Text decryption animation
- `pour` - Liquid pouring effect
- `aquarium` - Swimming ASCII fish
- `print` - Typewriter text printing
- `blackhole` - Text pulled into center vortex
- `ring-text` - Text in circular rings

### Available Themes

- `rama` - Space cadet color scheme (default)
- `nord` - Cool nordic palette
- `dracula` - Purple-dark theme
- `gruvbox` - Retro warm colors
- `tokyo-night` - Modern dark blue
- `catppuccin` - Pastel mocha
- `material` - Material Design colors
- `solarized` - Precision colors for machines and people
- `monochrome` - Black and white
- `eldritch` - Cosmic purple/cyan
- `dark` - Grayscale gradient
- `transishardjob` - Trans pride colors

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
- "Detected compositor: niri/hyprland/sway" - confirms Wayland detection
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

If running on X11, the daemon automatically falls back to xprintidle for idle detection. Multi-monitor support is not available on X11.

**Install xprintidle:**

```bash
# Arch/Manjaro
sudo pacman -S xprintidle

# Debian/Ubuntu
sudo apt install xprintidle

# Fedora
sudo dnf install xprintidle
```

