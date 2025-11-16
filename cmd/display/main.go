// main.go - Entry point for display component
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Nomadcxx/sysc-walls/internal/animations"
	"github.com/Nomadcxx/sysc-walls/internal/clock"
	"github.com/Nomadcxx/sysc-walls/internal/version"
	"github.com/Nomadcxx/sysc-walls/pkg/utils"

	syscGo "github.com/Nomadcxx/sysc-Go/animations"
)

// loadTextContent loads text from a file with fallback to default SYSC.txt
func loadTextContent(customPath string, debug bool) string {
	// Try custom path first if provided
	if customPath != "" {
		if content, err := os.ReadFile(customPath); err == nil {
			if debug {
				fmt.Fprintf(os.Stderr, "Loaded text from: %s\n", customPath)
			}
			return strings.TrimSpace(string(content))
		} else if debug {
			fmt.Fprintf(os.Stderr, "Failed to load custom text file %s: %v\n", customPath, err)
		}
	}

	// Try default SYSC.txt from config directory (primary location)
	homeDir := os.Getenv("HOME")
	defaultPaths := []string{
		filepath.Join(homeDir, ".config", "sysc-walls", "ascii", "SYSC.txt"),
		filepath.Join(homeDir, ".local", "share", "syscgo", "walls", "SYSC.txt"),
		filepath.Join(homeDir, ".local", "share", "sysc-walls", "SYSC.txt"),
		"/usr/share/sysc-walls/ascii/SYSC.txt", // AUR package location
		"/usr/share/syscgo/assets/SYSC.txt",
		"/usr/share/sysc-walls/SYSC.txt",
		"sysc-Go/assets/SYSC.txt", // For development
	}

	for _, path := range defaultPaths {
		if content, err := os.ReadFile(path); err == nil {
			if debug {
				fmt.Fprintf(os.Stderr, "Loaded ASCII art from: %s\n", path)
			}
			return strings.TrimSpace(string(content))
		}
	}

	// Final fallback - simple text
	if debug {
		fmt.Fprintf(os.Stderr, "Warning: No ASCII art files found, using fallback text\n")
	}
	return "SYSC-WALLS"
}

// isTextBasedEffect checks if an effect uses text content
// Now uses sysc-Go registry instead of hardcoded list
func isTextBasedEffect(effect string) bool{
	return syscGo.IsTextBasedEffect(effect)
}

// dimANSIColors reduces the intensity of ANSI RGB colors by a factor
// factor should be between 0.0 (black) and 1.0 (original)
func dimANSIColors(text string, factor float64) string {
	// Match ANSI RGB color codes: \x1b[38;2;R;G;Bm
	re := regexp.MustCompile(`\x1b\[38;2;(\d+);(\d+);(\d+)m`)

	return re.ReplaceAllStringFunc(text, func(match string) string {
		// Extract RGB values
		parts := re.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}

		r, _ := strconv.Atoi(parts[1])
		g, _ := strconv.Atoi(parts[2])
		b, _ := strconv.Atoi(parts[3])

		// Dim the colors
		r = int(float64(r) * factor)
		g = int(float64(g) * factor)
		b = int(float64(b) * factor)

		// Reconstruct ANSI code
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	})
}

// dimLineRegion dims a specific region of a line (from start to end column)
func dimLineRegion(line string, startCol, endCol int, factor float64) string {
	// Convert to runes to handle multi-byte characters and ANSI codes
	runes := []rune(line)
	if startCol < 0 || startCol >= len(runes) {
		return line
	}
	if endCol > len(runes) {
		endCol = len(runes)
	}

	// Extract the region, dim it, and reconstruct
	before := string(runes[:startCol])
	region := string(runes[startCol:endCol])
	after := string(runes[endCol:])

	return before + dimANSIColors(region, factor) + after
}

// overlayLine overlays overlay text onto base
// For now, just returns overlay (base is already dimmed separately)
func overlayLine(base, overlay string, width int) string {
	// The overlay contains bright datetime text
	// The base is already dimmed in the calling function
	// Simply return the overlay which will show bright text on dimmed background
	return overlay
}

