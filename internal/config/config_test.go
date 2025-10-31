package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewConfig verifies default configuration values
func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Check default values
	if cfg.GetIdleTimeout() != 300*time.Second {
		t.Errorf("Default idle timeout = %v, want %v", cfg.GetIdleTimeout(), 300*time.Second)
	}

	if cfg.GetMinDuration() != 30*time.Second {
		t.Errorf("Default min duration = %v, want %v", cfg.GetMinDuration(), 30*time.Second)
	}

	if cfg.IsDebug() != false {
		t.Error("Default debug should be false")
	}

	if cfg.GetAnimationEffect() != "matrix" {
		t.Errorf("Default effect = %s, want matrix", cfg.GetAnimationEffect())
	}

	if cfg.GetAnimationTheme() != "nord" {
		t.Errorf("Default theme = %s, want nord", cfg.GetAnimationTheme())
	}

	if cfg.ShouldCycleAnimations() != true {
		t.Error("Default cycle animations should be true")
	}
}

// TestParseDuration tests duration string parsing
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		// Valid inputs
		{"5s", 5 * time.Second, false},
		{"30s", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"10m", 10 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"2h", 2 * time.Hour, false},
		{"300", 300 * time.Second, false}, // Bare number = seconds
		{"60", 60 * time.Second, false},

		// Invalid inputs
		{"invalid", 0, true},
		{"5x", 0, true},
		{"", 0, true},
		{"-5s", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("parseDuration(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseDuration(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{60 * time.Second, "1m"},
		{5 * time.Minute, "5m"},
		{3600 * time.Second, "1h"},
		{2 * time.Hour, "2h"},
		{90 * time.Second, "1m"}, // 90s rounds to 1m
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.input)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConfigGettersSetters tests all getter/setter methods
func TestConfigGettersSetters(t *testing.T) {
	cfg := NewConfig()

	// Test SetIdleTimeout
	err := cfg.SetIdleTimeout("10m")
	if err != nil {
		t.Errorf("SetIdleTimeout(10m) error: %v", err)
	}
	if cfg.GetIdleTimeout() != 10*time.Minute {
		t.Errorf("GetIdleTimeout() = %v, want 10m", cfg.GetIdleTimeout())
	}

	// Test invalid timeout
	err = cfg.SetIdleTimeout("invalid")
	if err == nil {
		t.Error("SetIdleTimeout(invalid) should return error")
	}

	// Test SetDebug
	cfg.SetDebug(true)
	if !cfg.IsDebug() {
		t.Error("SetDebug(true) failed")
	}

	// Test SetAnimationEffect
	cfg.SetAnimationEffect("fire")
	if cfg.GetAnimationEffect() != "fire" {
		t.Errorf("SetAnimationEffect(fire) failed, got %s", cfg.GetAnimationEffect())
	}

	// Test SetAnimationTheme
	cfg.SetAnimationTheme("dracula")
	if cfg.GetAnimationTheme() != "dracula" {
		t.Errorf("SetAnimationTheme(dracula) failed, got %s", cfg.GetAnimationTheme())
	}

	// Test terminal settings
	cfg.SetTerminalKitty(false)
	if cfg.IsTerminalKitty() {
		t.Error("SetTerminalKitty(false) failed")
	}

	cfg.SetTerminalFullscreen(false)
	if cfg.IsTerminalFullscreen() {
		t.Error("SetTerminalFullscreen(false) failed")
	}
}

// TestLoadFromFile tests loading configuration from file
func TestLoadFromFile(t *testing.T) {
	// Create temporary directory for test configs
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	// Test 1: Non-existent file creates default
	cfg := NewConfig()
	err := cfg.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() should create default config, got error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("LoadFromFile() didn't create config file")
	}

	// Test 2: Load valid config file
	validConfig := `# Test config
[idle]
timeout = 10m
min_duration = 60s

[daemon]
debug = true

[animation]
effect = fire
theme = gruvbox
cycle = false

[terminal]
kitty = false
fullscreen = false
`
	err = os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg2 := NewConfig()
	err = cfg2.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	// Verify loaded values
	if cfg2.GetIdleTimeout() != 10*time.Minute {
		t.Errorf("Loaded timeout = %v, want 10m", cfg2.GetIdleTimeout())
	}
	if cfg2.GetMinDuration() != 60*time.Second {
		t.Errorf("Loaded min_duration = %v, want 60s", cfg2.GetMinDuration())
	}
	if !cfg2.IsDebug() {
		t.Error("Loaded debug = false, want true")
	}
	if cfg2.GetAnimationEffect() != "fire" {
		t.Errorf("Loaded effect = %s, want fire", cfg2.GetAnimationEffect())
	}
	if cfg2.GetAnimationTheme() != "gruvbox" {
		t.Errorf("Loaded theme = %s, want gruvbox", cfg2.GetAnimationTheme())
	}
	if cfg2.ShouldCycleAnimations() {
		t.Error("Loaded cycle = true, want false")
	}
	if cfg2.IsTerminalKitty() {
		t.Error("Loaded kitty = true, want false")
	}
	if cfg2.IsTerminalFullscreen() {
		t.Error("Loaded fullscreen = true, want false")
	}
}

