package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func useInstallerRepoRoot(t *testing.T) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
}

func TestUpdateConfigRejectsSymlinkedConfigDirectory(t *testing.T) {
	useInstallerRepoRoot(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")

	if err := os.MkdirAll(filepath.Join(home, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(home, ".config", "sysc-walls")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	err := updateConfig(&model{})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("updateConfig() error = %v, want symlink refusal", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "daemon.conf")); !os.IsNotExist(err) {
		t.Errorf("config was written through the symlink: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "ascii")); !os.IsNotExist(err) {
		t.Errorf("ASCII art directory was created through the symlink: %v", err)
	}
}

func TestUpdateConfigRejectsSymlinkedConfigFileOnOverride(t *testing.T) {
	useInstallerRepoRoot(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")

	configDir := filepath.Join(home, ".config", "sysc-walls")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "victim.conf")
	original := []byte("[outside]\nkeep = true\n")
	if err := os.WriteFile(outside, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(configDir, "daemon.conf")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	err := updateConfig(&model{overrideConfig: true})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("updateConfig() error = %v, want symlink refusal", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("file outside the config tree was replaced: %q", got)
	}
}

func TestUpdateConfigRejectsSymlinkedASCIIArtDirectory(t *testing.T) {
	useInstallerRepoRoot(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")

	configDir := filepath.Join(home, ".config", "sysc-walls")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(configDir, "ascii")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	err := updateConfig(&model{})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("updateConfig() error = %v, want symlink refusal", err)
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("ASCII art was installed through the symlink: %v", entries)
	}
}