// overlayDateTime overlays date-time on animation output
func overlayDateTime(animOutput string, width, height int, isTextBased bool, position string) string {
	// Get datetime lines
	datetimeLines := clock.RenderDateTime()

	// Split animation output into lines
	animLines := strings.Split(animOutput, "\n")

	// Ensure we have enough lines
	for len(animLines) < height {
		animLines = append(animLines, strings.Repeat(" ", width))
	}

	if isTextBased {
		// For text-based effects: append datetime below the animation
		// Find the last non-empty line
		lastNonEmpty := len(animLines) - 1
		for lastNonEmpty >= 0 && strings.TrimSpace(animLines[lastNonEmpty]) == "" {
			lastNonEmpty--
		}

		// Insert datetime starting a few lines below the text
		startLine := lastNonEmpty + 3
		if startLine >= len(animLines) {
			startLine = len(animLines) - len(datetimeLines) - 1
		}
		if startLine < 0 {
			startLine = 0
		}

		// Center and insert datetime lines
		centeredDateTime := clock.CenterLines(datetimeLines, width)
		for i, line := range centeredDateTime {
			if startLine+i < len(animLines) {
				animLines[startLine+i] = line
			}
		}
	} else {
		// For non-text effects: overlay with dimming at specified position
		// Calculate starting position based on user preference
		var startLine int
		switch position {
		case "top":
			startLine = 2 // Small margin from top
		case "center":
			startLine = (height - len(datetimeLines)) / 2
		case "bottom":
			startLine = height - len(datetimeLines) - 2
		default:
			startLine = height - len(datetimeLines) - 2 // fallback to bottom
		}

		// Ensure we don't go out of bounds
		if startLine < 0 {
			startLine = 0
		}
		if startLine+len(datetimeLines) > len(animLines) {
			startLine = len(animLines) - len(datetimeLines)
			if startLine < 0 {
				startLine = 0
			}
		}

		// Get datetime lines with bright colors
		centeredDateTime := clock.CenterLinesBright(datetimeLines, width)

		// Dim the animation area behind datetime and overlay
		for i, dtLine := range centeredDateTime {
			lineIdx := startLine + i
			if lineIdx >= len(animLines) {
				break
			}

			// Dim the entire line where datetime will appear
			animLines[lineIdx] = dimANSIColors(animLines[lineIdx], 0.35)

			// Overlay datetime on top (character by character to preserve spacing)
			animLines[lineIdx] = overlayLine(animLines[lineIdx], dtLine, width)
		}
	}

	return strings.Join(animLines, "\n")
}

func main() {
	// Parse command line flags
	var (
		effect           = flag.String("effect", "matrix", "Animation effect to display")
		theme            = flag.String("theme", "dracula", "Color theme for animation")
		file             = flag.String("file", "", "Text file for text-based effects")
		datetime         = flag.Bool("datetime", false, "Show date and time overlay")
		datetimePosition = flag.String("datetime-position", "bottom", "Position of datetime overlay: top, center, bottom")
		showVersion      = flag.Bool("version", false, "Show version information")
		showVersionV     = flag.Bool("v", false, "Show version information (shorthand)")
		debug            = flag.Bool("debug", false, "Enable debug logging")
		noClear      = flag.Bool("no-clear", false, "Don't clear the screen before animation")
		fullScreen   = flag.Bool("fullscreen", false, "Run in fullscreen mode")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion || *showVersionV {
		fmt.Printf("%s\n", version.GetFullVersion())
		os.Exit(0)
	}

	// If fullscreen is requested, give terminal time to resize
	if *fullScreen {
		// Give terminal time to fully enter fullscreen mode
		time.Sleep(300 * time.Millisecond)
	}

	// Get terminal dimensions AFTER possibly entering fullscreen
	width, height, err := utils.GetTerminalSize()
	if err != nil && *debug {
		fmt.Fprintf(os.Stderr, "Error getting terminal size: %v\n", err)
	}

	// Retry getting size a few times if it looks wrong
	for i := 0; i < 10 && (width < 100 || height < 40); i++ {
		time.Sleep(100 * time.Millisecond)
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

	// Load text content for text-based effects
	var textContent string
	if isTextBasedEffect(*effect) {
		textContent = loadTextContent(*file, *debug)
	}

	// Create animation based on effect
	anim, err := animations.CreateAnimationWithText(*effect, width, height, *theme, textContent)
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

	// Screensaver runs infinitely
	totalFrames := -1

	// Store values for use in goroutine
	showDateTime := *datetime
	effectName := *effect
	isTextEffect := isTextBasedEffect(effectName)

	if *debug {
		fmt.Printf("Starting animation: %s with theme %s\n", *effect, *theme)
		fmt.Printf("Terminal size: %dx%d\n", width, height)
		fmt.Printf("Duration: infinite (screensaver mode)\n")
		fmt.Printf("DateTime overlay: %v\n", showDateTime)
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

				// Get rendered output
				output := anim.Render()

				// Apply datetime overlay if enabled
				if showDateTime {
					output = overlayDateTime(output, width, height, isTextEffect, *datetimePosition)
				}

				// Print animation
				fmt.Print(output)

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
