// main.go - Enhanced daemon with proper lifecycle management
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/internal/systemd"
	"github.com/Nomadcxx/sysc-walls/pkg/daemonize"
	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

// Daemon struct to manage screensaver lifecycle
type Daemon struct {
	config    *config.Config
	idleTimer *time.Timer
	ctx       context.Context
	cancel    context.CancelFunc
	systemD   *systemd.SystemD
	idleDet   *idle.IdleDetector
	debug     bool
}

// NewDaemon creates a new daemon instance
func NewDaemon(cfg *config.Config) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		config:    cfg,
		idleTimer: time.NewTimer(cfg.GetIdleTimeout()),
		ctx:       ctx,
		cancel:    cancel,
		systemD:   systemd.NewSystemD(cfg),
		idleDet:   idle.NewIdleDetector(cfg),
	}
}

func main() {
	// Parse command line flags
	var (
		runAsDaemon = flag.Bool("daemon", false, "Run as daemon (no output)")
		configPath  = flag.String("config", "", "Path to config file")
		start       = flag.Bool("start", false, "Start the daemon")
		stop        = flag.Bool("stop", false, "Stop the daemon")
		test        = flag.Bool("test", false, "Test mode - activate screensaver immediately")
		debug       = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	// Expand config path with default
	expandedConfigPath := *configPath
	if expandedConfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}
		expandedConfigPath = filepath.Join(homeDir, ".config", "sysc-walls", "daemon.conf")
	} else {
		expandedConfigPath = os.ExpandEnv(expandedConfigPath)
		if strings.HasPrefix(expandedConfigPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("Failed to get home directory: %v", err)
			}
			expandedConfigPath = filepath.Join(homeDir, expandedConfigPath[2:])
		}
	}

	// Initialize config manager
	cfg := config.NewConfig()
	if err := cfg.LoadFromFile(expandedConfigPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure debug if requested
	if *debug {
		cfg.SetDebug(true)
	}

	// Create daemon instance
	daemon := NewDaemon(cfg)
	daemon.debug = *debug

	// Setup signal handling for graceful shutdown and activity detection
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	go func() {
		for sig := range c {
			if daemon.debug {
				log.Printf("Received signal: %v", sig)
			}

			switch sig {
			case os.Interrupt, syscall.SIGTERM:
				fmt.Println("Shutting down gracefully...")
				daemon.Shutdown()
				os.Exit(0)
			case syscall.SIGUSR1, syscall.SIGUSR2:
				// Activity detected via signal
				if daemon.debug {
					log.Println("Activity detected via signal")
				}
				daemon.onActivity()
			}
		}
	}()

	// Handle specific commands
	if *start {
		if *runAsDaemon {
			// Daemonize the process
			d := daemonize.NewDaemon("sysc-walls-daemon")
			if err := d.Daemonize(); err != nil {
				log.Fatalf("Failed to daemonize: %v", err)
			}
			setupLogging()
		}

		fmt.Println("Starting sysc-walls daemon...")
		daemon.Run()
		return
	}

	if *stop {
		fmt.Println("Stopping sysc-walls daemon...")
		daemon.StopScreensaver()
		daemon.Shutdown()
		return
	}

	// Test mode - activate screensaver immediately
	if *test {
		fmt.Println("Test mode: Activating screensaver immediately...")
		daemon.debug = true
		daemon.LaunchScreensaver()
		fmt.Println("Screensaver activated in test mode. Press Ctrl+C to stop.")

		// Wait for interrupt signal
		<-c
		fmt.Println("Test mode: Stopping screensaver...")
		daemon.StopScreensaver()
		daemon.Shutdown()
		return
	}

	// No command specified, print usage
	fmt.Println("Usage: sysc-walls-daemon [options]")
	fmt.Println("Options:")
	fmt.Println("  -start              Start the daemon (requires sudo)")
	fmt.Println("  -stop               Stop the daemon (requires sudo)")
	fmt.Println("  -test               Test mode - activate screensaver immediately")
	fmt.Println("  -daemon             Run as daemon (background)")
	fmt.Println("  -config             Path to config file")
	fmt.Println("  -debug              Enable debug logging")
	flag.PrintDefaults()
}

// Run starts the main daemon loop
func (d *Daemon) Run() {
	// Start idle detector for timing-based detection
	if err := d.idleDet.Start(d.ctx); err != nil {
		log.Printf("Failed to start idle detector: %v", err)
	}

	// Start activity monitoring via xinput if available
	d.startActivityMonitoring()

	// Start main event loop
	d.eventLoop()
}

// startActivityMonitoring starts monitoring for user activity
func (d *Daemon) startActivityMonitoring() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				return
			case <-ticker.C:
				// Simple activity detection - this can be enhanced
				// For now, we'll rely on the idle detector's timing
				// In a real implementation, this would monitor input devices
			}
		}
	}()
}

// eventLoop handles all events
func (d *Daemon) eventLoop() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.idleDet.Events().Idle:
			if d.debug {
				log.Println("Idle detector triggered")
			}
			// Stop timer since we're using native detection
			d.idleTimer.Stop()
			d.onIdle()
		case <-d.idleDet.Events().Resume:
			log.Println("Daemon received resume event from channel")
			if d.debug {
				log.Println("Idle detector resume")
			}
			d.onActivity()
		case <-d.idleTimer.C:
			if d.debug {
				log.Println("Timer triggered idle (fallback)")
			}
			d.onIdle()
		}
	}
}

