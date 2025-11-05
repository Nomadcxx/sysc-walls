// main.go - Test CLI for sysc-walls screensaver with comprehensive debugging
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/compositor"
	"github.com/Nomadcxx/sysc-walls/internal/config"
	"github.com/Nomadcxx/sysc-walls/internal/systemd"
	"github.com/spf13/cobra"
)

var (
	// Basic options
	effect        string
	theme         string
	singleMonitor bool

	// Output control
	listCompositors bool
	listOutputs     bool
	testOutput      string

	// Debugging flags
	debug      bool
	verbose    bool
	dryRun     bool
	testFocus  bool
	traceFocus bool

	// Timing control (in milliseconds)
	focusDelay  int
	launchDelay int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "test-screensaver",
		Short: "Test and debug sysc-walls multi-monitor screensaver",
		Long:  getHeader() + "\n\nA testing and debugging tool for sysc-walls screensaver.\nLaunch screensavers immediately without waiting for idle timeout.",
		Run:   runScreensaver,
	}

	// Basic options
	rootCmd.Flags().StringVarP(&effect, "effect", "e", "matrix", "Animation effect (matrix, fire, rain, etc.)")
	rootCmd.Flags().StringVarP(&theme, "theme", "t", "nord", "Color theme (nord, dracula, gruvbox, etc.)")
	rootCmd.Flags().BoolVarP(&singleMonitor, "single", "s", false, "Launch on single monitor only")

	// Output control
	rootCmd.Flags().BoolVarP(&listCompositors, "list-compositors", "c", false, "List detected compositor and exit")
	rootCmd.Flags().BoolVarP(&listOutputs, "list-outputs", "o", false, "List all outputs and exit")
	rootCmd.Flags().StringVar(&testOutput, "test-output", "", "Test on specific output only (e.g., DP-1)")

	// Debugging flags
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output with detailed timing")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without actually launching")
	rootCmd.Flags().BoolVar(&testFocus, "test-focus", false, "Test focusing each monitor without launching screensaver")
	rootCmd.Flags().BoolVar(&traceFocus, "trace-focus", false, "Show focus changes in real-time")

	// Timing control
	rootCmd.Flags().IntVar(&focusDelay, "focus-delay", 100, "Milliseconds to wait after focusing (default 100)")
	rootCmd.Flags().IntVar(&launchDelay, "launch-delay", 150, "Milliseconds to wait between launches (default 150)")

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

	// Handle focus testing
	if testFocus {
		runFocusTest()
		return
	}

	// Create config with specified effect and theme
	cfg := config.NewConfig()
	cfg.SetAnimationEffect(effect)
	cfg.SetAnimationTheme(theme)
	cfg.SetDebug(debug || verbose)

	// Build screensaver command
	screensaverCmd := cfg.GetScreensaverCommand()

	// Show configuration
	showConfig(screensaverCmd)

	if singleMonitor || testOutput != "" {
		// Launch single instance
		launchSingle(cfg, screensaverCmd)
	} else {
		// Launch on all monitors
		launchMultiMonitor(cfg, screensaverCmd)
	}
}

func showConfig(cmd string) {
	fmt.Printf("Effect:       %s\n", effect)
	fmt.Printf("Theme:        %s\n", theme)
	fmt.Printf("Mode:         %s\n", getMode())

	if verbose {
		fmt.Printf("Focus delay:  %dms\n", focusDelay)
		fmt.Printf("Launch delay: %dms\n", launchDelay)
		fmt.Printf("Command:      %s\n", cmd)
	}

	if dryRun {
		fmt.Printf("\n⚠ DRY RUN MODE - No screensavers will actually launch\n")
	}

	fmt.Println()
}

func getMode() string {
	if testOutput != "" {
		return fmt.Sprintf("Specific Output (%s)", testOutput)
	}
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
		fmt.Println()
		fmt.Println("Make sure you're running on Wayland and one of these compositors is active.")
		return
	}

	fmt.Printf("✓ Detected: %s\n", comp.Name())
	fmt.Println()

	// Get focused output
	focused, err := comp.GetFocusedOutput()
	if err != nil {
		fmt.Printf("Focused output: (unknown - %v)\n", err)
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

	fmt.Println()
	fmt.Println("Tip: Use --test-output <name> to test on a specific output")
}

