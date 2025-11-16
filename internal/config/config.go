// config.go - Configuration management
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	syscGo "github.com/Nomadcxx/sysc-Go/animations"
)

// Available animation effects - auto-generated from sysc-Go registry
var AvailableEffects = syscGo.GetEffectNames()

// Available color themes - auto-generated from sysc-Go registry
var AvailableThemes = syscGo.GetThemeNames()

// MinimumSyscGoVersion is the minimum required version of sysc-Go
const MinimumSyscGoVersion = "1.0.1"

// Config represents the daemon configuration
type Config struct {
	idleTimeout         time.Duration
	minDuration         time.Duration
	debug               bool
	animationEffect     string
	animationTheme      string
	animationFile       string // Custom artwork file path for text-based effects
	animationDatetime   bool   // Show date/time overlay (only for non-text effects)
	datetimePosition    string // Position of datetime: "top", "center", "bottom"
	cycleAnimations     bool
	terminalKitty       bool
	terminalFullscreen  bool
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		idleTimeout:        300 * time.Second, // 5 minutes default
		minDuration:        30 * time.Second,  // 30 seconds default
		debug:              false,
		animationEffect:    "matrix-art",
		animationTheme:     "rama",
		animationDatetime:  false,    // datetime overlay disabled by default
		datetimePosition:   "bottom", // datetime position: top, center, or bottom
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
	case "animation.file":
		// Expand environment variables and home directory
		expandedPath := os.ExpandEnv(value)
		expandedPath = strings.Replace(expandedPath, "~", os.Getenv("HOME"), 1)
		// Validate that file path is absolute
		if filepath.IsAbs(expandedPath) {
			c.animationFile = expandedPath
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Animation file path must be absolute, got '%s'. Ignoring.\n", value)
		}
	case "animation.datetime":
		if boolVal, err := strconv.ParseBool(value); err == nil {
			c.animationDatetime = boolVal
		}
	case "datetime.position":
		// Validate position value
		value = strings.ToLower(value)
		if value == "top" || value == "center" || value == "centre" || value == "bottom" {
			// Normalize "centre" to "center"
			if value == "centre" {
				value = "center"
			}
			c.datetimePosition = value
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Invalid datetime position '%s'. Must be top, center, or bottom. Using default.\n", value)
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

// CheckSyscGoVersion verifies that the sysc-Go library version is compatible
func CheckSyscGoVersion() error {
	actualVersion := syscGo.GetLibraryVersion()
	if actualVersion < MinimumSyscGoVersion {
		return fmt.Errorf("sysc-Go version mismatch: found %s, requires >= %s",
			actualVersion, MinimumSyscGoVersion)
	}
	return nil
}

// GetSyscGoVersion returns the current sysc-Go library version
func GetSyscGoVersion() string {
	return syscGo.GetLibraryVersion()
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

// GetAnimationFile returns the custom animation file path
func (c *Config) GetAnimationFile() string {
	return c.animationFile
}

// GetAnimationDatetime returns whether datetime overlay is enabled
func (c *Config) GetAnimationDatetime() bool {
	return c.animationDatetime
}

// GetDatetimePosition returns the datetime overlay position (top, center, bottom)
func (c *Config) GetDatetimePosition() string {
	return c.datetimePosition
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

// isSafeIdentifier checks if a string contains only safe characters (alphanumeric, hyphens, underscores)
// This prevents shell metacharacters and command injection
func isSafeIdentifier(s string) bool {
	// Allow: letters, numbers, hyphens, underscores
	// Block: shell metacharacters like ; | & $ ( ) ` < > etc.
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, s)
	return matched
}

// isSafePath checks if a file path is absolute and within allowed directories
func isSafePath(path string) bool {
	// Must be absolute path
	if !filepath.IsAbs(path) {
		return false
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(path)

	// Only allow paths under user's home, /usr/share, /usr/local/share
	homeDir := os.Getenv("HOME")
	allowedPrefixes := []string{
		filepath.Join(homeDir, ".local", "share"),
		filepath.Join(homeDir, ".config"),
		"/usr/share",
		"/usr/local/share",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
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

// GetScreensaverCommand returns the command and arguments to launch the screensaver
// Returns (terminal, args, error) where terminal is the executable and args are its arguments
func (c *Config) GetScreensaverCommand() (string, []string, error) {
	terminal := c.GetTerminalLauncher()
	effect := c.GetAnimationEffect()
	theme := c.GetAnimationTheme()
	file := c.GetAnimationFile()

	// Validate effect name (prevent command injection)
	if !isSafeIdentifier(effect) {
		return "", nil, fmt.Errorf("invalid animation effect: %s (contains unsafe characters)", effect)
	}

	// Validate theme name (prevent command injection)
	if !isSafeIdentifier(theme) {
		return "", nil, fmt.Errorf("invalid animation theme: %s (contains unsafe characters)", theme)
	}

	// Build arguments array
	args := c.GetTerminalArgs()
	args = append(args, "--class", "sysc-walls-screensaver")
	args = append(args, "/usr/local/bin/sysc-walls-display", "--effect", effect, "--theme", theme)

	// Add custom file path if specified and valid
	if file != "" {
		if !isSafePath(file) {
			return "", nil, fmt.Errorf("invalid animation file path: %s (must be absolute path in allowed directory)", file)
		}
		args = append(args, "--file", file)
	}

	// Add datetime overlay if enabled and compatible with effect
	datetime := c.GetAnimationDatetime()
	if datetime {
		// Check if effect is text-based (datetime overlay is incompatible with text-based effects)
		if syscGo.IsTextBasedEffect(effect) {
			// Log warning but don't fail - just disable datetime for this launch
			fmt.Fprintf(os.Stderr, "Warning: DateTime overlay disabled - incompatible with text-based effect '%s'\n", effect)
			fmt.Fprintf(os.Stderr, "         DateTime only works with non-text effects like: matrix, fire, rain, aquarium, fireworks, beams\n")
		} else {
			// Effect is compatible, add --datetime flag and position
			args = append(args, "--datetime")
			position := c.GetDatetimePosition()
			args = append(args, "--datetime-position", position)
		}
	}

	args = append(args, "--fullscreen")

	return terminal, args, nil
}

// GetScreensaverCommandString returns the command as a string for logging purposes only
// DO NOT use this for execution - use GetScreensaverCommand() instead
func (c *Config) GetScreensaverCommandString() string {
	terminal, args, err := c.GetScreensaverCommand()
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	parts := append([]string{terminal}, args...)
	return strings.Join(parts, " ")
}
