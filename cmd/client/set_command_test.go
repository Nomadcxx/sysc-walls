package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// clientBin is built once for the whole package: handleSetCommand calls
// os.Exit, so the CLI has to run as a subprocess, and rebuilding per test
// dominates the runtime otherwise.
var clientBin string

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("go"); err != nil {
		clientBin = ""
		os.Exit(m.Run())
	}

	dir, err := os.MkdirTemp("", "sysc-walls-client-bin")
	if err != nil {
		panic(err)
	}
	clientBin = filepath.Join(dir, "client")

	build := exec.Command("go", "build", "-o", clientBin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		panic("building client: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// runClient invokes the client binary as a subprocess so the real exit status
// and output can be asserted.
func runClient(t *testing.T, home string, args ...string) (stdout string, code int) {
	t.Helper()

	if clientBin == "" {
		t.Skip("go toolchain not available")
	}

	cmd := exec.Command(clientBin, args...)
	cmd.Env = append(os.Environ(), "HOME="+home)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return out.String(), ee.ExitCode()
		}
		t.Fatalf("running client: %v", err)
	}
	return out.String(), 0
}

func readEffect(t *testing.T, home string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, ".config", "sysc-walls", "daemon.conf"))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "effect") {
			if i := strings.Index(line, "="); i >= 0 {
				return strings.TrimSpace(line[i+1:])
			}
		}
	}
	return ""
}

// TestSetRejectsUnknownEffect covers #52: an invalid effect was accepted
// silently, reported as a success, and left the stored value untouched.
func TestSetRejectsUnknownEffect(t *testing.T) {
	home := t.TempDir()

	// Seed a known-good value.
	if out, code := runClient(t, home, "set", "effect", "matrix"); code != 0 {
		t.Fatalf("seeding effect failed (code %d): %s", code, out)
	}

	out, code := runClient(t, home, "set", "effect", "not-a-real-effect")

	if code == 0 {
		t.Errorf("set with an invalid effect exited 0, want non-zero; output: %s", out)
	}
	if strings.Contains(out, "Set animation effect to:") {
		t.Errorf("set reported success for a rejected value; output: %s", out)
	}
	if got := readEffect(t, home); got != "matrix" {
		t.Errorf("stored effect = %q, want %q — a rejected value must not be written", got, "matrix")
	}
}

// TestSetRejectsUnknownTheme is the same case for themes.
func TestSetRejectsUnknownTheme(t *testing.T) {
	home := t.TempDir()

	if out, code := runClient(t, home, "set", "theme", "rama"); code != 0 {
		t.Fatalf("seeding theme failed (code %d): %s", code, out)
	}

	out, code := runClient(t, home, "set", "theme", "not-a-real-theme")

	if code == 0 {
		t.Errorf("set with an invalid theme exited 0, want non-zero; output: %s", out)
	}
	if strings.Contains(out, "Set animation theme to:") {
		t.Errorf("set reported success for a rejected value; output: %s", out)
	}
}

// TestSetAcceptsValidValues guards the fix from over-correcting: valid values
// must still succeed and still be written.
func TestSetAcceptsValidValues(t *testing.T) {
	home := t.TempDir()

	out, code := runClient(t, home, "set", "effect", "fireworks")
	if code != 0 {
		t.Fatalf("set effect fireworks failed (code %d): %s", code, out)
	}
	if !strings.Contains(out, "Set animation effect to: fireworks") {
		t.Errorf("expected a success message, got: %s", out)
	}
	if got := readEffect(t, home); got != "fireworks" {
		t.Errorf("stored effect = %q, want %q", got, "fireworks")
	}

	out, code = runClient(t, home, "set", "timeout", "7m")
	if code != 0 {
		t.Fatalf("set timeout 7m failed (code %d): %s", code, out)
	}
}

// TestSetFailsWhenConfigCannotBeWritten covers the second half of #52: a
// failed save used to be a warning with exit 0.
func TestSetFailsWhenConfigCannotBeWritten(t *testing.T) {
	home := t.TempDir()

	// A regular file where the config directory needs to be makes MkdirAll fail.
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("preparing home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("blocking .config: %v", err)
	}

	out, code := runClient(t, home, "set", "effect", "matrix")
	if code == 0 {
		t.Errorf("set exited 0 despite being unable to write the config; output: %s", out)
	}
	if strings.Contains(out, "Set animation effect to:") {
		t.Errorf("set reported success although the write failed; output: %s", out)
	}
}
