// main.go - Entry point for CLI client
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/config"
)

func main() {
	// Simple commands without complex flag parsing
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "set":
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "Usage: sysc-walls set <key> <value>\n")
			os.Exit(1)
		}
		handleSetCommand(os.Args[2], os.Args[3])
	case "run":
		handleRunCommand(os.Args[2:])
	case "test":
		handleTestCommand(os.Args[2:])
	case "start":
		handleStartCommand()
	case "stop":
		handleStopCommand()
	case "status":
		handleStatusCommand()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("Usage: sysc-walls [command] [args...]\n\n")
	fmt.Println("Commands:")
	fmt.Println("  set <key> <value>  Set configuration values")
	fmt.Println("  run [effect] [theme] Run screensaver display (foreground)")
	fmt.Println("  test [effect] [theme] Test screensaver (10s preview)")
	fmt.Println("  start              Start the daemon service")
	fmt.Println("  stop               Stop the daemon service")
	fmt.Println("  status             Check daemon and service status")
	fmt.Println("  help               Show this help message")

	fmt.Println("\nSet commands:")
	fmt.Println("  sysc-walls set effect matrix")
	fmt.Println("  sysc-walls set theme dracula")
	fmt.Println("  sysc-walls set timeout 5m")
	fmt.Println("  sysc-walls set kitty")
	fmt.Println("  sysc-walls set fullscreen")

	fmt.Println("\nRun commands:")
	fmt.Println("  sysc-walls run matrix dracula")
	fmt.Println("  sysc-walls run fire nord")
	fmt.Println("  sysc-walls run  # uses current config")
}

func handleSetCommand(key, value string) {
	cfg := loadConfig()

	// The confirmation is only printed once the change is on disk, so a failed
	// write can never be reported as a success.
	var confirm string

	switch key {
	case "effect":
		if err := cfg.SetAnimationEffect(value); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting effect: %v\n", err)
			os.Exit(1)
		}
		confirm = fmt.Sprintf("Set animation effect to: %s", value)
	case "theme":
		if err := cfg.SetAnimationTheme(value); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting theme: %v\n", err)
			os.Exit(1)
		}
		confirm = fmt.Sprintf("Set animation theme to: %s", value)
	case "timeout":
		if err := cfg.SetIdleTimeout(value); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting timeout: %v\n", err)
			os.Exit(1)
		}
		confirm = fmt.Sprintf("Set idle timeout to: %s", value)
	case "kitty":
		cfg.SetTerminalKitty(true)
		confirm = "Terminal set to: kitty"
	case "xterm":
		cfg.SetTerminalKitty(false)
		confirm = "Terminal set to: xterm"
	case "fullscreen":
		cfg.SetTerminalFullscreen(true)
		confirm = "Display mode set to: fullscreen"
	case "windowed":
		cfg.SetTerminalFullscreen(false)
		confirm = "Display mode set to: windowed"
	default:
		fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
		os.Exit(1)
	}

	// A failed save is a failure. Reporting it as a warning and exiting 0
	// would tell the user their change took effect when the file on disk
	// still holds the old value.
	if err := cfg.SaveToFile(defaultConfigPath()); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(confirm)
}

func handleRunCommand(args []string) {
	cfg := loadConfig()

	// Get display binary
	displayBinary, err := findDisplayBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Build command args
	cmdArgs := buildDisplayArgs(cfg, args)

	// Run in foreground
	fmt.Printf("Running screensaver (Ctrl+C to stop)\n")
	runDisplayForeground(displayBinary, cmdArgs, 0)
}

func handleTestCommand(args []string) {
	cfg := loadConfig()

	// Get display binary
	displayBinary, err := findDisplayBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Build command args
	cmdArgs := buildDisplayArgs(cfg, args)

	// Run with 10 second timeout
	fmt.Printf("Test mode: 10 second preview (Ctrl+C to stop early)\n")
	runDisplayForeground(displayBinary, cmdArgs, 10*time.Second)
}

