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

	"github.com/charmbracelet/lipgloss"
	"github.com/Nomadcxx/sysc-walls/internal/compositor"
	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/internal/systemd"
	"github.com/Nomadcxx/sysc-walls/internal/version"
	"github.com/Nomadcxx/sysc-walls/pkg/daemonize"
	"github.com/Nomadcxx/sysc-walls/pkg/idle"
)

// RAMA theme colors matching installer
var (
	colorPrimary   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef233c"))        // RAMA Red Pantone
	colorSecondary = lipgloss.NewStyle().Foreground(lipgloss.Color("#d90429"))        // RAMA Fire engine red
	colorAccent    = lipgloss.NewStyle().Foreground(lipgloss.Color("#edf2f4"))        // RAMA Anti-flash white
	colorMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8d99ae"))        // RAMA Cool gray
	colorError     = lipgloss.NewStyle().Foreground(lipgloss.Color("#d90429"))        // RAMA Fire engine red
	colorWarning   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef233c"))        // RAMA Red Pantone
	colorBold      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ef233c"))
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
		runAsDaemon  = flag.Bool("daemon", false, "Run as daemon (no output)")
		configPath   = flag.String("config", "", "Path to config file")
		start        = flag.Bool("start", false, "Start the daemon")
		stop         = flag.Bool("stop", false, "Stop the daemon")
		test         = flag.Bool("test", false, "Test mode - activate screensaver immediately")
		demo         = flag.Bool("demo", false, "Demo mode - cycle through all effects (30s each)")
		debug        = flag.Bool("debug", false, "Enable debug logging")
		showVersion  = flag.Bool("version", false, "Show version information")
		showVersionV = flag.Bool("v", false, "Show version information (shorthand)")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion || *showVersionV {
		fmt.Printf("%s\n", version.GetFullVersion())
		os.Exit(0)
	}

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

	// Check sysc-Go library version compatibility
	if err := config.CheckSyscGoVersion(); err != nil {
		log.Fatalf("sysc-Go version incompatibility: %v", err)
	}

	// Initialize config manager
	cfg := config.NewConfig()
	if err := cfg.LoadFromFile(expandedConfigPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure debug if requested
	if *debug {
		cfg.SetDebug(true)
		log.Printf("Version: %s", version.GetFullVersion())
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
		showTestMode(daemon, *debug, c)
		return
	}

	// Demo mode - cycle through all effects
	if *demo {
		showDemoMode(daemon, *debug, c)
		return
	}

	// No command specified, print usage
	showUsage()
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

	// Get validated screensaver command
	terminal, args, err := d.config.GetScreensaverCommand()
	if err != nil {
		log.Printf("ERROR: Invalid screensaver configuration: %v", err)
		return
	}

	if d.debug {
		log.Printf("Launching screensaver: %s %v", terminal, args)
	}

	// Detect compositor
	comp, err := compositor.DetectCompositor()
	if err != nil {
		// Fallback: launch single instance without multi-monitor support
		if d.debug {
			log.Printf("Compositor detection failed: %v, launching single instance", err)
		}
		if err := d.systemD.LaunchScreensaver(terminal, args, "default"); err != nil {
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
		if err := d.systemD.LaunchScreensaver(terminal, args, "default"); err != nil {
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
	// Use longer delays for better reliability across different compositors
	for i, output := range outputs {
		if d.debug {
			log.Printf("Launching on output %d/%d: %s", i+1, len(outputs), output.Name)
		}

		// Focus this output
		if err := comp.FocusOutput(output.Name); err != nil {
			log.Printf("Failed to focus output %s: %v", output.Name, err)
			continue
		}

		// Longer delay to ensure compositor fully processes the focus change
		// Some compositors need more time to settle before launching windows
		time.Sleep(250 * time.Millisecond)

		// Launch screensaver (window should follow focus)
		if err := d.systemD.LaunchScreensaver(terminal, args, output.Name); err != nil {
			log.Printf("Failed to launch screensaver on %s: %v", output.Name, err)
			continue
		}

		// Longer delay between launches to ensure windows initialize properly
		// This helps prevent race conditions with compositor window placement
		if i < len(outputs)-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	// Give all windows substantial time to fully initialize and become fullscreen
	// This is critical for proper multi-monitor rendering in all compositors
	time.Sleep(600 * time.Millisecond)

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

// loadASCII loads the ASCII art from ascii.txt
func loadASCII() string {
	// Try to load from ascii.txt in current directory or project root
	paths := []string{
		"ascii.txt",
		"../ascii.txt",
		"../../ascii.txt",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// Fallback ASCII if file not found
	return `▄▀▀▀▀ █   █ ▄▀▀▀▀ ▄▀▀▀▀          ▄▀ █   █
 ▀▀▀▄ ▀▀▀▀█  ▀▀▀▄ █     ▀▀▀▀▀  ▄▀   █ █ █
▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀   ▀▀▀▀       ▀      ▀ ▀`
}

// showTestMode displays the test mode interface
func showTestMode(daemon *Daemon, debugMode bool, sigChan chan os.Signal) {
	// Show ASCII art header
	ascii := loadASCII()
	fmt.Println()
	fmt.Println(colorPrimary.Render(ascii))
	fmt.Println()
	fmt.Println(colorBold.Render("        TEST MODE"))
	fmt.Println()

	daemon.debug = true

	// Show compositor info if debug enabled
	if debugMode {
		fmt.Println(colorSecondary.Render("Configuration:"))
		fmt.Println("  Effect: " + colorAccent.Render(daemon.config.GetAnimationEffect()))
		fmt.Println("  Theme:  " + colorAccent.Render(daemon.config.GetAnimationTheme()))
		fmt.Println()

		fmt.Println(colorSecondary.Render("Compositor Detection:"))
		comp, err := compositor.DetectCompositor()
		if err != nil {
			fmt.Println(colorWarning.Render("  ⚠  No compositor detected"))
			fmt.Println(colorMuted.Render("     Will attempt single-monitor launch"))
		} else {
			fmt.Println(colorAccent.Render("  ✓  Detected: ") + colorBold.Render(comp.Name()))

			// List outputs
			if outputs, err := comp.ListOutputs(); err == nil {
				fmt.Println()
				fmt.Println(colorSecondary.Render(fmt.Sprintf("Found %d output(s):", len(outputs))))
				for i, output := range outputs {
					focusMarker := ""
					if output.Focused {
						focusMarker = colorAccent.Render(" [focused]")
					}
					fmt.Printf("  %d. %s%s\n", i+1, output.Name, focusMarker)
				}
			}
		}
		fmt.Println()
	}

	fmt.Println(colorSecondary.Render("Launching screensaver..."))
	startTime := time.Now()
	daemon.LaunchScreensaver()
	elapsed := time.Since(startTime)

	processCount := daemon.systemD.GetProcessCount()
	if debugMode {
		fmt.Println(colorAccent.Render("✓ Launch complete") + colorMuted.Render(fmt.Sprintf(" (%dms)", elapsed.Milliseconds())))
		fmt.Println(colorMuted.Render(fmt.Sprintf("  Processes: %d", processCount)))
		if pids, err := daemon.systemD.GetPIDs(); err == nil {
			fmt.Println(colorMuted.Render(fmt.Sprintf("  PIDs: %v", pids)))
		}
	} else {
		fmt.Println(colorAccent.Render("✓ Screensaver launched"))
	}

	fmt.Println()
	fmt.Println(colorMuted.Render("Press Ctrl+C to stop"))
	fmt.Println()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println()
	fmt.Println(colorSecondary.Render("Stopping screensaver..."))
	daemon.StopScreensaver()
	daemon.Shutdown()
	fmt.Println(colorAccent.Render("✓ Stopped"))
}

// showDemoMode cycles through all effects for recording showcase
func showDemoMode(daemon *Daemon, debugMode bool, sigChan chan os.Signal) {
	// Show ASCII art header
	ascii := loadASCII()
	fmt.Println()
	fmt.Println(colorPrimary.Render(ascii))
	fmt.Println()
	fmt.Println(colorBold.Render("        DEMO MODE"))
	fmt.Println()

	daemon.debug = debugMode

	// Define demo effect order (text-based effects first, then non-text effects)
	// Note: decrypt, pour, and print are not yet implemented in sysc-Go
	demoEffects := []string{
		// Text-based effects
		"fire-text",
		"matrix-art",
		"beam-text",
		"rain-art",
		"ring-text",
		"blackhole",
		// Non-text effects
		"matrix",
		"rain",
		"fire",
		"fireworks",
		"beams",
		"aquarium",
	}

	effectDuration := 15 * time.Second
	theme := daemon.config.GetAnimationTheme()

	fmt.Println(colorSecondary.Render("Demo Configuration:"))
	fmt.Println(fmt.Sprintf("  Effects: %d total", len(demoEffects)))
	fmt.Println(fmt.Sprintf("  Duration: %v per effect", effectDuration))
	fmt.Println(fmt.Sprintf("  Theme: %s", theme))
	fmt.Println(fmt.Sprintf("  Total runtime: ~%v", time.Duration(len(demoEffects))*effectDuration))
	fmt.Println()
	fmt.Println(colorMuted.Render("Note: Demo runs on single monitor only, input detection disabled"))
	fmt.Println(colorMuted.Render("Press Ctrl+C to stop at any time"))
	fmt.Println()

	// Store original effect
	originalEffect := daemon.config.GetAnimationEffect()

	// Cycle through effects
	for i, effect := range demoEffects {
		// Check for interrupt
		select {
		case <-sigChan:
			fmt.Println()
			fmt.Println(colorSecondary.Render("Demo interrupted"))
			daemon.config.SetAnimationEffect(originalEffect)
			daemon.Shutdown()
			return
		default:
		}

		fmt.Println(colorPrimary.Render(fmt.Sprintf("[%d/%d] %s", i+1, len(demoEffects), effect)))

		// Set current effect
		daemon.config.SetAnimationEffect(effect)

		// Get validated screensaver command
		terminal, args, err := daemon.config.GetScreensaverCommand()
		if err != nil {
			fmt.Println(colorError.Render(fmt.Sprintf("  ✗ Invalid configuration: %v", err)))
			continue
		}

		// Replace screensaver class with demo class to avoid conflict with running service
		for i, arg := range args {
			if arg == "sysc-walls-screensaver" {
				args[i] = "sysc-walls-demo"
			}
		}

		if debugMode {
			cmdParts := append([]string{terminal}, args...)
			fmt.Println(colorMuted.Render("  Command: " + strings.Join(cmdParts, " ")))
		}

		// Launch on single monitor only
		if err := daemon.systemD.LaunchScreensaver(terminal, args, "demo"); err != nil {
			fmt.Println(colorError.Render(fmt.Sprintf("  ✗ Failed to launch: %v", err)))
			continue
		}

		if debugMode {
			if pids, err := daemon.systemD.GetPIDs(); err == nil {
				fmt.Println(colorMuted.Render(fmt.Sprintf("  PID: %v", pids)))
			}
		}

		// Give Kitty time to initialize its window (especially important for first launch)
		time.Sleep(300 * time.Millisecond)

		// Wait for duration or interrupt
		timer := time.NewTimer(effectDuration)
		select {
		case <-timer.C:
			// Duration elapsed, stop and continue to next
			daemon.StopScreensaver()
			if i < len(demoEffects)-1 {
				time.Sleep(500 * time.Millisecond) // Brief pause between effects
			}
		case <-sigChan:
			timer.Stop()
			daemon.StopScreensaver()
			fmt.Println()
			fmt.Println(colorSecondary.Render("Demo interrupted"))
			daemon.config.SetAnimationEffect(originalEffect)
			daemon.Shutdown()
			return
		}
	}

	// Restore original effect
	daemon.config.SetAnimationEffect(originalEffect)

	fmt.Println()
	fmt.Println(colorAccent.Render("✓ Demo complete"))
	fmt.Println()
	daemon.Shutdown()
}

// showUsage displays usage information
func showUsage() {
	// Show ASCII art header
	ascii := loadASCII()
	fmt.Println()
	fmt.Println(colorPrimary.Render(ascii))
	fmt.Println()
	fmt.Println(colorBold.Render("   TERMINAL SCREENSAVER DAEMON"))
	fmt.Println()

	fmt.Println(colorSecondary.Render("Usage:"))
	fmt.Println("  sysc-walls-daemon [options]")
	fmt.Println()

	fmt.Println(colorSecondary.Render("Options:"))
	fmt.Println("  " + colorAccent.Render("-start") + "              Start the daemon")
	fmt.Println("  " + colorAccent.Render("-stop") + "               Stop the daemon")
	fmt.Println("  " + colorAccent.Render("-test") + "               Test screensaver immediately")
	fmt.Println("  " + colorAccent.Render("-test -debug") + "        Test with detailed diagnostics")
	fmt.Println("  " + colorAccent.Render("-demo") + "               Cycle through all effects (30s each)")
	fmt.Println("  " + colorAccent.Render("-demo -debug") + "        Demo with command output")
	fmt.Println("  " + colorAccent.Render("-daemon") + "             Run as background daemon")
	fmt.Println("  " + colorAccent.Render("-config") + " " + colorMuted.Render("<path>") + "      Path to config file")
	fmt.Println("  " + colorAccent.Render("-debug") + "              Enable debug logging")
	fmt.Println()

	fmt.Println(colorSecondary.Render("Testing:"))
	fmt.Println("  " + colorPrimary.Render("sysc-walls-daemon -test") + colorMuted.Render("              # Quick test"))
	fmt.Println("  " + colorPrimary.Render("sysc-walls-daemon -test -debug") + colorMuted.Render("       # With diagnostics"))
	fmt.Println("  " + colorPrimary.Render("sysc-walls-daemon -demo") + colorMuted.Render("              # Showcase all effects"))
	fmt.Println()

	fmt.Println(colorSecondary.Render("Service:"))
	fmt.Println("  " + colorPrimary.Render("systemctl --user enable sysc-walls.service") + colorMuted.Render("   # Enable"))
	fmt.Println("  " + colorPrimary.Render("systemctl --user start sysc-walls.service") + colorMuted.Render("    # Start"))
	fmt.Println()

	fmt.Println(colorSecondary.Render("Available Effects:"))
	fmt.Println("  " + colorMuted.Render(strings.Join(config.AvailableEffects, ", ")))
	fmt.Println()

	fmt.Println(colorSecondary.Render("Available Themes:"))
	fmt.Println("  " + colorMuted.Render(strings.Join(config.AvailableThemes, ", ")))
	fmt.Println()

	fmt.Println(colorMuted.Render("Config: ~/.config/sysc-walls/daemon.conf"))
	fmt.Println(colorMuted.Render("Logs:   journalctl --user -u sysc-walls.service -f"))
	fmt.Println()
}
