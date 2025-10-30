# sysc-walls

> ⚠️ **IN DEVELOPMENT** - This project is actively being developed

A terminal-based screensaver that combines beautiful animations from [sysc-Go](https://github.com/Nomadcxx/sysc-Go) with intelligent idle detection. Features fullscreen kitty terminal integration and systemd service support.

## Quick Start

### Installation (Recommended)
```bash
# Download and run the installer
sudo ./installer
```

### Manual Build
```bash
go build ./cmd/daemon/
go build ./cmd/display/ 
go build ./cmd/client/
```

## Usage

### Start the Screensaver
```bash
# Test screensaver immediately
sysc-walls-client test matrix nord

# Or run specific animation  
sysc-walls-client run fire dracula

# Start the idle daemon (5min timeout)
sudo systemctl start sysc-walls.service
```

### Configure
```bash
# Set animation and theme
sysc-walls-client set effect matrix
sysc-walls-client set theme dracula

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

✨ **Beautiful animations** from sysc-Go library  
⏰ **Smart idle detection** for Wayland and X11  
🚀 **Systemd integration** for auto-start  
🎨 **Customizable themes** and animations  
📱 **CLI management** for easy configuration  
⚡ **Resource efficient** terminal screensaver  

## Animation Effects

Available animations: `matrix`, `fire`, `fireworks`, `rain`, `beams`, `beam-text`, `decrypt`, `pour`, `aquarium`, `print`

Themes: `dracula`, `gruvbox`, `nord`, `ocean`, `forest`

## Configuration

Config file: `~/.config/sysc-walls/daemon.conf`

```ini
[idle]
timeout = 5m          # Idle timeout
min_duration = 30s    # Minimum screensaver duration

[animation] 
effect = matrix       # Animation effect
theme = dracula       # Color theme
cycle = true          # Cycle through animations

[terminal]
kitty = true          # Use kitty terminal
fullscreen = true     # Fullscreen mode
```

## Development

Built with Go using sysc-Go animations library. Features:

- **Daemon**: Idle detection and screensaver management
- **Display**: Animation rendering with terminal integration  
- **Client**: Intuitive CLI for configuration
- **Systemd**: Automatic service management

## License

MIT