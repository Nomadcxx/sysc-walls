package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTextContent_PreservesLeadingIndentation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "skull.txt")

	content := "   /\\\n  /__\\\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got := loadTextContent(path, false)
	want := "   /\\\n  /__\\"

	if got != want {
		t.Fatalf("loadTextContent() mismatch\nwant:\n%q\ngot:\n%q", want, got)
	}
}

func TestLoadTextContent_NormalizesCRLF(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "skull-crlf.txt")

	content := "A\r\nB\r\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got := loadTextContent(path, false)
	want := "A\nB"

	if got != want {
		t.Fatalf("loadTextContent() mismatch\nwant:\n%q\ngot:\n%q", want, got)
	}
}
