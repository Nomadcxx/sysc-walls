// main.go - Test CLI for sysc-walls screensaver
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/internal/systemd"
	"github.com/spf13/cobra"
)

var (
	effect          string
	theme           string
	debug           bool
	singleMonitor   bool
	listCompositors bool
	listOutputs     bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "test-screensaver",
		Short: "Test and launch sysc-walls screensaver",
		Long:  getHeader() + "\n\nA testing tool for sysc-walls screensaver.\nLaunch screensavers immediately without waiting for idle timeout.",
		Run:   runScreensaver,
	}

	rootCmd.Flags().StringVarP(&effect, "effect", "e", "matrix", "Animation effect (matrix, fire, rain, etc.)")
	rootCmd.Flags().StringVarP(&theme, "theme", "t", "nord", "Color theme (nord, dracula, gruvbox, etc.)")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.Flags().BoolVarP(&singleMonitor, "single", "s", false, "Launch on single monitor only")
	rootCmd.Flags().BoolVarP(&listCompositors, "list-compositors", "c", false, "List detected compositor and exit")
	rootCmd.Flags().BoolVarP(&listOutputs, "list-outputs", "o", false, "List all outputs and exit")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getHeader() string {
	return `
▄▀▀▀▀ █   █ ▄▀▀▀▀ ▄▀▀▀▀          ▄▀ █   █
 ▀▀▀▄ ▀▀▀▀█  ▀▀▀▄ █     ▀▀▀▀▀  ▄▀   █ █ █
▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀   ▀▀▀▀       ▀      ▀ ▀
       SCREENSAVER TEST TOOL`
}

func runScreensaver(cmd *cobra.Command, args []string) {
	// Print header
	fmt.Println(getHeader())
	fmt.Println()

	// Handle list commands
	if listCompositors {
		showCompositorInfo()
		return
	}

	if listOutputs {
		showOutputs()
		return
	}

	// Create config with specified effect and theme
	cfg := config.NewConfig()
	cfg.SetAnimationEffect(effect)
	cfg.SetAnimationTheme(theme)
	cfg.SetDebug(debug)

	// Build screensaver command
	screensaverCmd := cfg.GetScreensaverCommand()

	fmt.Printf("Effect: %s\n", effect)
	fmt.Printf("Theme:  %s\n", theme)
	fmt.Printf("Mode:   %s\n", getMode())
	fmt.Println()

	if singleMonitor {
		// Launch single instance
		fmt.Println("Launching screensaver on current monitor...")
		systemD := systemd.NewSystemD(cfg)
		if err := systemD.LaunchScreensaver(screensaverCmd, "current"); err != nil {
			log.Fatalf("Failed to launch screensaver: %v", err)
		}
		fmt.Println("✓ Screensaver launched")
		fmt.Println("\nPress Ctrl+C to stop")

		// Wait for interrupt
		waitForInterrupt()

		// Stop screensaver
		systemD.StopScreensaver()
	} else {
		// Launch on all monitors
		launchMultiMonitor(cfg, screensaverCmd)
	}
}

func getMode() string {
	if singleMonitor {
		return "Single Monitor"
	}
	return "Multi-Monitor (all outputs)"
}

func showCompositorInfo() {
	fmt.Println("Detecting compositor...")
	fmt.Println()

	comp, err := compositor.DetectCompositor()
	if err != nil {
		fmt.Printf("❌ No supported compositor detected: %v\n", err)
		fmt.Println()
		fmt.Println("Supported compositors:")
		fmt.Println("  • Niri")
		fmt.Println("  • Hyprland")
		fmt.Println("  • Sway")
		return
	}

	fmt.Printf("✓ Detected: %s\n", comp.Name())
	fmt.Println()

	// Get focused output
	focused, err := comp.GetFocusedOutput()
	if err != nil {
		fmt.Printf("Focused output: (unknown)\n")
	} else {
		fmt.Printf("Focused output: %s\n", focused)
	}
}

func showOutputs() {
	comp, err := compositor.DetectCompositor()
	if err != nil {
		fmt.Printf("❌ Failed to detect compositor: %v\n", err)
		return
	}

	fmt.Printf("Compositor: %s\n", comp.Name())
	fmt.Println()

	outputs, err := comp.ListOutputs()
	if err != nil {
		fmt.Printf("❌ Failed to list outputs: %v\n", err)
		return
	}

	fmt.Printf("Found %d output(s):\n", len(outputs))
	fmt.Println()

	for i, output := range outputs {
		focused := ""
		if output.Focused {
			focused = " [FOCUSED]"
		}
		fmt.Printf("%d. %s%s\n", i+1, output.Name, focused)
		if output.Width > 0 && output.Height > 0 {
			fmt.Printf("   Resolution: %dx%d\n", output.Width, output.Height)
		}
	}
}

func launchMultiMonitor(cfg *config.Config, screensaverCmd string) {
	fmt.Println("Detecting compositor...")

	comp, err := compositor.DetectCompositor()
	if err != nil {
		log.Fatalf("Failed to detect compositor: %v", err)
	}

	fmt.Printf("✓ Compositor: %s\n", comp.Name())
	fmt.Println()

	outputs, err := comp.ListOutputs()
	if err != nil {
		log.Fatalf("Failed to list outputs: %v", err)
	}

	fmt.Printf("Found %d output(s):\n", len(outputs))
	for _, output := range outputs {
		fmt.Printf("  • %s\n", output.Name)
	}
	fmt.Println()

	// Get original focus
	originalFocus, _ := comp.GetFocusedOutput()

	// Launch on each output
	systemD := systemd.NewSystemD(cfg)

	fmt.Println("Launching screensaver on all outputs...")
	for i, output := range outputs {
		if debug {
			fmt.Printf("  [%d/%d] Focusing %s...\n", i+1, len(outputs), output.Name)
		}

		if err := comp.FocusOutput(output.Name); err != nil {
			fmt.Printf("  ⚠ Failed to focus %s: %v\n", output.Name, err)
			continue
		}

		time.Sleep(100 * time.Millisecond)

		if err := systemD.LaunchScreensaver(screensaverCmd, output.Name); err != nil {
			fmt.Printf("  ⚠ Failed to launch on %s: %v\n", output.Name, err)
			continue
		}

		if !debug {
			fmt.Printf("  ✓ Launched on %s\n", output.Name)
		}

		if i < len(outputs)-1 {
			time.Sleep(150 * time.Millisecond)
		}
	}

	// Restore focus
	if originalFocus != "" {
		comp.FocusOutput(originalFocus)
	}

	fmt.Println()
	fmt.Printf("✓ Screensaver launched on %d output(s)\n", systemD.GetProcessCount())
	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for interrupt
	waitForInterrupt()

	// Stop all screensavers
	fmt.Println("\nStopping screensaver...")
	systemD.StopScreensaver()
	fmt.Println("✓ Stopped")
}

func waitForInterrupt() {
	// Wait forever (user will Ctrl+C)
	select {}
}
