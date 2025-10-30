// main.go - Entry point for CLI client
package main

import (
	"fmt"
	"os"

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
	fmt.Println("  run [effect] [theme] Run screensaver display")
	fmt.Println("  start              Start the daemon")
	fmt.Println("  stop               Stop the daemon")
	fmt.Println("  test [effect] [theme] Test screensaver immediately")
	fmt.Println("  status             Check daemon status")
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
	cfg := config.NewConfig()

	switch key {
	case "effect":
		cfg.SetAnimationEffect(value)
		fmt.Printf("Set animation effect to: %s\n", value)
	case "theme":
		cfg.SetAnimationTheme(value)
		fmt.Printf("Set animation theme to: %s\n", value)
	case "timeout":
		if err := cfg.SetIdleTimeout(value); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting timeout: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set idle timeout to: %s\n", value)
	case "kitty":
		cfg.SetTerminalKitty(true)
		fmt.Println("Terminal set to: kitty")
	case "xterm":
		cfg.SetTerminalKitty(false)
		fmt.Println("Terminal set to: xterm")
	case "fullscreen":
		cfg.SetTerminalFullscreen(true)
		fmt.Println("Display mode set to: fullscreen")
	case "windowed":
		cfg.SetTerminalFullscreen(false)
		fmt.Println("Display mode set to: windowed")
	default:
		fmt.Fprintf(os.Stderr, "Unknown config key: %s\n", key)
		os.Exit(1)
	}

	cfg.SaveToFile("")
}

func handleRunCommand(args []string) {
	cfg := config.NewConfig()

	var effect, theme string
	if len(args) >= 1 {
		effect = args[0]
	} else {
		effect = cfg.GetAnimationEffect()
	}

	if len(args) >= 2 {
		theme = args[1]
	} else {
		theme = cfg.GetAnimationTheme()
	}

	fmt.Printf("Running screensaver with effect: %s and theme: %s\n", effect, theme)
	fmt.Println("Press Ctrl+C to stop.")

	// This would launch the display component in real implementation
	fmt.Printf("Command would be: /usr/local/bin/sysc-walls-display -effect %s -theme %s\n", effect, theme)
}

func handleTestCommand(args []string) {
	cfg := config.NewConfig()

	var effect, theme string
	if len(args) >= 1 {
		effect = args[0]
	} else {
		effect = cfg.GetAnimationEffect()
	}

	if len(args) >= 2 {
		theme = args[1]
	} else {
		theme = cfg.GetAnimationTheme()
	}

	fmt.Printf("Test mode: Starting screensaver with effect: %s and theme: %s\n", effect, theme)
	fmt.Println("Press Ctrl+C to stop.")

	// This would launch the display component in real implementation
	fmt.Printf("Command would be: /usr/local/bin/sysc-walls-display -effect %s -theme %s -fullscreen\n", effect, theme)
}

func handleStartCommand() {
	fmt.Println("Starting sysc-walls daemon...")
	fmt.Println("Use: systemctl start sysc-walls.service")
}

func handleStopCommand() {
	fmt.Println("Stopping sysc-walls daemon...")
	fmt.Println("Use: systemctl stop sysc-walls.service")
}

func handleStatusCommand() {
	cfg := config.NewConfig()

	fmt.Println("sysc-walls status:")
	fmt.Printf("  Animation effect: %s\n", cfg.GetAnimationEffect())
	fmt.Printf("  Animation theme: %s\n", cfg.GetAnimationTheme())
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

	fmt.Println("\nSystemd service status:")
	fmt.Println("Use: systemctl status sysc-walls.service")
}