func runFocusTest() {
	fmt.Println("Focus Testing Mode")
	fmt.Println("==================")
	fmt.Println()

	comp, err := compositor.DetectCompositor()
	if err != nil {
		log.Fatalf("Failed to detect compositor: %v", err)
	}

	fmt.Printf("Compositor: %s\n", comp.Name())
	fmt.Println()

	outputs, err := comp.ListOutputs()
	if err != nil {
		log.Fatalf("Failed to list outputs: %v", err)
	}

	fmt.Printf("Testing focus on %d output(s)...\n", len(outputs))
	fmt.Println()

	originalFocus, _ := comp.GetFocusedOutput()
	fmt.Printf("Original focus: %s\n", originalFocus)
	fmt.Println()

	for i, output := range outputs {
		fmt.Printf("[%d/%d] Focusing %s...", i+1, len(outputs), output.Name)

		start := time.Now()
		if err := comp.FocusOutput(output.Name); err != nil {
			fmt.Printf(" ❌ FAILED: %v\n", err)
			continue
		}
		elapsed := time.Since(start)

		fmt.Printf(" ✓ (%dms)\n", elapsed.Milliseconds())

		// Verify focus changed
		time.Sleep(time.Duration(focusDelay) * time.Millisecond)
		currentFocus, _ := comp.GetFocusedOutput()
		if currentFocus != output.Name {
			fmt.Printf("  ⚠ Warning: Focus may not have changed (current: %s)\n", currentFocus)
		}

		time.Sleep(time.Second) // Pause between focuses for visibility
	}

	fmt.Println()
	fmt.Printf("Restoring focus to: %s\n", originalFocus)
	if originalFocus != "" {
		comp.FocusOutput(originalFocus)
	}

	fmt.Println()
	fmt.Println("✓ Focus test complete")
}

func launchSingle(cfg *config.Config, screensaverCmd string) {
	outputName := "current"
	if testOutput != "" {
		outputName = testOutput

		// Verify output exists and focus it
		comp, err := compositor.DetectCompositor()
		if err != nil {
			log.Fatalf("Failed to detect compositor: %v", err)
		}

		outputs, err := comp.ListOutputs()
		if err != nil {
			log.Fatalf("Failed to list outputs: %v", err)
		}

		found := false
		for _, output := range outputs {
			if output.Name == testOutput {
				found = true
				break
			}
		}

		if !found {
			log.Fatalf("Output '%s' not found. Use --list-outputs to see available outputs.", testOutput)
		}

		fmt.Printf("Focusing %s...\n", testOutput)
		if err := comp.FocusOutput(testOutput); err != nil {
			log.Fatalf("Failed to focus %s: %v", testOutput, err)
		}
		time.Sleep(time.Duration(focusDelay) * time.Millisecond)
	}

	if dryRun {
		fmt.Printf("Would launch screensaver on: %s\n", outputName)
		return
	}

	fmt.Printf("Launching screensaver on %s...\n", outputName)
	systemD := systemd.NewSystemD(cfg)

	start := time.Now()
	if err := systemD.LaunchScreensaver(screensaverCmd, outputName); err != nil {
		log.Fatalf("Failed to launch screensaver: %v", err)
	}
	elapsed := time.Since(start)

	if verbose {
		fmt.Printf("✓ Launched in %dms\n", elapsed.Milliseconds())
	} else {
		fmt.Println("✓ Screensaver launched")
	}

	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for interrupt
	waitForInterrupt(systemD)
}

