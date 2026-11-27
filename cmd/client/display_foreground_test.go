package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func writeDisplayScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "display.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestDisplayPreviewStopsWithSIGTERM is the regression test for #96: the
// timeout path must let the display binary restore the terminal instead of
// killing it outright.
func TestDisplayPreviewStopsWithSIGTERM(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "restored")
	script := writeDisplayScript(t, fmt.Sprintf(
		"trap 'printf \"\\033[?25h\" > %s; exit 0' TERM\nwhile :; do sleep 0.05; done\n",
		strconv.Quote(marker),
	))

	originalGrace := displayTerminateGrace
	displayTerminateGrace = 2 * time.Second
	t.Cleanup(func() { displayTerminateGrace = originalGrace })

	start := time.Now()
	runDisplayForeground(script, nil, 100*time.Millisecond)
	elapsed := time.Since(start)

	restored, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("display did not run its SIGTERM handler: %v", err)
	}
	if string(restored) != "\033[?25h" {
		t.Fatalf("display restore sequence = %q, want cursor show", restored)
	}
	if elapsed >= displayTerminateGrace {
		t.Fatalf("preview took %v; the display was not stopped gracefully", elapsed)
	}
}

// TestDisplayPreviewKillsAStuckProcess verifies the bounded fallback still
// exists for a display process that ignores SIGTERM.
func TestDisplayPreviewKillsAStuckProcess(t *testing.T) {
	script := writeDisplayScript(t, "trap '' TERM\nwhile :; do sleep 0.05; done\n")

	originalGrace := displayTerminateGrace
	displayTerminateGrace = 100 * time.Millisecond
	t.Cleanup(func() { displayTerminateGrace = originalGrace })

	done := make(chan struct{})
	go func() {
		defer close(done)
		runDisplayForeground(script, nil, 50*time.Millisecond)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("stuck display process was not killed after SIGTERM was ignored")
	}
}