func handleStartCommand() {
	fmt.Println("Starting sysc-walls daemon...")
	cmd := exec.Command("systemctl", "--user", "start", "sysc-walls.service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start daemon: %v\n%s\n", err, output)
		os.Exit(1)
	}
	fmt.Println("Daemon started successfully")
}

func handleStopCommand() {
	fmt.Println("Stopping sysc-walls daemon...")
	cmd := exec.Command("systemctl", "--user", "stop", "sysc-walls.service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop daemon: %v\n%s\n", err, output)
		os.Exit(1)
	}
	fmt.Println("Daemon stopped successfully")
}

func handleStatusCommand() {
	cfg := loadConfig()

	fmt.Println("Configuration:")
	fmt.Printf("  Effect: %s\n", cfg.GetAnimationEffect())
	fmt.Printf("  Theme: %s\n", cfg.GetAnimationTheme())
	fmt.Printf("  Idle timeout: %v\n", cfg.GetIdleTimeout())
	if cfg.IsTerminalKitty() {
		fmt.Println("  Terminal: kitty")
	} else {
		fmt.Println("  Terminal: xterm")
	}
	if cfg.IsTerminalFullscreen() {
		fmt.Println("  Display: fullscreen")
	} else {
		fmt.Println("  Display: windowed")
	}

	fmt.Println("\nService status:")
	cmd := exec.Command("systemctl", "--user", "status", "sysc-walls.service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get service status: %v\n", err)
	}
	fmt.Print(string(output))
}

// loadConfig loads the configuration from the default path
func loadConfig() *config.Config {
	cfg := config.NewConfig()
	configPath := defaultConfigPath()
	if err := cfg.LoadFromFile(configPath); err != nil {
		// Use defaults if config doesn't exist
		fmt.Fprintf(os.Stderr, "Warning: Could not load config from %s: %v\n", configPath, err)
		fmt.Fprintf(os.Stderr, "Using default configuration\n")
	}
	return cfg
}

// defaultConfigPath returns the config path the client reads and writes, the
// same path the daemon loads.
func defaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(homeDir, ".config", "sysc-walls", "daemon.conf")
}

// findDisplayBinary finds the sysc-walls-display binary
func findDisplayBinary() (string, error) {
	// Check common locations
	paths := []string{
		"/usr/local/bin/sysc-walls-display",
		"sysc-walls-display", // Check PATH
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("sysc-walls-display not found in PATH or /usr/local/bin")
}

// buildDisplayArgs builds the argument list for sysc-walls-display
func buildDisplayArgs(cfg *config.Config, overrides []string) []string {
	effect := cfg.GetAnimationEffect()
	theme := cfg.GetAnimationTheme()

	// Override with command line args
	if len(overrides) >= 1 {
		effect = overrides[0]
	}
	if len(overrides) >= 2 {
		theme = overrides[1]
	}

	args := []string{
		"--effect", effect,
		"--theme", theme,
	}

	// Add file if configured
	if file := cfg.GetAnimationFile(); file != "" {
		args = append(args, "--file", file)
	}

	// Add datetime if enabled
	if cfg.GetAnimationDatetime() {
		args = append(args, "--datetime")
		args = append(args, "--datetime-position", cfg.GetDatetimePosition())
	}

	// Add fullscreen
	args = append(args, "--fullscreen")

	return args
}

// runDisplayForeground runs the display binary in foreground with optional timeout
func runDisplayForeground(binary string, args []string, timeout time.Duration) {
	ctx := context.Background()
	var cancel context.CancelFunc

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		if cancel != nil {
			cancel()
		}
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("\nPreview completed")
		} else if ctx.Err() == context.Canceled {
			fmt.Println("\nStopped by user")
		} else {
			fmt.Fprintf(os.Stderr, "Error running display: %v\n", err)
			os.Exit(1)
		}
	}
}