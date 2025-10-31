# sysc-walls

> IN DEVELOPMENT - This project is actively being developed

A terminal-based screensaver that combines animations from sysc-Go with intelligent idle detection. Features direct binary execution and systemd service support.

## Quick Start

### Installation (Recommended)
```bash
# Download and run the installer
sudo ./installer
```

### Manual Build
```bash
go build -o sysc-walls-daemon ./cmd/daemon/
go build -o sysc-walls-display ./cmd/display/
go build -o sysc-walls-client ./cmd/client/
```

## Usage

### Start the Screensaver
```bash
# Test screensaver immediately
./sysc-walls-daemon -test

# Start the idle daemon (5min timeout)
sudo systemctl start sysc-walls.service
```

### Configure
```bash
# Set animation and theme
sysc-walls-client set effect matrix
sysc-walls-client set theme nord

# Set idle timeout (5 minutes)
sysc-walls-client set timeout 5m

# Check current settings
sysc-walls-client status
```

### Daemon Management
```bash
# Enable auto-start on boot
sudo systemctl enable sysc-walls.service

# Start/stop daemon
sudo systemctl start sysc-walls.service
sudo systemctl stop sysc-walls.service

# Check status
systemctl status sysc-walls.service
```

## Features

- Terminal animations using sysc-Go library
- Smart idle detection for Wayland and X11
- Systemd service integration for auto-start
- Direct binary execution (no terminal dependency)
- CLI management for configuration
- Resource efficient screensaver

## Animation Effects

Available animations: matrix, fire, fireworks, rain, beams, beam-text, decrypt, pour, aquarium, print

Available themes: dracula, gruvbox, nord, tokyo-night, catppuccin, material, solarized, monochrome, transishardjob

## Configuration

Config file: ~/.config/sysc-walls/daemon.conf

```ini
[idle]
timeout = 5m          # Idle timeout
min_duration = 30s    # Minimum screensaver duration

[animation]
effect = matrix       # Animation effect
theme = nord          # Color theme
cycle = true          # Cycle through animations

[daemon]
debug = false         # Enable debug logging
```

## Development

Built with Go using sysc-Go animations library:

- Daemon: Idle detection and screensaver management
- Display: Animation rendering with direct execution
- Client: CLI configuration management
- Systemd: Service integration

## License

MIT