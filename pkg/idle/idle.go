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
	"sync"
	"sync/atomic"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
	evdev "github.com/gvalkov/golang-evdev"
)

// IdleDetector handles system idle detection
type IdleDetector struct {
	config      *config.Config
	mu          sync.Mutex
	lastActive  time.Time
	idleTimeout time.Duration
	idleChan    chan struct{}
	resumeChan  chan struct{}
	nativeIdle  string

	// lostChan is signalled when the native Wayland event loop ends
	// unexpectedly. The caller has to re-evaluate its idle source when that
	// happens, because nothing else in the process will notice.
	lostChan chan struct{}

	// hasActivity records whether anything at all can report a resume event:
	// the Wayland listener, the X11 poller, or an evdev device. It is
	// separate from nativeIdle, which only records sources that report idle on
	// their own. The caller needs both, because a screensaver launched with
	// no way to report activity cannot be dismissed once it is up.
	hasActivity atomic.Bool

	// lastResumeSignal coalesces resume events. See signalResume.
	lastResumeSignal time.Time
}

// resumeCoalesceWindow is the minimum gap between resume events.
//
// A resume event's only job is to tell the consumer that nothing should be
// showing. The consumer may shell out to do it, and the evdev monitor sees
// every key repeat and every mouse-move packet, so an unthrottled resume per
// event turns ordinary typing into a stream of subprocess spawns. One event
// per window conveys exactly the same information.
const resumeCoalesceWindow = 250 * time.Millisecond

// Events provides channels for idle and resume events
type Events struct {
	Idle   chan struct{}
	Resume chan struct{}

	// Lost fires when a native idle source ends unexpectedly. The caller
	// should re-evaluate which idle source is in charge; it will not be
	// told again.
	Lost chan struct{}
}

// NewIdleDetector creates a new idle detector
func NewIdleDetector(cfg *config.Config) *IdleDetector {
	return &IdleDetector{
		config:      cfg,
		idleTimeout: cfg.GetIdleTimeout(),
		idleChan:    make(chan struct{}, 10), // Larger buffer to prevent drops
		resumeChan:  make(chan struct{}, 10), // Larger buffer to prevent drops
		lastActive:  time.Now(),
		lostChan:    make(chan struct{}, 1),
	}
}

// Events returns the idle and resume event channels
func (d *IdleDetector) Events() *Events {
	return &Events{
		Idle:   d.idleChan,
		Resume: d.resumeChan,
		Lost:   d.lostChan,
	}
}

// reportLost records that the native idle source stopped working and wakes the
// caller so it can re-arm its fallback.
func (d *IdleDetector) reportLost() {
	select {
	case d.lostChan <- struct{}{}:
	default:
	}
}

// Native idle source names reported by NativeIdleSource.
const (
	SourceWaylandIdleNotify = "wayland-ext-idle-notify"
	SourceX11Xprintidle     = "x11-xprintidle"
)

// NativeIdleSource names the source that emits idle events on its own, or ""
// when none was started. Callers use it to decide whether a wall-clock fallback
// timer is needed: these sources encode the timeout and track real user
// activity, so a fallback timer alongside one would fire during active use.
//
// The evdev and polling monitors are not native sources — they only report
// activity (resume), never idle.
//
// Valid once Start has returned, including when Start returns an error: a
// failed Wayland init that fell back to a working X11 poller reports the X11
// source. This is a record of what started successfully, not a liveness check;
// it is never cleared, so a source that later stops working still reports.
func (d *IdleDetector) NativeIdleSource() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.nativeIdle
}

func (d *IdleDetector) setNativeIdleSource(name string) {
	d.mu.Lock()
	d.nativeIdle = name
	d.mu.Unlock()
}

// clearNativeIdleSource records that a source which was reporting idle has
// stopped working.
func (d *IdleDetector) clearNativeIdleSource() {
	d.mu.Lock()
	d.nativeIdle = ""
	d.mu.Unlock()
}

// signalResume delivers a resume event, throttled to at most one per
// resumeCoalesceWindow. It never blocks.
func (d *IdleDetector) signalResume() {
	now := time.Now()

	d.mu.Lock()
	if now.Sub(d.lastResumeSignal) < resumeCoalesceWindow {
		d.mu.Unlock()
		return
	}
	d.lastResumeSignal = now
	d.mu.Unlock()

	select {
	case d.resumeChan <- struct{}{}:
	default:
		// Channel already has a value, don't block
	}
}