// onActivity handles user activity (stop screensaver, reset timer)
func (d *Daemon) onActivity() {
	log.Println("onActivity called - stopping screensaver")
	if d.debug {
		log.Println("User activity detected")
	}

	d.resetIdleTimer()
	d.StopScreensaver()
	log.Println("onActivity completed")
}

// onIdle handles idle timeout (launch screensaver)
func (d *Daemon) onIdle() {
	if d.debug {
		log.Println("System idle, launching screensaver")
	}

	d.LaunchScreensaver()
	d.resetIdleTimer()
}

// resetIdleTimer resets the idle timeout timer
func (d *Daemon) resetIdleTimer() {
	d.idleTimer.Stop()
	d.idleTimer.Reset(d.config.GetIdleTimeout())
}

// LaunchScreensaver starts the screensaver on all monitors
func (d *Daemon) LaunchScreensaver() {
	// Don't launch if already running
	if d.systemD.IsRunning() {
		if d.debug {
			log.Println("Screensaver already running, skipping launch")
		}
		return
	}

	screensaverCmd := d.config.GetScreensaverCommand()

	if d.debug {
		log.Printf("Launching screensaver: %s", screensaverCmd)
	}

	// Detect compositor
	comp, err := compositor.DetectCompositor()
	if err != nil {
		// Fallback: launch single instance without multi-monitor support
		if d.debug {
			log.Printf("Compositor detection failed: %v, launching single instance", err)
		}
		if err := d.systemD.LaunchScreensaver(screensaverCmd, "default"); err != nil {
			log.Printf("Failed to launch screensaver: %v", err)
		}
		return
	}

	if d.debug {
		log.Printf("Detected compositor: %s", comp.Name())
	}

	// Get all outputs
	outputs, err := comp.ListOutputs()
	if err != nil {
		log.Printf("Failed to list outputs: %v", err)
		// Fallback: launch single instance
		if err := d.systemD.LaunchScreensaver(screensaverCmd, "default"); err != nil {
			log.Printf("Failed to launch screensaver: %v", err)
		}
		return
	}

	if d.debug {
		log.Printf("Found %d outputs", len(outputs))
		for _, output := range outputs {
			log.Printf("  - %s", output.Name)
		}
	}

	// Save original focused output for restoration
	originalFocus, err := comp.GetFocusedOutput()
	if err != nil {
		if d.debug {
			log.Printf("Failed to get focused output: %v", err)
		}
		originalFocus = "" // Will skip restoration if empty
	} else {
		if d.debug {
			log.Printf("Original focused output: %s", originalFocus)
		}
	}

	// Launch screensaver on each output using sequential focusing
	for i, output := range outputs {
		if d.debug {
			log.Printf("Launching on output %d/%d: %s", i+1, len(outputs), output.Name)
		}

		// Focus this output
		if err := comp.FocusOutput(output.Name); err != nil {
			log.Printf("Failed to focus output %s: %v", output.Name, err)
			continue
		}

		// Small delay to ensure focus is applied
		time.Sleep(100 * time.Millisecond)

		// Launch screensaver (window should follow focus)
		if err := d.systemD.LaunchScreensaver(screensaverCmd, output.Name); err != nil {
			log.Printf("Failed to launch screensaver on %s: %v", output.Name, err)
			continue
		}

		// Delay between launches
		if i < len(outputs)-1 {
			time.Sleep(150 * time.Millisecond)
		}
	}

	// Restore original focus
	if originalFocus != "" {
		if err := comp.FocusOutput(originalFocus); err != nil {
			if d.debug {
				log.Printf("Failed to restore focus to %s: %v", originalFocus, err)
			}
		} else {
			if d.debug {
				log.Printf("Restored focus to: %s", originalFocus)
			}
		}
	}

	// Log final state
	processCount := d.systemD.GetProcessCount()
	if d.debug {
		log.Printf("Screensaver launched on %d outputs", processCount)
	}
	if pids, err := d.systemD.GetPIDs(); err == nil {
		if d.debug {
			log.Printf("Process PIDs: %v", pids)
		}
	}
}

// StopScreensaver stops the screensaver
func (d *Daemon) StopScreensaver() {
	if d.debug {
		log.Println("StopScreensaver called")
	}

	// First try systemd's tracked processes
	if err := d.systemD.StopScreensaver(); err != nil {
		log.Printf("SystemD stop failed: %v, trying pkill fallback", err)

		// Fallback: kill by specific class name to avoid killing all kitty instances
		killCmd := exec.Command("pkill", "-f", "kitty.*--class.*sysc-walls-screensaver")
		if err := killCmd.Run(); err != nil {
			log.Printf("pkill fallback also failed: %v", err)
		} else {
			if d.debug {
				log.Println("Screensaver killed via pkill fallback")
			}
		}
	} else {
		if d.debug {
			log.Println("Screensaver stopped via SystemD")
		}
	}

	if d.debug {
		log.Println("StopScreensaver finished")
	}
}

// Shutdown cleans up resources
func (d *Daemon) Shutdown() {
	d.cancel()

	// Stop screensaver
	d.StopScreensaver()

	// Stop timer
	d.idleTimer.Stop()
}

// setupLogging sets up logging to a file for daemonized processes
func setupLogging() {
	// Use user's home directory for log file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	logDir := filepath.Join(homeDir, ".local", "share", "sysc-walls")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logFile := filepath.Join(logDir, "daemon.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Redirect stdout and stderr to log file
	log.SetOutput(f)
}
