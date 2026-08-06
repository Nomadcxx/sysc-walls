package config

import (
	"os"
	"path/filepath"
	"strings"
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

	if cfg.GetAnimationEffect() != "matrix-art" {
		t.Errorf("Default effect = %s, want matrix-art", cfg.GetAnimationEffect())
	}

	if cfg.GetAnimationTheme() != "rama" {
		t.Errorf("Default theme = %s, want rama", cfg.GetAnimationTheme())
	}

	if cfg.GetAnimationDatetime() {
		t.Error("Default datetime should be false")
	}

	if cfg.GetDatetimePosition() != "bottom" {
		t.Errorf("Default datetime position = %s, want bottom", cfg.GetDatetimePosition())
	}

	if cfg.GetDatetimeInterval() != time.Second {
		t.Errorf("Default datetime interval = %v, want %v", cfg.GetDatetimeInterval(), time.Second)
	}

	if cfg.ShouldCycleAnimations() != false {
		t.Error("Default cycle animations should be false")
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
datetime = true
cycle = false

[datetime]
position = top
interval = 2s

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
	if !cfg2.GetAnimationDatetime() {
		t.Error("Loaded datetime = false, want true")
	}
	if cfg2.GetDatetimePosition() != "top" {
		t.Errorf("Loaded datetime position = %s, want top", cfg2.GetDatetimePosition())
	}
	if cfg2.GetDatetimeInterval() != 2*time.Second {
		t.Errorf("Loaded datetime interval = %v, want 2s", cfg2.GetDatetimeInterval())
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
	cfg.parseConfigLine("animation.datetime", "true")
	cfg.parseConfigLine("datetime.position", "center")
	cfg.parseConfigLine("datetime.interval", "3s")

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
	if !cfg2.GetAnimationDatetime() {
		t.Error("Saved/loaded datetime = false, want true")
	}
	if cfg2.GetDatetimePosition() != "center" {
		t.Errorf("Saved/loaded datetime position = %s, want center", cfg2.GetDatetimePosition())
	}
	if cfg2.GetDatetimeInterval() != 3*time.Second {
		t.Errorf("Saved/loaded datetime interval = %v, want 3s", cfg2.GetDatetimeInterval())
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
	cfg.parseConfigLine("animation.datetime", "true")
	cfg.parseConfigLine("datetime.position", "top")
	cfg.parseConfigLine("datetime.interval", "2s")

	// Use a fake display binary in PATH so this test doesn't depend on host install.
	tmpDir := t.TempDir()
	fakeDisplay := filepath.Join(tmpDir, "sysc-walls-display")
	if err := os.WriteFile(fakeDisplay, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake display binary: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)

	cmd := cfg.GetScreensaverCommandString()
	if cmd == "" {
		t.Error("GetScreensaverCommandString() returned empty string")
	}

	// Verify it contains key components
	if !strings.Contains(cmd, "kitty") {
		t.Errorf("Command missing 'kitty': %s", cmd)
	}
	if !strings.Contains(cmd, "sysc-walls-display") {
		t.Errorf("Command missing 'sysc-walls-display': %s", cmd)
	}
	if !strings.Contains(cmd, "--effect") {
		t.Errorf("Command missing '--effect': %s", cmd)
	}
	if !strings.Contains(cmd, "matrix") {
		t.Errorf("Command missing 'matrix': %s", cmd)
	}
	if !strings.Contains(cmd, "--theme") {
		t.Errorf("Command missing '--theme': %s", cmd)
	}
	if !strings.Contains(cmd, "nord") {
		t.Errorf("Command missing 'nord': %s", cmd)
	}
	if !strings.Contains(cmd, "--datetime") {
		t.Errorf("Command missing '--datetime': %s", cmd)
	}
	if !strings.Contains(cmd, "--datetime-position top") {
		t.Errorf("Command missing '--datetime-position top': %s", cmd)
	}
	if !strings.Contains(cmd, "--datetime-interval 2s") {
		t.Errorf("Command missing '--datetime-interval 2s': %s", cmd)
	}
}

func TestGetScreensaverCommandFireTextDisablesDatetime(t *testing.T) {
	cfg := NewConfig()
	cfg.SetAnimationEffect("fire-text")
	cfg.SetAnimationTheme("nord")
	cfg.parseConfigLine("animation.datetime", "true")
	cfg.parseConfigLine("datetime.position", "top")
	cfg.parseConfigLine("datetime.interval", "2s")

	// Use a fake display binary in PATH so this test doesn't depend on host install.
	tmpDir := t.TempDir()
	fakeDisplay := filepath.Join(tmpDir, "sysc-walls-display")
	if err := os.WriteFile(fakeDisplay, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake display binary: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)

	cmd := cfg.GetScreensaverCommandString()
	if cmd == "" {
		t.Error("GetScreensaverCommandString() returned empty string")
	}

	if strings.Contains(cmd, "--datetime") {
		t.Errorf("Command should not include '--datetime' for fire-text: %s", cmd)
	}
	if strings.Contains(cmd, "--datetime-position") {
		t.Errorf("Command should not include '--datetime-position' for fire-text: %s", cmd)
	}
	if strings.Contains(cmd, "--datetime-interval") {
		t.Errorf("Command should not include '--datetime-interval' for fire-text: %s", cmd)
	}
}

func TestGetScreensaverCommandBeamTextAllowsDatetime(t *testing.T) {
	cfg := NewConfig()
	cfg.SetAnimationEffect("beam-text")
	cfg.SetAnimationTheme("nord")
	cfg.parseConfigLine("animation.datetime", "true")
	cfg.parseConfigLine("datetime.position", "center")
	cfg.parseConfigLine("datetime.interval", "2s")

	tmpDir := t.TempDir()
	fakeDisplay := filepath.Join(tmpDir, "sysc-walls-display")
	if err := os.WriteFile(fakeDisplay, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake display binary: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)

	cmd := cfg.GetScreensaverCommandString()
	if cmd == "" {
		t.Error("GetScreensaverCommandString() returned empty string")
	}
	if !strings.Contains(cmd, "--datetime") {
		t.Errorf("Command should include '--datetime' for beam-text: %s", cmd)
	}
	if !strings.Contains(cmd, "--datetime-position center") {
		t.Errorf("Command should include '--datetime-position center' for beam-text: %s", cmd)
	}
	if !strings.Contains(cmd, "--datetime-interval 2s") {
		t.Errorf("Command should include '--datetime-interval 2s' for beam-text: %s", cmd)
	}
}

// TestCycleInterval tests cycle interval configuration
func TestCycleInterval(t *testing.T) {
	cfg := NewConfig()

	// Check default value (5 minutes)
	if cfg.GetCycleInterval() != 5*time.Minute {
		t.Errorf("Default cycle interval = %v, want %v", cfg.GetCycleInterval(), 5*time.Minute)
	}

	// Test SetCycleInterval with valid values
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30s", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"invalid", 0, true},
		{"0s", 0, true},
		{"-5m", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := cfg.SetCycleInterval(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SetCycleInterval(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("SetCycleInterval(%q) unexpected error: %v", tt.input, err)
				}
				if cfg.GetCycleInterval() != tt.expected {
					t.Errorf("GetCycleInterval() = %v, want %v", cfg.GetCycleInterval(), tt.expected)
				}
			}
		})
	}
}

func TestInvalidDatetimeIntervalFallsBackToDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	configContent := `[animation]
effect = matrix-art
theme = rama
datetime = true

[datetime]
position = center
interval = 0s
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg := NewConfig()
	if err := cfg.LoadFromFile(configPath); err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if cfg.GetDatetimeInterval() != time.Second {
		t.Errorf("datetime interval = %v, want default %v", cfg.GetDatetimeInterval(), time.Second)
	}
	if cfg.GetDatetimePosition() != "center" {
		t.Errorf("datetime position = %s, want center", cfg.GetDatetimePosition())
	}
	if !cfg.GetAnimationDatetime() {
		t.Error("animation.datetime should remain true")
	}
}

// TestCycleIntervalLoading tests loading cycle_interval from config file
func TestCycleIntervalLoading(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	// Create config with cycle_interval
	configContent := `[idle]
timeout = 5m
min_duration = 30s

[animation]
effect = matrix
theme = dracula
cycle = true
cycle_interval = 10m
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg := NewConfig()
	err = cfg.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	if cfg.GetCycleInterval() != 10*time.Minute {
		t.Errorf("Loaded cycle_interval = %v, want %v", cfg.GetCycleInterval(), 10*time.Minute)
	}
	if !cfg.ShouldCycleAnimations() {
		t.Error("Loaded cycle = false, want true")
	}
}