// markHasActivity records that a resume-capable source is running.
func (d *IdleDetector) markHasActivity() {
	d.hasActivity.Store(true)
}

// HasActivitySource reports whether any running source can emit a resume
// event, meaning a screensaver would be dismissable once launched.
//
// This is deliberately not the same question as NativeIdleSource. That one
// asks whether something reports idle by itself; this one asks whether
// anything at all can tell us the user came back. With neither available, a
// wall-clock timer would still expire during active use and the screensaver
// it launches could only be cleared by stopping the daemon.
//
// Valid once Start has returned.
func (d *IdleDetector) HasActivitySource() bool {
	return d.hasActivity.Load()
}

// Start starts the idle detector
func (d *IdleDetector) Start(ctx context.Context) error {
	// Initialize last active time
	d.mu.Lock()
	d.lastActive = time.Now()
	d.mu.Unlock()

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

// waylandDetector is the part of the CGO detector idle detection depends on.
type waylandDetector interface {
	Start() error
	Stop()
}

// newWaylandDetector builds the detector for the Wayland path. Tests replace it
// to reach the failure paths, which need no compositor.
var newWaylandDetector = func(timeout time.Duration, onIdle, onResume func()) (waylandDetector, error) {
	return NewWaylandCGODetector(timeout, onIdle, onResume)
}

// startWaylandIdleDetection starts native Wayland idle detection using ext-idle-notify-v1
func (d *IdleDetector) startWaylandIdleDetection(ctx context.Context) error {
	log.Println("Starting Wayland idle detection using CGO bindings to native libwayland")

	// Create Wayland idle detector
	onIdle := func() {
		if d.config.IsDebug() {
			log.Println("[Go callback] Wayland idle callback invoked")
		}
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
		if d.config.IsDebug() {
			log.Println("[Go callback] Wayland resume callback invoked")
		}
		d.mu.Lock()
		d.lastActive = time.Now()
		d.mu.Unlock()

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

	detector, err := newWaylandDetector(d.idleTimeout, onIdle, onResume)
	if err != nil {
		log.Printf("Failed to create Wayland CGO detector: %v", err)
		log.Println("Falling back to X11 detection if available")
		d.startX11Monitor(ctx)
		return err
	}

	// Start the Wayland detector. A failure here has to fall back the same way
	// creation does: without it the daemon has no idle source and no activity
	// source, so the caller's fallback timer would launch a screensaver that
	// no keypress could dismiss.
	if err := detector.Start(); err != nil {
		log.Printf("Failed to start Wayland CGO detector: %v", err)
		log.Println("Falling back to X11 detection if available")
		detector.Stop()
		d.startX11Monitor(ctx)
		return err
	}

	// The compositor now owns idle timing via ext-idle-notify-v1
	d.setNativeIdleSource(SourceWaylandIdleNotify)

	// The compositor's resumed event is the primary activity signal.
	d.markHasActivity()

	// If that event loop ever ends without the detector being stopped, the
	// source is gone. Clear it and tell the caller, because the fallback
	// decision is made once at startup and would otherwise leave the daemon
	// with no idle source for the rest of its life.
	if setter, ok := detector.(interface{ SetOnLost(func()) }); ok {
		setter.SetOnLost(func() {
			d.clearNativeIdleSource()
			d.reportLost()
		})
	}

	// Also start direct input device monitoring as a backup
	// This catches cases where compositor's idle detection has issues (e.g., niri multi-monitor)
	log.Println("Starting input device monitoring as backup for Wayland")
	go d.startInputDeviceMonitor(ctx)

	// Monitor context cancellation and stop the detector
	go func() {
		<-ctx.Done()
		detector.Stop()
	}()

	return nil
}

// startX11Monitor starts X11 idle detection using xprintidle.
//
// Activity monitoring starts regardless; only the idle poller depends on a
// working xprintidle, so a system without one still detects input.
func (d *IdleDetector) startX11Monitor(ctx context.Context) {
	// Start input device monitoring for immediate activity detection
	go d.startInputDeviceMonitor(ctx)

	// A usable xprintidle is one that runs and returns a reading. Finding the
	// binary is not enough: it also needs a reachable X server, which a
	// Wayland session without Xwayland does not have. Claiming the idle source
	// without checking would disable the caller's fallback timer in favour of
	// a poller that can never report idle.
	xprintidlePath, err := exec.LookPath("xprintidle")
	if err != nil {
		log.Println("xprintidle not found, X11 idle detection not available")
		return
	}
	if _, err := readXprintidle(xprintidlePath); err != nil {
		log.Printf("xprintidle found at %s but not usable, X11 idle detection not available: %v", xprintidlePath, err)
		return
	}

	// xprintidle reports the X server's idle clock, and that clock only tracks
	// input delivered to the X server. In a Wayland session that is not the
	// same thing as the user's activity: with Xwayland running, keystrokes in
	// a native Wayland application never reach it, so the X clock keeps
	// climbing while the user types. Claiming the idle source here would then
	// report idle over live work, which is the worst failure this package has.
	//
	// So the source is only claimed when this really is an X session. The
	// activity monitors started above are unaffected either way, and the
	// caller's fallback timer stays armed when we decline to claim it.
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		log.Println("X11 idle detection not used in a Wayland session: the X server's idle clock does not observe native Wayland input")
		return
	}

	// xprintidle reports real idle time, so the poller emits idle events on
	// its own once the configured timeout elapses
	d.setNativeIdleSource(SourceX11Xprintidle)
	d.markHasActivity()

	// Start xprintidle monitoring in a goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		reportedFailure := false

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				idleTime, err := readXprintidle(xprintidlePath)
				if err != nil {
					// Warn once per outage rather than every second: a poller
					// that stops reporting means no idle event fires again
					if !reportedFailure {
						log.Printf("xprintidle stopped reporting, idle detection is degraded: %v", err)
						reportedFailure = true
					}
					continue
				}
				reportedFailure = false

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
					d.signalResume()

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
}

// xprintidleTimeout bounds a single xprintidle invocation. The command talks
// to an X server over a socket; if that server stops responding it can block
// indefinitely, and a blocking Output() would wedge the caller — including the
// synchronous startup probe, which would stop the daemon reaching its event
// loop at all while still looking alive to systemd.
const xprintidleTimeout = 2 * time.Second

// readXprintidle runs xprintidle once and returns the idle time it reports.
// It is bounded by xprintidleTimeout and gives up rather than hanging.
func readXprintidle(path string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), xprintidleTimeout)
	defer cancel()

	// WaitDelay matters as much as the context here. Killing the process on
	// timeout is not enough on its own: if anything it spawned still holds the
	// output pipe, Wait blocks on that pipe and the timeout never takes
	// effect. WaitDelay bounds that second wait.
	cmd := exec.CommandContext(ctx, path)
	cmd.WaitDelay = time.Second
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 0, fmt.Errorf("%s did not respond within %v", path, xprintidleTimeout)
		}
		return 0, fmt.Errorf("running %s: %w", path, err)
	}

	ms, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0, fmt.Errorf("parsing %s output %q: %w", path, output, err)
	}

	return time.Duration(ms) * time.Millisecond, nil
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

	// Listen for activity signals with rate limiting
	lastLogTime := time.Now().Add(-10 * time.Second) // Allow first log immediately
	for {
		select {
		case <-ctx.Done():
			return
		case <-activityChan:
			// Activity detected on any device. MarkActive updates the activity
			// clock and sends a throttled resume; the second, unconditional
			// send this used to have doubled the event rate for no gain.
			d.MarkActive()

			if d.config.IsDebug() && time.Since(lastLogTime) > 5*time.Second {
				log.Println("Input device activity detected")
				lastLogTime = time.Now()
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
	if err == nil {
		// This device is really being watched, so a resume can arrive.
		d.markHasActivity()
	}
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

	// Last resort: nothing can report activity. The caller checks
	// HasActivitySource and will not arm a timer that would launch a
	// screensaver nothing can dismiss.
	log.Println("No activity detection source available: the screensaver will not be launched automatically")
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
				// MarkActive already sends the throttled resume; this used to
				// send a second, unconditional one on top of it.
				d.MarkActive()

				if d.config.IsDebug() {
					log.Println("X11 activity detected")
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
	d.mu.Lock()
	d.lastActive = time.Now()
	d.mu.Unlock()

	// Fire resume event if we're currently idle
	d.signalResume()

	// Clear any pending idle event
	select {
	case <-d.idleChan:
	default:
	}

	if d.config.IsDebug() {
		log.Println("System activity detected")
	}
}
