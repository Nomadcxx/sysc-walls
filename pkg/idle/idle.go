// idle.go - Idle detection implementation
package idle

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

// IdleDetector handles system idle detection
type IdleDetector struct {
	config      *config.Config
	lastActive  time.Time
	idleTimeout time.Duration
	idleChan    chan struct{}
	resumeChan  chan struct{}
}

// Events provides channels for idle and resume events
type Events struct {
	Idle   chan struct{}
	Resume chan struct{}
}

// NewIdleDetector creates a new idle detector
func NewIdleDetector(cfg *config.Config) *IdleDetector {
	return &IdleDetector{
		config:      cfg,
		idleTimeout: cfg.GetIdleTimeout(),
		idleChan:    make(chan struct{}, 1),
		resumeChan:  make(chan struct{}, 1),
		lastActive:  time.Now(),
	}
}

// Events returns the idle and resume event channels
func (d *IdleDetector) Events() *Events {
	return &Events{
		Idle:   d.idleChan,
		Resume: d.resumeChan,
	}
}

// Start starts the idle detector
func (d *IdleDetector) Start(ctx context.Context) error {
	// Create ticker for regular idle checks
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Initialize last active time
	d.lastActive = time.Now()

	log.Printf("Starting idle detector with timeout: %v", d.idleTimeout)

	// Start monitoring in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()

				// Calculate idle time
				idleTime := now.Sub(d.lastActive)

				if d.config.IsDebug() {
					log.Printf("Idle time: %v", idleTime)
				}

				// Check if we've exceeded the idle threshold
				if idleTime >= d.idleTimeout {
					// Fire idle event if not already fired
					select {
					case d.idleChan <- struct{}{}:
					default:
						// Channel already has a value, don't block
					}
				} else {
					// We're active, reset the idle event channel
					select {
					case <-d.idleChan:
					default:
					}
				}
			}
		}
	}()

	// Detect display server and start appropriate monitor
	displayServer := detectDisplayServer()

	// Start monitoring for display server specific idle detection
	switch displayServer {
	case "wayland":
		// Try to detect Wayland compositor and start appropriate monitor
		d.detectAndStartWaylandMonitor(ctx)
	case "x11":
		// Start X11 idle detection using xprintidle
		d.startX11Monitor(ctx)
	default:
		log.Println("No display server detected or unsupported")
	}

	return nil
}

// detectDisplayServer determines if we're running on Wayland or X11
func detectDisplayServer() string {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "x11"
	}
	return "none"
}

// detectAndStartWaylandMonitor detects the Wayland compositor and starts the appropriate monitor
func (d *IdleDetector) detectAndStartWaylandMonitor(ctx context.Context) {
	// Get the Wayland compositor
	waylandCompositor := os.Getenv("WAYLAND_DISPLAY")

	// Hyprland
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" ||
		(waylandCompositor != "" && (waylandCompositor == "wayland-1" || waylandCompositor == "wayland-0")) {
		// Try to use hypridle
		if _, err := os.Stat("/usr/bin/hypridle"); err == nil {
			d.startHypridleMonitor(ctx)
			return
		}
	}

	// GNOME/KDE/Sway and others - use the generic idle-inhibit protocol
	if _, err := os.Stat("/usr/bin/idle-inhibit"); err == nil {
		d.startGenericWaylandMonitor(ctx)
		return
	}

	log.Println("No suitable Wayland idle detection tool found, falling back to generic monitoring")
}

// startHypridleMonitor starts hypridle with custom settings
func (d *IdleDetector) startHypridleMonitor(ctx context.Context) {
	// Build the hypridle command
	cmdStr := fmt.Sprintf("hypridle general { on-timeout = 'echo IDLE'; on-resume = 'echo RESUME'; } listener { timeout = %d; }", int(d.idleTimeout.Seconds()))

	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | hypridle", cmdStr))

	// Create pipes to capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %v", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start hypridle monitor: %v", err)
		return
	}

	// Monitor the output in goroutines
	go d.readCommandOutput(stdout, "stdout")
	go d.readCommandOutput(stderr, "stderr")

	// Create a context for the goroutines
	goroutineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up a goroutine to kill the command when context is cancelled
	go func() {
		<-goroutineCtx.Done()
		cmd.Process.Kill()
	}()
}

