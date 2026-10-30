package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateConfigPreservesExistingASCIIArt(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	wantDefault, err := os.ReadFile(filepath.Join(root, "assets", "ascii", "SYSC2.txt"))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name           string
		overrideConfig bool
	}{
		{name: "keep config"},
		{name: "override config", overrideConfig: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("SUDO_USER", "")

			configDir := filepath.Join(home, ".config", "sysc-walls")
			asciiDir := filepath.Join(configDir, "ascii")
			if err := os.MkdirAll(asciiDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(configDir, "daemon.conf"), []byte("[custom]\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			customArt := []byte("MY CUSTOM ART\n")
			artPath := filepath.Join(asciiDir, "SYSC.txt")
			if err := os.WriteFile(artPath, customArt, 0o644); err != nil {
				t.Fatal(err)
			}

			if err := updateConfig(&model{overrideConfig: tc.overrideConfig}); err != nil {
				t.Fatalf("updateConfig() error = %v", err)
			}

			gotArt, err := os.ReadFile(artPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotArt) != string(customArt) {
				t.Errorf("custom art = %q, want %q", gotArt, customArt)
			}

			gotDefault, err := os.ReadFile(filepath.Join(asciiDir, "SYSC2.txt"))
			if err != nil {
				t.Fatalf("missing bundled art was not installed: %v", err)
			}
			if string(gotDefault) != string(wantDefault) {
				t.Errorf("installed default art does not match bundled art")
			}
		})
	}
}

func TestUpdateConfigReportsASCIIArtCopyErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to a read-only directory")
	}

	workDir := t.TempDir()
	assetDir := filepath.Join(workDir, "assets", "ascii")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "SYSC.txt"), []byte("default art\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	destDir := filepath.Join(home, ".config", "sysc-walls", "ascii")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(destDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	t.Chdir(workDir)

	if err := updateConfig(&model{}); err == nil {
		t.Fatal("updateConfig() error = nil, want ASCII art copy error")
	}
}

func TestUpdateConfigDoesNotFollowASCIIArtSymlinks(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	for _, dangling := range []bool{false, true} {
		name := "existing target"
		if dangling {
			name = "dangling target"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("SUDO_USER", "")

			asciiDir := filepath.Join(home, ".config", "sysc-walls", "ascii")
			if err := os.MkdirAll(asciiDir, 0o755); err != nil {
				t.Fatal(err)
			}
			targetPath := filepath.Join(home, "custom-art.txt")
			wantTarget := []byte("CUSTOM TARGET ART\n")
			if !dangling {
				if err := os.WriteFile(targetPath, wantTarget, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			artPath := filepath.Join(asciiDir, "SYSC.txt")
			if err := os.Symlink(targetPath, artPath); err != nil {
				t.Fatal(err)
			}

			if err := updateConfig(&model{}); err != nil {
				t.Fatalf("updateConfig() error = %v", err)
			}
			info, err := os.Lstat(artPath)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("ASCII art path mode = %v, want symlink", info.Mode())
			}

			gotTarget, err := os.ReadFile(targetPath)
			if dangling {
				if !os.IsNotExist(err) {
					t.Fatalf("dangling symlink target read error = %v, want not-exist", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(gotTarget) != string(wantTarget) {
				t.Errorf("symlink target = %q, want %q", gotTarget, wantTarget)
			}
		})
	}
}
