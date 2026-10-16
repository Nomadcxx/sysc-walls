package config

import (
	"os"
	"path/filepath"
	"testing"
)

func isolateShellTheme(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestAnimationThemeFollowsCurrentShellSelection(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	themeDir := filepath.Join(configDir, "sysc-shell")
	if err := os.MkdirAll(themeDir, 0700); err != nil {
		t.Fatal(err)
	}
	selection := filepath.Join(themeDir, "shell-theme")
	cfg := NewConfig()

	for _, name := range []string{"dracula", "blue"} {
		if err := os.WriteFile(selection, []byte(name+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if got := cfg.GetAnimationTheme(); got != name {
			t.Fatalf("theme = %q after publishing %q, want shell theme", got, name)
		}
	}
}

func TestAnimationThemeUsesConfiguredFallbackForInvalidSelection(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	themeDir := filepath.Join(configDir, "sysc-shell")
	if err := os.MkdirAll(themeDir, 0700); err != nil {
		t.Fatal(err)
	}
	selection := filepath.Join(themeDir, "shell-theme")
	if err := os.WriteFile(selection, []byte("not-a-theme\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := NewConfig()
	if got := cfg.GetAnimationTheme(); got != "rama" {
		t.Fatalf("invalid shell theme = %q, want configured fallback rama", got)
	}
	if err := os.Remove(selection); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/zero", selection); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetAnimationTheme(); got != "rama" {
		t.Fatalf("symlinked shell theme = %q, want configured fallback rama", got)
	}
}

func TestAnimationThemeConfigCanDisableFollowing(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	themeDir := filepath.Join(configDir, "sysc-shell")
	if err := os.MkdirAll(themeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "shell-theme"), []byte("dracula\n"), 0600); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "daemon.conf")
	if err := os.WriteFile(configFile, []byte("[animation]\ntheme = nord\nfollow_shell = false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := NewConfig()
	if err := cfg.LoadFromFile(configFile); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetAnimationTheme(); got != "nord" {
		t.Fatalf("theme with follow_shell=false = %q, want configured nord", got)
	}
}

func TestSetAnimationThemeDisablesFollowing(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	themeDir := filepath.Join(configDir, "sysc-shell")
	if err := os.MkdirAll(themeDir, 0700); err != nil {
		t.Fatal(err)
	}
	selection := filepath.Join(themeDir, "shell-theme")
	if err := os.WriteFile(selection, []byte("dracula\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := NewConfig()
	if err := cfg.SetAnimationTheme("nord"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selection, []byte("blue\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetAnimationTheme(); got != "nord" {
		t.Fatalf("theme after explicit set = %q, want nord", got)
	}
}