// TestSaveToFile tests saving configuration to file
func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save-test.conf")

	cfg := NewConfig()
	cfg.SetIdleTimeout("15m")
	cfg.SetDebug(true)
	cfg.SetAnimationEffect("rain")
	cfg.SetAnimationTheme("tokyo-night")

	err := cfg.SaveToFile(configPath)
	if err != nil {
		t.Fatalf("SaveToFile() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveToFile() didn't create file")
	}

	// Load and verify
	cfg2 := NewConfig()
	err = cfg2.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() after save failed: %v", err)
	}

	if cfg2.GetIdleTimeout() != 15*time.Minute {
		t.Errorf("Saved/loaded timeout = %v, want 15m", cfg2.GetIdleTimeout())
	}
	if !cfg2.IsDebug() {
		t.Error("Saved/loaded debug = false, want true")
	}
	if cfg2.GetAnimationEffect() != "rain" {
		t.Errorf("Saved/loaded effect = %s, want rain", cfg2.GetAnimationEffect())
	}
	if cfg2.GetAnimationTheme() != "tokyo-night" {
		t.Errorf("Saved/loaded theme = %s, want tokyo-night", cfg2.GetAnimationTheme())
	}
}

// TestGetTerminalLauncher tests terminal launcher command generation
func TestGetTerminalLauncher(t *testing.T) {
	cfg := NewConfig()

	// Default should be kitty
	if cfg.GetTerminalLauncher() != "kitty" {
		t.Errorf("Default terminal launcher = %s, want kitty", cfg.GetTerminalLauncher())
	}

	// Switch to xterm
	cfg.SetTerminalKitty(false)
	if cfg.GetTerminalLauncher() != "xterm" {
		t.Errorf("Terminal launcher after SetTerminalKitty(false) = %s, want xterm", cfg.GetTerminalLauncher())
	}
}

// TestGetTerminalArgs tests terminal argument generation
func TestGetTerminalArgs(t *testing.T) {
	cfg := NewConfig()

	// Default should include fullscreen
	args := cfg.GetTerminalArgs()
	if len(args) == 0 {
		t.Error("GetTerminalArgs() returned empty slice")
	}
	found := false
	for _, arg := range args {
		if arg == "--start-as=fullscreen" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GetTerminalArgs() missing --start-as=fullscreen")
	}

	// Disable fullscreen
	cfg.SetTerminalFullscreen(false)
	args = cfg.GetTerminalArgs()
	if len(args) != 0 {
		t.Errorf("GetTerminalArgs() with fullscreen=false should be empty, got %v", args)
	}
}

// TestGetScreensaverCommand tests screensaver command generation
func TestGetScreensaverCommand(t *testing.T) {
	cfg := NewConfig()
	cfg.SetAnimationEffect("matrix")
	cfg.SetAnimationTheme("nord")

	cmd := cfg.GetScreensaverCommand()
	if cmd == "" {
		t.Error("GetScreensaverCommand() returned empty string")
	}

	// Verify it contains key components
	if !contains(cmd, "kitty") {
		t.Errorf("Command missing 'kitty': %s", cmd)
	}
	if !contains(cmd, "sysc-walls-display") {
		t.Errorf("Command missing 'sysc-walls-display': %s", cmd)
	}
	if !contains(cmd, "--effect") {
		t.Errorf("Command missing '--effect': %s", cmd)
	}
	if !contains(cmd, "matrix") {
		t.Errorf("Command missing 'matrix': %s", cmd)
	}
	if !contains(cmd, "--theme") {
		t.Errorf("Command missing '--theme': %s", cmd)
	}
	if !contains(cmd, "nord") {
		t.Errorf("Command missing 'nord': %s", cmd)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
