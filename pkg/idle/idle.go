// idle.go - Idle detection implementation
package idle

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	evdev "github.com/gvalkov/golang-evdev"
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
		idleChan:    make(chan struct{}, 10),  // Larger buffer to prevent drops
		resumeChan:  make(chan struct{}, 10),  // Larger buffer to prevent drops
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
	// Initialize last active time
	d.lastActive = time.Now()

	log.Printf("Starting idle detector with timeout: %v", d.idleTimeout)

	// Detect display server and start appropriate monitor
	displayServer := detectDisplayServer()

	// Start monitoring for display server specific idle detection
	switch displayServer {
	case "wayland":
		// Use native Wayland idle detection
		return d.startWaylandIdleDetection(ctx)
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

// startWaylandIdleDetection starts native Wayland idle detection using ext-idle-notify-v1
func (d *IdleDetector) startWaylandIdleDetection(ctx context.Context) error {
	log.Println("Starting Wayland idle detection using CGO bindings to native libwayland")

	// Create Wayland idle detector
	onIdle := func() {
		log.Println("[Go callback] Wayland idle callback invoked")
		// Fire idle event
		select {
		case d.idleChan <- struct{}{}:
			if d.config.IsDebug() {
				log.Println("Idle event fired")
			}
		default:
			log.Println("[WARNING] Idle channel full, event dropped!")
		}
	}

	onResume := func() {
		log.Println("[Go callback] Wayland resume callback invoked")
		d.lastActive = time.Now()
		
		// Fire resume event
		select {
		case d.resumeChan <- struct{}{}:
			if d.config.IsDebug() {
				log.Println("Resume event fired")
			}
		default:
			log.Println("[WARNING] Resume channel full, event dropped!")
		}

		// Clear any pending idle event
		select {
		case <-d.idleChan:
		default:
		}
	}

	waylandDetector, err := NewWaylandCGODetector(d.idleTimeout, onIdle, onResume)
	if err != nil {
		log.Printf("Failed to create Wayland CGO detector: %v", err)
		log.Println("Falling back to X11 detection if available")
		d.startX11Monitor(ctx)
		return err
	}

	// Start the Wayland detector
	if err := waylandDetector.Start(); err != nil {
		log.Printf("Failed to start Wayland CGO detector: %v", err)
		return err
	}

	// Monitor context cancellation and stop the detector
	go func() {
		<-ctx.Done()
		waylandDetector.Stop()
	}()

	return nil
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
	// Discover all available input devices
	devices, err := discoverInputDevices()
	if err != nil {
		log.Printf("Failed to discover input devices: %v, falling back to polling", err)
		d.startInputDevicePolling(ctx)
		return
	}

	if len(devices) == 0 {
		log.Println("No input devices found, falling back to polling")
		d.startInputDevicePolling(ctx)
		return
	}

	if d.config.IsDebug() {
		log.Printf("Monitoring %d input devices for activity", len(devices))
	}

	// Create a channel for activity signals from all devices
	activityChan := make(chan struct{}, 10)

	// Start monitoring each device in a separate goroutine
	for _, devicePath := range devices {
		go d.monitorDevice(ctx, devicePath, activityChan)
	}

	// Listen for activity signals
	for {
		select {
		case <-ctx.Done():
			return
		case <-activityChan:
			// Activity detected on any device
			d.MarkActive()

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

// discoverInputDevices finds all available input event devices
func discoverInputDevices() ([]string, error) {
	devices := []string{}

	// List all event devices in /dev/input/
	files, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, err
	}

	// Filter to only keyboard and mouse devices
	for _, file := range files {
		device, err := evdev.Open(file)
		if err != nil {
			continue
		}

		// Check if device has key events (keyboard) or mouse events
		caps := device.Capabilities
		hasKeys := false
		hasPointer := false

		// Iterate through capabilities to check event types
		for capType := range caps {
			if capType.Type == evdev.EV_KEY {
				hasKeys = true
			}
			if capType.Type == evdev.EV_REL || capType.Type == evdev.EV_ABS {
				hasPointer = true
			}
		}

		device.File.Close()

		// Include devices that are keyboards or pointing devices
		if hasKeys || hasPointer {
			devices = append(devices, file)
		}
	}

	return devices, nil
}

// monitorDevice monitors a single input device for events
func (d *IdleDetector) monitorDevice(ctx context.Context, devicePath string, activityChan chan<- struct{}) {
	device, err := evdev.Open(devicePath)
	if err != nil {
		if d.config.IsDebug() {
			log.Printf("Failed to open device %s: %v", devicePath, err)
		}
		return
	}
	defer device.File.Close()

	if d.config.IsDebug() {
		log.Printf("Monitoring device: %s (%s)", devicePath, device.Name)
	}

	// Use non-blocking reads with select
	eventChan := make(chan *evdev.InputEvent, 10)
	errChan := make(chan error, 1)

	// Read events in a goroutine
	go func() {
		for {
			events, err := device.Read()
			if err != nil {
				errChan <- err
				return
			}
			for i := range events {
				select {
				case eventChan <- &events[i]:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Monitor for events or context cancellation
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errChan:
			if d.config.IsDebug() {
				log.Printf("Device %s read error: %v", devicePath, err)
			}
			return
		case event := <-eventChan:
			// Only care about key presses, mouse movements, button clicks
			if event.Type == evdev.EV_KEY || event.Type == evdev.EV_REL || event.Type == evdev.EV_ABS {
				select {
				case activityChan <- struct{}{}:
				default:
					// Don't block if channel is full
				}
			}
		}
	}
}

// startInputDevicePolling is a fallback method using device file polling
func (d *IdleDetector) startInputDevicePolling(ctx context.Context) {
	// Try to use X11 idle detection if available
	if hasXprintidle() {
		go d.monitorX11Idle(ctx)
		return
	}

	// Last resort: just rely on timer-based idle detection
	log.Println("Warning: No reliable activity detection method available")
}

// hasXprintidle checks if xprintidle command is available
func hasXprintidle() bool {
	_, err := exec.LookPath("xprintidle")
	return err == nil
}

// monitorX11Idle monitors X11 idle time using xprintidle
func (d *IdleDetector) monitorX11Idle(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastIdleTime int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cmd := exec.Command("xprintidle")
			output, err := cmd.Output()
			if err != nil {
				continue
			}

			var idleTimeMs int64
			_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &idleTimeMs)
			if err != nil {
				continue
			}

			// If idle time decreased, activity was detected
			if lastIdleTime > 0 && idleTimeMs < lastIdleTime {
				d.MarkActive()

				select {
				case d.resumeChan <- struct{}{}:
					if d.config.IsDebug() {
						log.Println("X11 activity detected")
					}
				default:
				}

				select {
				case <-d.idleChan:
				default:
				}
			}

			lastIdleTime = idleTimeMs
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
	// Trim whitespace and use strconv for proper parsing
	s = strings.TrimSpace(s)
	result, _ := strconv.Atoi(s)
	return result
}
