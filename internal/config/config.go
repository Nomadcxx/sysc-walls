// config.go - Configuration management
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Available animation effects
var AvailableEffects = []string{
	"matrix",
	"matrix-art",
	"fire",
	"fireworks",
	"rain",
	"rain-art",
	"beams",
	"beam-text",
	"decrypt",
	"pour",
	"aquarium",
	"print",
	"blackhole",
	"ring-text",
}

// Available color themes
var AvailableThemes = []string{
	"nord",
	"dracula",
	"gruvbox",
	"tokyo-night",
	"catppuccin",
	"material",
	"solarized",
	"monochrome",
	"trainsishardjob",
	"rama",
	"eldritch",
	"dark",
}

// Config represents the daemon configuration
type Config struct {
	idleTimeout        time.Duration
	minDuration        time.Duration
	debug              bool
	animationEffect    string
	animationTheme     string
	cycleAnimations    bool
	terminalKitty      bool
	terminalFullscreen bool
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		idleTimeout:        300 * time.Second, // 5 minutes default
		minDuration:        30 * time.Second,  // 30 seconds default
		debug:              false,
		animationEffect:    "matrix-art",
		animationTheme:     "rama",
		cycleAnimations:    false,
		terminalKitty:      true,
		terminalFullscreen: true,
	}
}

// LoadFromFile loads configuration from a file
func (c *Config) LoadFromFile(configPath string) error {
	// Expand home directory if needed
	expandedPath := os.ExpandEnv(configPath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to read the config file
	file, err := os.Open(expandedPath)
	if err != nil {
		// Config file doesn't exist, create a default one
		return c.createDefaultConfig(expandedPath)
	}
	defer file.Close()

	// Parse the config file
	// Simple INI-style format
	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section header [section]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			continue
		}

		// Split by '=' to get key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Prepend section to key if we're in a section
		if currentSection != "" {
			key = currentSection + "." + key
		}

		c.parseConfigLine(key, value)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// parseConfigLine parses a single configuration line
func (c *Config) parseConfigLine(key, value string) {
	switch key {
	case "idle.timeout":
		if duration, err := parseDuration(value); err == nil {
			c.idleTimeout = duration
		}
	case "idle.min_duration":
		if duration, err := parseDuration(value); err == nil {
			c.minDuration = duration
		}
	case "daemon.debug":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			c.debug = boolVal
		}
	case "animation.effect":
		if IsValidEffect(value) {
			c.animationEffect = value
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Invalid animation effect '%s' in config file. Using default.\n", value)
			fmt.Fprintf(os.Stderr, "Available effects: %s\n", strings.Join(AvailableEffects, ", "))
		}
	case "animation.theme":
		if IsValidTheme(value) {
			c.animationTheme = value
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Invalid animation theme '%s' in config file. Using default.\n", value)
			fmt.Fprintf(os.Stderr, "Available themes: %s\n", strings.Join(AvailableThemes, ", "))
		}
	case "animation.cycle":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			c.cycleAnimations = boolVal
		}
	case "terminal.kitty":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			c.terminalKitty = boolVal
		}
	case "terminal.fullscreen":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			c.terminalFullscreen = boolVal
		}
	}
}

// parseDuration parses a duration string (supports seconds, minutes, etc.)
func parseDuration(value string) (time.Duration, error) {
	// Simple parser for common duration formats
	if strings.HasSuffix(value, "s") {
		if seconds, err := strconv.Atoi(strings.TrimSuffix(value, "s")); err == nil {
			if seconds < 0 {
				return 0, fmt.Errorf("duration cannot be negative: %s", value)
			}
			return time.Duration(seconds) * time.Second, nil
		}
	} else if strings.HasSuffix(value, "m") {
		if minutes, err := strconv.Atoi(strings.TrimSuffix(value, "m")); err == nil {
			if minutes < 0 {
				return 0, fmt.Errorf("duration cannot be negative: %s", value)
			}
			return time.Duration(minutes) * time.Minute, nil
		}
	} else if strings.HasSuffix(value, "h") {
		if hours, err := strconv.Atoi(strings.TrimSuffix(value, "h")); err == nil {
			if hours < 0 {
				return 0, fmt.Errorf("duration cannot be negative: %s", value)
			}
			return time.Duration(hours) * time.Hour, nil
		}
	}

	// Try parsing as a number of seconds
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("duration cannot be negative: %s", value)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s", value)
}

// createDefaultConfig creates a default configuration file
func (c *Config) createDefaultConfig(configPath string) error {
	// Create default config file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write default config
	lines := []string{
		"# sysc-walls daemon configuration",
		"",
		"[idle]",
		fmt.Sprintf("timeout = %s", formatDuration(c.idleTimeout)),
		fmt.Sprintf("min_duration = %s", formatDuration(c.minDuration)),
		"",
		"[daemon]",
		fmt.Sprintf("debug = %t", c.debug),
		"",
		"[animation]",
		fmt.Sprintf("effect = %s", c.animationEffect),
		"# Available effects: " + strings.Join(AvailableEffects, ", "),
		fmt.Sprintf("theme = %s", c.animationTheme),
		"# Available themes: " + strings.Join(AvailableThemes, ", "),
		fmt.Sprintf("cycle = %t", c.cycleAnimations),
		"",
		"[terminal]",
		fmt.Sprintf("kitty = %t", c.terminalKitty),
		fmt.Sprintf("fullscreen = %t", c.terminalFullscreen),
	}

	for _, line := range lines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to config file: %w", err)
		}
	}

	return nil
}

