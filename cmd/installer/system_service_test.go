package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func installFakeSystemctl(t *testing.T, loadState string, showExit, disableExit int) string {
	t.Helper()

	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "systemctl.log")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$SYSTEMCTL_CALLS"
case "$2" in
  show)
    printf '%s\n' "$SYSTEMCTL_LOAD_STATE"
    exit "$SYSTEMCTL_SHOW_EXIT"
    ;;
  disable)
    echo "disabled"
    exit "$SYSTEMCTL_DISABLE_EXIT"
    ;;
  *)
    echo "unexpected systemctl invocation: $*" >&2
    exit 97
    ;;
esac
`
	path := filepath.Join(binDir, "systemctl")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake systemctl: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SYSTEMCTL_CALLS", logPath)
	t.Setenv("SYSTEMCTL_LOAD_STATE", loadState)
	t.Setenv("SYSTEMCTL_SHOW_EXIT", strconv.Itoa(showExit))
	t.Setenv("SYSTEMCTL_DISABLE_EXIT", strconv.Itoa(disableExit))
	return logPath
}

func TestDisableLegacySystemService(t *testing.T) {
	t.Run("missing service is a no-op", func(t *testing.T) {
		logPath := installFakeSystemctl(t, "not-found", 4, 0)
		if err := disableLegacySystemService(nil); err != nil {
			t.Fatalf("disableLegacySystemService() error = %v", err)
		}

		calls, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("read systemctl calls: %v", err)
		}
		if got, want := string(calls), "--system show --property=LoadState --value sysc-walls.service\n"; got != want {
			t.Fatalf("systemctl calls = %q, want %q", got, want)
		}
	})

	t.Run("loaded service is disabled in the system manager", func(t *testing.T) {
		logPath := installFakeSystemctl(t, "loaded", 0, 0)
		if err := disableLegacySystemService(nil); err != nil {
			t.Fatalf("disableLegacySystemService() error = %v", err)
		}

		calls, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("read systemctl calls: %v", err)
		}
		want := "--system show --property=LoadState --value sysc-walls.service\n" +
			"--system disable --now sysc-walls.service\n"
		if got := string(calls); got != want {
			t.Fatalf("systemctl calls = %q, want %q", got, want)
		}
	})

	t.Run("disable failure is returned", func(t *testing.T) {
		installFakeSystemctl(t, "loaded", 0, 1)
		if err := disableLegacySystemService(nil); err == nil || !strings.Contains(err.Error(), "disable --now") {
			t.Fatalf("disableLegacySystemService() error = %v, want disable failure", err)
		}
	})

	t.Run("inspection failure is returned", func(t *testing.T) {
		installFakeSystemctl(t, "", 1, 0)
		if err := disableLegacySystemService(nil); err == nil || !strings.Contains(err.Error(), "inspect") {
			t.Fatalf("disableLegacySystemService() error = %v, want inspection failure", err)
		}
	})
}
