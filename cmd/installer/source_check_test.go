package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cmd", "daemon"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cmd", "daemon", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o777); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The root installer builds the tree it is started in; a tree another
// account could change must be refused (audit sysc-2368).
func TestCheckSourceTree(t *testing.T) {
	uid, gid := os.Getuid(), os.Getgid()
	if uid == 0 {
		t.Skip("ownership cases need a non-root test user")
	}

	if err := checkSourceTree(writeTree(t), uid, gid); err != nil {
		t.Fatalf("own clean checkout refused: %v", err)
	}

	for _, tc := range []struct {
		name  string
		setup func(dir string) error
		uid   int
		gid   int
		want  string
	}{
		{"world-writable file", func(dir string) error {
			return os.Chmod(filepath.Join(dir, "cmd", "daemon", "main.go"), 0o666)
		}, uid, gid, "every account"},
		{"world-writable directory", func(dir string) error {
			return os.Chmod(filepath.Join(dir, "cmd"), 0o777)
		}, uid, gid, "every account"},
		{"group-writable by another group", func(dir string) error {
			return os.Chmod(filepath.Join(dir, "cmd", "daemon"), 0o775)
		}, uid, gid + 1, "group"},
		{"owned by another account", func(string) error { return nil }, uid + 1, gid, "another account"},
	} {
		dir := writeTree(t)
		if err := tc.setup(dir); err != nil {
			t.Fatal(err)
		}
		err := checkSourceTree(dir, tc.uid, tc.gid)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want one mentioning %q", tc.name, err, tc.want)
		}
	}

	// The user's own group may write, as with a private group and umask 002.
	dir := writeTree(t)
	if err := os.Chmod(filepath.Join(dir, "cmd", "daemon"), 0o775); err != nil {
		t.Fatal(err)
	}
	if err := checkSourceTree(dir, uid, gid); err != nil {
		t.Fatalf("own-group-writable checkout refused: %v", err)
	}
}
