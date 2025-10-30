// main.go - Entry point for daemon component
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/internal/systemd"
	"github.com/Nomadcxx/sysc-walls/pkg/daemonize"
	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

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

	// Initialize idle detector
	idleDetector := idle.NewIdleDetector(cfg)

	// Initialize systemd integration
	sd := systemd.NewSystemD(cfg)

	// Create a daemon instance
	d := daemonize.NewDaemon("sysc-walls-daemon")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Shutting down gracefully...")
		cancel()
	}()

	// Handle specific commands
	if *start {
		if *runAsDaemon {
			// Daemonize the process
			if err := d.Daemonize(); err != nil {
				log.Fatalf("Failed to daemonize: %v", err)
			}

			// Set up logging to a file
			setupLogging()
		}

		fmt.Println("Starting sysc-walls daemon...")

		// Start idle detector
		if err := idleDetector.Start(ctx); err != nil {
			log.Fatalf("Failed to start idle detector: %v", err)
		}

		// Handle idle events
		idleEvents := idleDetector.Events()
		screensaverCmd := cfg.GetScreensaverCommand()

		for {
			select {
			case <-ctx.Done():
				return
			case <-idleEvents.Idle:
				fmt.Println("System idle, launching screensaver")
				if err := sd.LaunchScreensaver(screensaverCmd); err != nil {
					log.Printf("Failed to launch screensaver: %v", err)
				}
			case <-idleEvents.Resume:
				fmt.Println("System activity detected, stopping screensaver")
				if err := sd.StopScreensaver(); err != nil {
					log.Printf("Failed to stop screensaver: %v", err)
				}
			}
		}
	}

	if *stop {
		fmt.Println("Stopping sysc-walls daemon...")
		if err := d.Stop(); err != nil {
			log.Printf("Failed to stop daemon: %v", err)
		}
		if err := sd.StopScreensaver(); err != nil {
			log.Printf("Failed to stop screensaver: %v", err)
		}
		return
	}

	// No command specified, handle test mode or print usage
	if *test {
		// Test mode - activate screensaver immediately
		fmt.Println("Test mode: Activating screensaver immediately...")
		screensaverCmd := cfg.GetScreensaverCommand()
		if err := sd.LaunchScreensaver(screensaverCmd); err != nil {
			log.Fatalf("Failed to launch screensaver in test mode: %v", err)
		}
		fmt.Println("Screensaver activated in test mode. Press Ctrl+C to stop.")

		// Wait for interrupt signal
		<-c
		fmt.Println("Test mode: Stopping screensaver...")
		if err := sd.StopScreensaver(); err != nil {
			log.Printf("Failed to stop screensaver: %v", err)
		}
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
