package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestResolveUserHomeUsesHOMEWithoutSudo is the non-sudo path: the invoking
// user is the target, so $HOME is authoritative.
func TestResolveUserHomeUsesHOMEWithoutSudo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")

	gotHome, uid, gid, err := resolveUserHome()
	if err != nil {
		t.Fatalf("resolveUserHome() error = %v", err)
	}
	if gotHome != home {
		t.Errorf("resolveUserHome() home = %q, want %q", gotHome, home)
	}
	if uid != os.Getuid() || gid != os.Getgid() {
		t.Errorf("resolveUserHome() uid/gid = %d/%d, want %d/%d", uid, gid, os.Getuid(), os.Getgid())
	}
}

// TestResolveUserHomeErrorsWithoutHOME: guessing a home when none is known is
// how the unit file ended up somewhere other than the config.
func TestResolveUserHomeErrorsWithoutHOME(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("SUDO_USER", "")

	if _, _, _, err := resolveUserHome(); err == nil {
		t.Error("resolveUserHome() error = nil with HOME unset, want an error rather than a guessed path")
	}
}

// TestResolveUserHomeUsesPasswdNotSlashHome is the core of #44: under sudo the
// home comes from the passwd database, not "/home/"+name. A test user is
// created with a home outside /home to make the two disagree.
func TestResolveUserHomeUsesPasswdNotSlashHome(t *testing.T) {
	if _, err := exec.LookPath("useradd"); err != nil {
		t.Skip("useradd not available")
	}
	if os.Geteuid() != 0 {
		t.Skip("needs root to create a test user")
	}

	const name = "swtesthome"
	home := "/opt/svc/" + name

	out, err := exec.Command("useradd", "-M", "-d", home, "-s", "/bin/false", name).CombinedOutput()
	if err != nil {
		t.Skipf("useradd failed: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("userdel", "-r", name).Run()
	})

	t.Setenv("SUDO_USER", name)

	gotHome, uid, _, err := resolveUserHome()
	if err != nil {
		t.Fatalf("resolveUserHome() error = %v", err)
	}

	if gotHome == "/home/"+name {
		t.Errorf("resolveUserHome() home = %q, want the passwd value %q", gotHome, home)
	}
	if gotHome != home {
		t.Errorf("resolveUserHome() home = %q, want %q", gotHome, home)
	}

	// The id must match the created account, so the unit gets chowned to a
	// real user rather than to root.
	if want := uidOf(t, name); uid != want {
		t.Errorf("resolveUserHome() uid = %d, want %d", uid, want)
	}
}

func uidOf(t *testing.T, user string) int {
	t.Helper()
	out, err := exec.Command("id", "-u", user).Output()
	if err != nil {
		t.Fatalf("id -u %s: %v", user, err)
	}
	uid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parsing uid: %v", err)
	}
	return uid
}

// TestBackupNameDoesNotClobber reproduces the data loss from #45: a fixed
// backup name means a second override overwrites the only copy of the user's
// settings.
func TestBackupNameDoesNotClobber(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "sysc-walls")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(configDir, "daemon.conf")

	// First run: user's real settings get backed up.
	if err := os.WriteFile(configPath, []byte("effect = matrix\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	first := nextFreeBackupPath(configPath)
	if err := os.WriteFile(first, []byte("effect = matrix\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	// Second run: the config now holds defaults, and must not land on top of
	// the first backup.
	if err := os.WriteFile(configPath, []byte("effect = fire\n"), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}
	second := nextFreeBackupPath(configPath)

	if second == first {
		t.Fatalf("second backup reused %s, overwriting the only copy of the original config", first)
	}

	got, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("reading the original backup: %v", err)
	}
	if !strings.Contains(string(got), "matrix") {
		t.Errorf("the original backup was overwritten: %q", got)
	}
}
