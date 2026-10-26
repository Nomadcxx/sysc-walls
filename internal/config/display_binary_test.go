package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDisplayBinaryPrefersExecutableDirectory(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(filepath.Dir(executable), "sysc-walls-display")
	writeExecutable(t, sibling)
	t.Cleanup(func() { _ = os.Remove(sibling) })

	home := t.TempDir()
	homeBinary := filepath.Join(home, ".local", "bin", "sysc-walls-display")
	writeExecutable(t, homeBinary)
	pathBinary := filepath.Join(t.TempDir(), "sysc-walls-display")
	writeExecutable(t, pathBinary)
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Dir(pathBinary))

	got, err := FindDisplayBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got != sibling {
		t.Fatalf("findDisplayBinary() = %q, want executable sibling %q", got, sibling)
	}
}

func TestFindDisplayBinarySearchesGreeterHome(t *testing.T) {
	home := t.TempDir()
	want := filepath.Join(home, ".local", "bin", "sysc-walls-display")
	writeExecutable(t, want)
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())

	got, err := FindDisplayBinary()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("findDisplayBinary() = %q, want %q", got, want)
	}
}

func TestFindDisplayBinaryResolvesExecutableSymlink(t *testing.T) {
	installDir := t.TempDir()
	executable := filepath.Join(installDir, "sysc-walls-daemon")
	writeExecutable(t, executable)
	want := filepath.Join(installDir, "sysc-walls-display")
	writeExecutable(t, want)

	linkDir := t.TempDir()
	link := filepath.Join(linkDir, "sysc-walls-daemon")
	if err := os.Symlink(executable, link); err != nil {
		t.Fatal(err)
	}

	pathBinary := filepath.Join(t.TempDir(), "sysc-walls-display")
	writeExecutable(t, pathBinary)
	t.Setenv("PATH", filepath.Dir(pathBinary))

	got, err := findDisplayBinary(link, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("findDisplayBinary() = %q, want resolved sibling %q", got, want)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("#!/bin/sh\nexit 0\n"); err != nil {
		file.Close()
		_ = os.Remove(path)
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		t.Fatal(err)
	}
}
