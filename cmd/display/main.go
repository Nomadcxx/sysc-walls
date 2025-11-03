// main.go - Entry point for display component
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/animations"
	"github.com/Nomadcxx/sysc-walls/pkg/utils"
)

func main() {
	// Parse command line flags
	var (
		effect     = flag.String("effect", "matrix", "Animation effect to display")
		theme      = flag.String("theme", "dracula", "Color theme for animation")
		duration   = flag.Int("duration", 0, "Duration in seconds (0 for infinite)")
		debug      = flag.Bool("debug", false, "Enable debug logging")
		noClear    = flag.Bool("no-clear", false, "Don't clear the screen before animation")
		fullScreen = flag.Bool("fullscreen", false, "Run in fullscreen mode")
	)
	flag.Parse()

	// If fullscreen is requested, give terminal time to resize
	if *fullScreen {
		// Give terminal time to fully enter fullscreen mode
		time.Sleep(100 * time.Millisecond)
	}

	// Get terminal dimensions AFTER possibly entering fullscreen
	width, height, err := utils.GetTerminalSize()
	if err != nil && *debug {
		fmt.Fprintf(os.Stderr, "Error getting terminal size: %v\n", err)
	}
	
	// Retry getting size a few times if it looks wrong
	for i := 0; i < 5 && (width < 100 || height < 40); i++ {
		time.Sleep(50 * time.Millisecond)
		width, height, err = utils.GetTerminalSize()
		if *debug {
			fmt.Fprintf(os.Stderr, "Retry %d: size=%dx%d\n", i+1, width, height)
		}
	}
	
	if *debug {
		fmt.Fprintf(os.Stderr, "Final terminal size: %dx%d\n", width, height)
	}

	// Setup terminal
	if !*noClear {
		utils.SetupTerminal()
	}
	defer utils.RestoreTerminal()

	// Create animation based on effect
	anim, err := animations.CreateAnimation(*effect, width, height, *theme)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating animation: %v\n", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown and resize
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Handle window resize
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	// Animation loop
	frame := 0
	ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
	defer ticker.Stop()

	// Determine animation duration
	var totalFrames int
	if *duration > 0 {
		totalFrames = *duration * 20 // 20 FPS
	} else {
		totalFrames = -1 // Infinite
	}

	if *debug {
		fmt.Printf("Starting animation: %s with theme %s\n", *effect, *theme)
		fmt.Printf("Terminal size: %dx%d\n", width, height)
		fmt.Printf("Duration: %d frames (%s)\n", totalFrames,
			map[bool]string{true: "infinite", false: fmt.Sprintf("%d seconds", *duration)}[!(*duration > 0)])
	}

	// Animation goroutine
	go func() {
		for frame < totalFrames || totalFrames == -1 {
			select {
			case <-ticker.C:
				// Update animation
				anim.Update(frame)

				// Render animation
				if !*noClear && frame == 0 {
					utils.ClearScreen()
				}

				// Print animation
				fmt.Print(anim.Render())

				// Move cursor to top
				fmt.Print("\033[H")

				frame++
			case <-c:
				// Received interrupt or termination signal
				if *debug {
					fmt.Printf("Received interrupt, stopping after %d frames\n", frame)
				}
				os.Exit(0)
			case <-sigwinch:
				// Window was resized
				newWidth, newHeight, err := utils.GetTerminalSize()
				if err != nil {
					if *debug {
						fmt.Fprintf(os.Stderr, "Error getting terminal size: %v\n", err)
					}
				} else {
					if newWidth != width || newHeight != height {
						if *debug {
							fmt.Printf("Terminal resized from %dx%d to %dx%d\n", width, height, newWidth, newHeight)
						}
						width, height = newWidth, newHeight
						anim.Resize(width, height)
					}
				}
			}
		}
	}()

	// Wait for interrupt or termination signal
	<-c
}
