# sysc-walls

A terminal-based screensaver that combines the animation capabilities of [sysc-Go](https://github.com/Nomadcxx/sysc-Go) with idle detection for a visually appealing screensaver experience using kitty in fullscreen mode.

## Status

This project is currently in active development. The following components have been implemented:

- Daemon component with idle detection and screensaver management
- Enhanced idle detection implementation for Wayland and X11
- Proper daemonization of the daemon process
- Animation handling system that interfaces with sysc-Go animations
- Display component with animation rendering and terminal resize handling
- CLI client for managing the screensaver configuration
- Configuration management system with INI-style configuration files

## Features

- Uses sysc-Go animations for beautiful, resource-efficient terminal screensaver
- Detects system idle and activates screensaver after configurable timeout
- Integrates with systemd for automatic startup and management
- Customizable idle timeout, animations, and themes
- Supports terminal resize events
- CLI client for configuration management
- Proper daemonization with PID file management
- Support for Wayland and X11 environments

## Installation

### Build from source

```bash
go build ./cmd/daemon/
go build ./cmd/display/
go build ./cmd/client/
```

## Usage

### Starting the daemon

```bash
# Start the daemon
sysc-walls-daemon -start

# Start with custom config
sysc-walls-daemon -start -config ~/.config/sysc-walls/custom.conf

# Enable debug mode
sysc-walls-daemon -start -debug

# Start the daemon as a proper system daemon
sysc-walls-daemon -start -daemon
```

### Stopping the daemon

```bash
# Stop the daemon and any running screensaver
sysc-walls-daemon -stop
```

### Using the CLI client

```bash
# Configure the screensaver
sysc-walls set -effect matrix -theme dracula -timeout 300s -kitty -fullscreen

# View current configuration
sysc-walls get -effect -theme -timeout

# Check daemon status
sysc-walls status
```

### Running the display component directly

```bash
# Run a specific animation
sysc-walls display --effect fire --theme dracula --duration 10
```

## Configuration

Configuration is managed via `~/.config/sysc-walls/daemon.conf`. On first run, a default config file will be created.

### Example Configuration

```ini
# sysc-walls daemon configuration

[idle]
timeout = 300s        # Time before screensaver activates (5 minutes)
min_duration = 30s    # Minimum time screensaver should run

[daemon]
debug = false         # Enable debug logging

[animation]
effect = matrix       # Animation effect (matrix, fire, fireworks, rain, etc.)
theme = dracula       # Color theme (dracula, gruvbox, nord, tokyo-night, etc.)
cycle = true          # Cycle through multiple animations

[terminal]
kitty = true          # Use kitty terminal
fullscreen = true     # Start terminal in fullscreen mode
```

## System Integration

### Systemd Service

To automatically start the daemon, create a systemd service file:

```ini
[Unit]
Description=sysc-walls idle daemon
After=graphical.target

[Service]
Type=simple
ExecStart=/usr/bin/sysc-walls-daemon -start
Restart=always
RestartSec=1

[Install]
WantedBy=default.target
```

Install the service with:

```bash
# Copy service file
sudo cp sysc-walls.service /etc/systemd/system/

# Enable and start the service
sudo systemctl enable sysc-walls.service
sudo systemctl start sysc-walls.service
```

## Development

### Project Structure

```
sysc-walls/
├── cmd/
│   ├── daemon/         # Daemon process for idle detection
│   ├── display/        # Screensaver display application
│   └── client/         # CLI client for configuration
├── internal/
│   ├── config/         # Configuration management
│   ├── animations/     # Custom animation handling
│   └── systemd/        # System service integration
├── pkg/
│   ├── daemonize/      # Daemonization utilities
│   ├── idle/           # Idle detection logic
│   └── utils/          # Utility functions
└── assets/
    └── themes/         # Custom themes for animations
```

### Implemented Components

1. **Daemon (`cmd/daemon`)**:
   - Entry point with command-line flag parsing
   - Configuration loading from file
   - Idle detector initialization
   - Systemd integration for screensaver management
   - Signal handling for graceful shutdown
   - Proper daemonization with PID file management
   - Logging to a file for daemonized processes

2. **Configuration (`internal/config`)**:
   - Custom configuration management (without external dependencies)
   - Support for loading and saving configuration files
   - Type-safe configuration options
   - Duration parsing for time-based settings

3. **Idle Detection (`pkg/idle`)**:
   - Interface for idle detection
   - Enhanced implementation with Wayland and X11 support
   - Event channels for idle and resume events
   - Integration with external tools (hypridle for Wayland, xprintidle for X11)

4. **Daemonization (`pkg/daemonize`)**:
   - Proper process daemonization
   - PID file management
   - Graceful stopping of daemon process

5. **Animation Handling (`internal/animations`)**:
   - Factory method for creating animation objects
   - Wrapper types for different animation effects
   - Support for rendering animations via sysc-Go library
   - Resize handling for terminal changes

6. **Display (`cmd/display`)**:
   - Terminal setup and cleanup
   - Animation rendering loop
   - Window resize handling
   - Command-line flag parsing

7. **CLI Client (`cmd/client`)**:
   - Subcommand support (set, get, start, stop, status)
   - Configuration management
   - User-friendly output formatting

### Remaining Tasks

1. **Animation Integration**:
   - Optimize animation rendering for better performance
   - Add animation cycling functionality
   - Implement direct integration with sysc-Go library (not just the command-line tool)

2. **Multi-Display Support**:
   - Implement detection of active display
   - Handle display changes dynamically
   - Support for multiple monitors

3. **Animation Improvements**:
   - Create custom themes for animations
   - Add more animation effects
   - Implement animation randomization

4. **System Integration**:
   - Add support for suspend/resume events
   - Integrate with existing screensaver lock systems

5. **Testing and Optimization**:
   - Add unit tests for key components
   - Create integration tests for the full screensaver workflow
   - Optimize performance and resource usage

## License

MIT