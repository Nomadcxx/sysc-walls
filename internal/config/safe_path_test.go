package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsSafePathAllowedPaths pins the paths that must remain allowed: the
// check exists to stop a path escaping the allowed directories, it must not
// reject legitimate files inside them.
func TestIsSafePathAllowedPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	allowed := []string{
		"/usr/share",
		"/usr/share/",
		"/usr/share/sysc-walls/art.txt",
		"/usr/share/./art.txt",
		"/usr/share/x/../art.txt",
		"/usr/local/share",
		"/usr/local/share/art.txt",
		filepath.Join(home, ".config"),
		filepath.Join(home, ".config", "sysc-walls", "daemon.conf"),
		filepath.Join(home, ".local", "share", "sysc-walls", "art.txt"),
	}

	for _, path := range allowed {
		if !isSafePath(path) {
			t.Errorf("isSafePath(%q) = false, want true", path)
		}
	}
}

// TestIsSafePathBoundaryEscapes pins the rejection of paths that only share a
// textual prefix with an allowed directory but are not inside it (#36).
func TestIsSafePathBoundaryEscapes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	blocked := []string{
		"/usr/shareMORE/art.txt",
		"/usr/share2/art.txt",
		"/usr/local/shareX/art.txt",
		"/usr/local/share2/art.txt",
		filepath.Join(home, ".configuring", "evil.txt"),
		filepath.Join(home, ".config.bak", "evil.txt"),
		filepath.Join(home, ".configX", "evil.txt"),
		filepath.Join(home, ".local", "shareX", "evil.txt"),
		filepath.Join(home, ".local", "share2", "evil.txt"),
		filepath.Join(home, ".local", "share.bak", "evil.txt"),
	}

	for _, path := range blocked {
		if isSafePath(path) {
			t.Errorf("isSafePath(%q) = true, want false", path)
		}
	}
}

// TestIsSafePathCleaningAndRejects verifies traversal is resolved before the
// boundary check and that relative and traversal inputs stay rejected.
func TestIsSafePathCleaningAndRejects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	blocked := []string{
		"",
		"relative/path.txt",
		"./art.txt",
		"/usr/share/../../etc/passwd",
		"/usr/share/../lib/art.txt",
		filepath.Join(home, ".config", "..", "..", "outside.txt"),
		filepath.Join(home, ".config", "..", ".configX", "evil.txt"),
	}

	for _, path := range blocked {
		if isSafePath(path) {
			t.Errorf("isSafePath(%q) = true, want false", path)
		}
	}
}

// TestIsSafePathWithoutHome checks behaviour when HOME is unset: system paths
// stay allowed and home-based prefixes are simply not offered.
func TestIsSafePathWithoutHome(t *testing.T) {
	t.Setenv("HOME", "")

	if !isSafePath("/usr/share/art.txt") {
		t.Error("isSafePath(/usr/share/art.txt) = false, want true")
	}
	if isSafePath("/home/someone/.config/art.txt") {
		t.Error("isSafePath(/home/someone/.config/art.txt) = true, want false")
	}
}

// TestGetScreensaverCommandFilePathValidation covers the call site: a config
// file path that escapes the allowed directories is rejected, while a
// legitimate path still makes it into the command as --file.
func TestGetScreensaverCommandFilePathValidation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Fake display binary so findDisplayBinary() succeeds and the test
	// exercises file path validation, not binary discovery.
	tmpDir := t.TempDir()
	fakeDisplay := filepath.Join(tmpDir, "sysc-walls-display")
	if err := os.WriteFile(fakeDisplay, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write fake display binary: %v", err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	writeConfig := func(t *testing.T, filePath string) string {
		t.Helper()
		cfgPath := filepath.Join(t.TempDir(), "daemon.conf")
		content := "[animation]\nfile = " + filePath + "\n"
		if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}
		return cfgPath
	}

	t.Run("boundary escape is rejected", func(t *testing.T) {
		escape := filepath.Join(home, ".configuring", "evil.txt")
		cfg := NewConfig()
		if err := cfg.LoadFromFile(writeConfig(t, escape)); err != nil {
			t.Fatalf("LoadFromFile: %v", err)
		}
		if got := cfg.GetAnimationFile(); got != escape {
			t.Fatalf("precondition failed: animation.file = %q, want %q", got, escape)
		}
		_, _, err := cfg.GetScreensaverCommand()
		if err == nil || !strings.Contains(err.Error(), "invalid animation file path") {
			t.Fatalf("GetScreensaverCommand() error = %v, want invalid animation file path error", err)
		}
	})

	t.Run("legitimate path is accepted", func(t *testing.T) {
		good := filepath.Join(home, ".config", "sysc-walls", "art.txt")
		cfg := NewConfig()
		if err := cfg.LoadFromFile(writeConfig(t, good)); err != nil {
			t.Fatalf("LoadFromFile: %v", err)
		}
		terminal, args, err := cfg.GetScreensaverCommand()
		if err != nil {
			t.Fatalf("GetScreensaverCommand() error = %v", err)
		}
		if terminal == "" {
			t.Fatal("GetScreensaverCommand() returned empty terminal")
		}
		found := false
		for i, a := range args {
			if a == "--file" && i+1 < len(args) && args[i+1] == good {
				found = true
			}
		}
		if !found {
			t.Fatalf("args %v do not contain --file %s", args, good)
		}
	})
}
