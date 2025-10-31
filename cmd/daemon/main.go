// main.go - Enhanced daemon with proper lifecycle management
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/internal/systemd"
	"github.com/Nomadcxx/sysc-walls/pkg/daemonize"
	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

// Daemon struct to manage screensaver lifecycle
type Daemon struct {
	config    *config.Config
	idleTimer *time.Timer
	saverPID  int
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
		saverPID:  0,
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
		configPath  = flag.String("config", "~/.config/sysc-walls/daemon.conf", "Path to config file")
		start       = flag.Bool("start", false, "Start the daemon")
		stop        = flag.Bool("stop", false, "Stop the daemon")
		test        = flag.Bool("test", false, "Test mode - activate screensaver immediately")
		debug       = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	// Initialize config manager
	cfg := config.NewConfig()
	if err := cfg.LoadFromFile(*configPath); err != nil {
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
			d.onIdle()
		case <-d.idleDet.Events().Resume:
			if d.debug {
				log.Println("Idle detector resume")
			}
			d.onActivity()
		case <-d.idleTimer.C:
			if d.debug {
				log.Println("Timer triggered idle")
			}
			d.onIdle()
		}
	}
}

// onActivity handles user activity (stop screensaver, reset timer)
func (d *Daemon) onActivity() {
	if d.debug {
		log.Println("User activity detected")
	}

	d.resetIdleTimer()
	d.StopScreensaver()
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
	if !d.idleTimer.Stop() {
		<-d.idleTimer.C
	}
	d.idleTimer.Reset(d.config.GetIdleTimeout())
}

// LaunchScreensaver starts the screensaver
func (d *Daemon) LaunchScreensaver() {
	screensaverCmd := d.config.GetScreensaverCommand()

	if d.debug {
		log.Printf("Launching screensaver: %s", screensaverCmd)
	}

	if err := d.systemD.LaunchScreensaver(screensaverCmd); err != nil {
		log.Printf("Failed to launch screensaver: %v", err)
		return
	}

	// Get PID for tracking
	if pid, err := d.systemD.GetPID(); err == nil && pid != nil {
		d.saverPID = *pid
		if d.debug {
			log.Printf("Screensaver launched with PID: %d", d.saverPID)
		}
	}
}

// StopScreensaver stops the screensaver
func (d *Daemon) StopScreensaver() {
	if d.debug {
		log.Println("Stopping screensaver")
	}

	if err := d.systemD.StopScreensaver(); err != nil {
		log.Printf("Failed to stop screensaver: %v", err)
	}

	d.saverPID = 0
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
	// Open log file
	f, err := os.OpenFile("/var/log/sysc-walls-daemon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Redirect stdout and stderr to log file
	log.SetOutput(f)
}