// TestSetIdleTimeout_RejectsNonPositive verifies that a zero or missing idle
// timeout is refused.
//
// A zero timeout arms the daemon's fallback timer with no delay, so it launches
// the screensaver, resets, and fires again as fast as the CPU allows.
func TestSetIdleTimeout_RejectsNonPositive(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "zero seconds", value: "0s", wantErr: true},
		{name: "zero minutes", value: "0m", wantErr: true},
		{name: "bare zero", value: "0", wantErr: true},
		{name: "one second", value: "1s"},
		{name: "fifteen minutes", value: "15m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			before := cfg.GetIdleTimeout()

			err := cfg.SetIdleTimeout(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("SetIdleTimeout(%q) error = nil, want an error", tt.value)
				}
				if got := cfg.GetIdleTimeout(); got != before {
					t.Errorf("GetIdleTimeout() = %v after a rejected value, want %v unchanged", got, before)
				}
				return
			}

			if err != nil {
				t.Fatalf("SetIdleTimeout(%q) error = %v", tt.value, err)
			}
		})
	}
}

// TestLoadFromFile_IgnoresNonPositiveIdleTimeout verifies that a bad timeout in
// the config file leaves the default in place instead of arming a zero timer.
func TestLoadFromFile_IgnoresNonPositiveIdleTimeout(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "daemon.conf")
	if err := os.WriteFile(configPath, []byte("idle.timeout = 0s\n"), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg := NewConfig()
	want := cfg.GetIdleTimeout()

	if err := cfg.LoadFromFile(configPath); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if got := cfg.GetIdleTimeout(); got != want {
		t.Errorf("GetIdleTimeout() = %v, want the default %v", got, want)
	}
}