// readCommandOutput reads output from a pipe (stdout or stderr)
func (d *IdleDetector) readCommandOutput(reader io.ReadCloser, streamType string) {
	defer reader.Close()

	// Create a scanner to read line by line
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := trimWhitespace(scanner.Text())

		if streamType == "stdout" {
			if line == "IDLE" {
				log.Println("Wayland idle detected (hypridle)")
				d.idleChan <- struct{}{}
			} else if line == "RESUME" {
				log.Println("Wayland resume detected (hypridle)")
				d.lastActive = time.Now()
				d.resumeChan <- struct{}{}
			}
		} else {
			// stderr output
			if len(line) > 0 {
				log.Printf("hypridle stderr: %s", line)
			}
		}
	}
}

// startGenericWaylandMonitor starts a generic Wayland idle detection using idle-inhibit
func (d *IdleDetector) startGenericWaylandMonitor(ctx context.Context) {
	// This is a placeholder for a generic Wayland idle detection
	// In a real implementation, we would use a tool like idle-inhibit
	log.Println("Generic Wayland idle detection not yet implemented")
}

// startX11Monitor starts X11 idle detection using xprintidle
func (d *IdleDetector) startX11Monitor(ctx context.Context) {
	// Check if xprintidle is available
	if _, err := os.Stat("/usr/bin/xprintidle"); err != nil {
		log.Println("xprintidle not found, X11 idle detection not available")
		return
	}

	// Start xprintidle monitoring in a goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Run xprintidle
				cmd := exec.Command("xprintidle")
				output, err := cmd.Output()
				if err != nil {
					if d.config.IsDebug() {
						log.Printf("xprintidle error: %v", err)
					}
					continue
				}

				// Parse the idle time in milliseconds
				idleMs := parseInt(string(output))
				idleTime := time.Duration(idleMs) * time.Millisecond

				// Check if we've exceeded the idle threshold
				if idleTime >= d.idleTimeout {
					// Fire idle event
					select {
					case d.idleChan <- struct{}{}:
					default:
						// Channel already has a value, don't block
					}
				} else {
					// We're active, fire resume event and clear idle
					select {
					case d.resumeChan <- struct{}{}:
					default:
						// Channel already has a value, don't block
					}

					// Clear any pending idle event
					select {
					case <-d.idleChan:
					default:
					}
				}

				if d.config.IsDebug() {
					log.Printf("X11 idle time: %v", idleTime)
				}
			}
		}
	}()

	// Start input device monitoring for immediate activity detection
	go d.startInputDeviceMonitor(ctx)
}

// startInputDeviceMonitor monitors input devices for immediate activity detection
func (d *IdleDetector) startInputDeviceMonitor(ctx context.Context) {
	// Simple file stat monitoring of input devices as a proxy for activity
	// This is a basic approach that works on many systems

	// Check common input device paths
	devicePaths := []string{
		"/dev/input/event0",
		"/dev/input/event1",
		"/dev/input/event2",
		"/dev/input/mouse0",
		"/dev/input/mice",
	}

	// Get initial stat times
	lastModTimes := make(map[string]time.Time)
	for _, path := range devicePaths {
		if info, err := os.Stat(path); err == nil {
			lastModTimes[path] = info.ModTime()
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond) // Check frequently for activity
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			activityDetected := false

			// Check if any input device files have been modified
			for _, path := range devicePaths {
				if info, err := os.Stat(path); err == nil {
					if modTime, exists := lastModTimes[path]; exists {
						if info.ModTime().After(modTime) {
							// Device was accessed, activity detected
							activityDetected = true
							lastModTimes[path] = info.ModTime()
							break
						}
					} else {
						lastModTimes[path] = info.ModTime()
					}
				}
			}

			if activityDetected {
				// Fire resume event immediately
				select {
				case d.resumeChan <- struct{}{}:
					if d.config.IsDebug() {
						log.Println("Input device activity detected")
					}
				default:
					// Channel already has a value, don't block
				}

				// Clear any pending idle event
				select {
				case <-d.idleChan:
				default:
				}
			}
		}
	}
}

// MarkActive marks the system as active (e.g., on keyboard/mouse input)
func (d *IdleDetector) MarkActive() {
	d.lastActive = time.Now()

	// Fire resume event if we're currently idle
	select {
	case d.resumeChan <- struct{}{}:
	default:
		// Channel already has a value, don't block
	}

	// Clear any pending idle event
	select {
	case <-d.idleChan:
	default:
	}

	if d.config.IsDebug() {
		log.Println("System activity detected")
	}
}

// Helper functions

// trimWhitespace removes leading and trailing whitespace
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isWhitespace(s[start]) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isWhitespace checks if a byte is whitespace
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// parseInt parses an integer from a string
func parseInt(s string) int {
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			break
		}
	}
	return result
}