func launchMultiMonitor(cfg *config.Config, screensaverCmd string) {
	if verbose {
		fmt.Println("Multi-Monitor Launch Sequence")
		fmt.Println("==============================")
		fmt.Println()
	}

	fmt.Println("Detecting compositor...")
	comp, err := compositor.DetectCompositor()
	if err != nil {
		log.Fatalf("Failed to detect compositor: %v", err)
	}

	fmt.Printf("✓ Compositor: %s\n", comp.Name())
	if verbose {
		fmt.Println()
	}

	outputs, err := comp.ListOutputs()
	if err != nil {
		log.Fatalf("Failed to list outputs: %v", err)
	}

	fmt.Printf("\nFound %d output(s):\n", len(outputs))
	for _, output := range outputs {
		focusMarker := ""
		if output.Focused {
			focusMarker = " [focused]"
		}
		fmt.Printf("  • %s%s\n", output.Name, focusMarker)
	}
	fmt.Println()

	// Get original focus
	originalFocus, _ := comp.GetFocusedOutput()
	if verbose && originalFocus != "" {
		fmt.Printf("Original focus: %s\n", originalFocus)
		fmt.Println()
	}

	if dryRun {
		fmt.Println("DRY RUN - Would perform these actions:")
		for i, output := range outputs {
			fmt.Printf("  %d. Focus %s (wait %dms)\n", i+1, output.Name, focusDelay)
			fmt.Printf("     Launch screensaver\n")
			if i < len(outputs)-1 {
				fmt.Printf("     Wait %dms\n", launchDelay)
			}
		}
		fmt.Printf("  %d. Restore focus to %s\n", len(outputs)+1, originalFocus)
		return
	}

	// Launch on each output
	systemD := systemd.NewSystemD(cfg)

	fmt.Println("Launching screensaver on all outputs...")
	totalStart := time.Now()

	for i, output := range outputs {
		if verbose {
			fmt.Printf("\n[%d/%d] Processing %s:\n", i+1, len(outputs), output.Name)
		}

		// Focus output
		if verbose || traceFocus {
			fmt.Printf("  → Focusing %s...", output.Name)
		}

		focusStart := time.Now()
		if err := comp.FocusOutput(output.Name); err != nil {
			fmt.Printf(" ❌ Failed: %v\n", err)
			continue
		}
		focusElapsed := time.Since(focusStart)

		if verbose || traceFocus {
			fmt.Printf(" ✓ (%dms)\n", focusElapsed.Milliseconds())
		}

		// Wait for focus to be applied
		if verbose {
			fmt.Printf("  → Waiting %dms for focus...\n", focusDelay)
		}
		time.Sleep(time.Duration(focusDelay) * time.Millisecond)

		// Launch screensaver
		if verbose {
			fmt.Printf("  → Launching screensaver...")
		}

		launchStart := time.Now()
		if err := systemD.LaunchScreensaver(screensaverCmd, output.Name); err != nil {
			fmt.Printf(" ❌ Failed: %v\n", err)
			continue
		}
		launchElapsed := time.Since(launchStart)

		if verbose {
			fmt.Printf(" ✓ (%dms)\n", launchElapsed.Milliseconds())
		} else if !debug {
			fmt.Printf("  ✓ Launched on %s\n", output.Name)
		}

		// Delay between launches
		if i < len(outputs)-1 {
			if verbose {
				fmt.Printf("  → Waiting %dms before next output...\n", launchDelay)
			}
			time.Sleep(time.Duration(launchDelay) * time.Millisecond)
		}
	}

	totalElapsed := time.Since(totalStart)

	// Restore focus
	if originalFocus != "" {
		if verbose {
			fmt.Printf("\nRestoring focus to: %s\n", originalFocus)
		}
		comp.FocusOutput(originalFocus)
	}

	fmt.Println()
	processCount := systemD.GetProcessCount()
	if verbose {
		fmt.Printf("✓ Launch complete in %dms\n", totalElapsed.Milliseconds())
		fmt.Printf("  Processes: %d/%d successful\n", processCount, len(outputs))
		if pids, err := systemD.GetPIDs(); err == nil {
			fmt.Printf("  PIDs: %v\n", pids)
		}
	} else {
		fmt.Printf("✓ Screensaver launched on %d output(s)\n", processCount)
	}

	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for interrupt
	waitForInterrupt(systemD)
}

func waitForInterrupt(systemD *systemd.SystemD) {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	<-sigChan

	// Stop all screensavers
	fmt.Println("\nStopping screensaver...")
	if err := systemD.StopScreensaver(); err != nil {
		if verbose {
			fmt.Printf("Stop completed with error: %v\n", err)
		}
	}
	fmt.Println("✓ Stopped")
}