// formatDuration formats a duration as a string
func formatDuration(d time.Duration) string {
	if d >= time.Hour {
		hours := int(d / time.Hour)
		return fmt.Sprintf("%dh", hours)
	} else if d >= time.Minute {
		minutes := int(d / time.Minute)
		return fmt.Sprintf("%dm", minutes)
	} else {
		seconds := int(d / time.Second)
		return fmt.Sprintf("%ds", seconds)
	}
}

// SaveToFile saves the configuration to a file
func (c *Config) SaveToFile(configPath string) error {
	// Expand home directory if needed
	expandedPath := os.ExpandEnv(configPath)

	// Use default path if not provided
	if expandedPath == "" {
		expandedPath = os.ExpandEnv("~/.config/sysc-walls/daemon.conf")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create config file
	file, err := os.Create(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write config
	lines := []string{
		"# sysc-walls daemon configuration",
		"",
		"[idle]",
		fmt.Sprintf("timeout = %s", formatDuration(c.idleTimeout)),
		fmt.Sprintf("min_duration = %s", formatDuration(c.minDuration)),
		"",
		"[daemon]",
		fmt.Sprintf("debug = %t", c.debug),
		"",
		"[animation]",
		fmt.Sprintf("effect = %s", c.animationEffect),
		"# Available effects: " + strings.Join(AvailableEffects, ", "),
		fmt.Sprintf("theme = %s", c.animationTheme),
		"# Available themes: " + strings.Join(AvailableThemes, ", "),
		fmt.Sprintf("cycle = %t", c.cycleAnimations),
		"",
		"[terminal]",
		fmt.Sprintf("kitty = %t", c.terminalKitty),
		fmt.Sprintf("fullscreen = %t", c.terminalFullscreen),
	}

	for _, line := range lines {
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write to config file: %w", err)
		}
	}

	return nil
}

// GetIdleTimeout returns the idle timeout duration
func (c *Config) GetIdleTimeout() time.Duration {
	return c.idleTimeout
}

// SetIdleTimeout sets the idle timeout duration
func (c *Config) SetIdleTimeout(timeoutStr string) error {
	duration, err := parseDuration(timeoutStr)
	if err != nil {
		return err
	}
	c.idleTimeout = duration
	return nil
}

// GetMinDuration returns the minimum duration the screensaver should run
func (c *Config) GetMinDuration() time.Duration {
	return c.minDuration
}

// IsDebug returns whether debug mode is enabled
func (c *Config) IsDebug() bool {
	return c.debug
}

// SetDebug sets debug mode
func (c *Config) SetDebug(debug bool) {
	c.debug = debug
}

// GetAnimationEffect returns the default animation effect
func (c *Config) GetAnimationEffect() string {
	return c.animationEffect
}

// SetAnimationEffect sets the animation effect with validation
func (c *Config) SetAnimationEffect(effect string) error {
	if !IsValidEffect(effect) {
		return fmt.Errorf("invalid animation effect: %s\nAvailable effects: %s", effect, strings.Join(AvailableEffects, ", "))
	}
	c.animationEffect = effect
	return nil
}

// GetAnimationTheme returns the default animation theme
func (c *Config) GetAnimationTheme() string {
	return c.animationTheme
}

// SetAnimationTheme sets the animation theme with validation
func (c *Config) SetAnimationTheme(theme string) error {
	if !IsValidTheme(theme) {
		return fmt.Errorf("invalid animation theme: %s\nAvailable themes: %s", theme, strings.Join(AvailableThemes, ", "))
	}
	c.animationTheme = theme
	return nil
}

// IsValidEffect checks if the effect is valid
func IsValidEffect(effect string) bool {
	for _, e := range AvailableEffects {
		if e == effect {
			return true
		}
	}
	return false
}

// IsValidTheme checks if the theme is valid
func IsValidTheme(theme string) bool {
	for _, t := range AvailableThemes {
		if t == theme {
			return true
		}
	}
	return false
}

// ShouldCycleAnimations returns whether animations should be cycled
func (c *Config) ShouldCycleAnimations() bool {
	return c.cycleAnimations
}

// IsTerminalKitty returns whether to use kitty terminal
func (c *Config) IsTerminalKitty() bool {
	return c.terminalKitty
}

// SetTerminalKitty sets the terminal preference
func (c *Config) SetTerminalKitty(kitty bool) {
	c.terminalKitty = kitty
}

// IsTerminalFullscreen returns whether to use fullscreen mode
func (c *Config) IsTerminalFullscreen() bool {
	return c.terminalFullscreen
}

// SetTerminalFullscreen sets the fullscreen preference
func (c *Config) SetTerminalFullscreen(fullscreen bool) {
	c.terminalFullscreen = fullscreen
}

// GetTerminalLauncher returns the command to launch the terminal
func (c *Config) GetTerminalLauncher() string {
	if c.terminalKitty {
		return "kitty"
	}
	return "xterm"
}

// GetTerminalArgs returns the arguments for the terminal launcher
func (c *Config) GetTerminalArgs() []string {
	args := []string{}

	if c.terminalFullscreen {
		args = append(args, "--start-as=fullscreen")
	}

	return args
}

// GetScreensaverCommand returns the command to launch the screensaver
func (c *Config) GetScreensaverCommand() string {
	terminal := c.GetTerminalLauncher()
	args := c.GetTerminalArgs()
	effect := c.GetAnimationEffect()
	theme := c.GetAnimationTheme()

	// Build the command to launch kitty with the display binary
	// Use unique class name to identify screensaver window
	parts := []string{terminal}
	parts = append(parts, args...)
	parts = append(parts, "--class", "sysc-walls-screensaver")
	parts = append(parts, "/usr/local/bin/sysc-walls-display", "--effect", effect, "--theme", theme, "--fullscreen")

	return strings.Join(parts, " ")
}
