package clock

import (
	"strings"
	"testing"
	"time"
)

func TestRenderClock_OutputHeight(t *testing.T) {
	lines := RenderClock("12:00:00 PM")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestRenderClock_ConsistentLineWidths(t *testing.T) {
	lines := RenderClock("12:34:56 AM")
	if len(lines) < 2 {
		t.Fatal("expected at least 2 lines")
	}
	width := len([]rune(lines[0]))
	for i, line := range lines {
		w := len([]rune(line))
		if w != width {
			t.Errorf("line %d width %d != line 0 width %d", i, w, width)
		}
	}
}

func TestRenderClock_AllDigitsSupported(t *testing.T) {
	for _, ch := range "0123456789: AMP" {
		if _, ok := clockDigits[ch]; !ok {
			t.Errorf("missing glyph for %q", string(ch))
		}
	}
}

func TestRenderClock_UnknownCharFallsBackToSpace(t *testing.T) {
	lines := RenderClock("X")
	spaceLines := RenderClock(" ")
	for i := range lines {
		if lines[i] != spaceLines[i] {
			t.Errorf("unknown char 'X' line %d: got %q, want space %q", i, lines[i], spaceLines[i])
		}
	}
}

func TestGetDateTime_TimeFormat(t *testing.T) {
	timeStr, _ := GetDateTime()

	if !strings.Contains(timeStr, ":") {
		t.Errorf("time string %q missing colon", timeStr)
	}
	if !strings.HasSuffix(timeStr, "AM") && !strings.HasSuffix(timeStr, "PM") {
		t.Errorf("time string %q missing AM/PM suffix", timeStr)
	}
}

func TestGetDateTime_DateFormat(t *testing.T) {
	_, dateStr := GetDateTime()

	now := time.Now()
	yearStr := now.Format("2006")
	if !strings.Contains(dateStr, yearStr) {
		t.Errorf("date string %q missing current year %s", dateStr, yearStr)
	}
	if dateStr != strings.ToUpper(dateStr) {
		t.Errorf("date string %q should be all uppercase", dateStr)
	}
}

func TestRenderDateTime_Structure(t *testing.T) {
	lines := RenderDateTime()

	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines (3 clock + blank + date), got %d", len(lines))
	}

	for i := 0; i < 3; i++ {
		if strings.TrimSpace(lines[i]) == "" {
			t.Errorf("clock line %d should not be empty", i)
		}
	}

	if strings.TrimSpace(lines[3]) != "" {
		t.Errorf("line 3 should be blank separator, got %q", lines[3])
	}

	if strings.TrimSpace(lines[4]) == "" {
		t.Error("date line should not be empty")
	}
}

func TestCenterLines_Centering(t *testing.T) {
	lines := []string{"AB", "ABCD"}
	centered := CenterLines(lines, 10)

	if !strings.HasPrefix(centered[0], "    ") {
		t.Errorf("expected 4-space padding for 'AB' in width 10, got %q", centered[0])
	}

	if !strings.HasPrefix(centered[1], "   ") {
		t.Errorf("expected 3-space padding for 'ABCD' in width 10, got %q", centered[1])
	}
}

func TestCenterLines_LineWiderThanWidth(t *testing.T) {
	lines := []string{"ABCDEFGHIJ"}
	centered := CenterLines(lines, 5)

	if centered[0] != "ABCDEFGHIJ" {
		t.Errorf("line wider than width should be unchanged, got %q", centered[0])
	}
}

func TestCenterLinesBright_HasANSI(t *testing.T) {
	lines := []string{"HI"}
	bright := CenterLinesBright(lines, 20)

	if !strings.Contains(bright[0], "\x1b[38;2;255;255;255m") {
		t.Error("expected bright white ANSI code")
	}
	if !strings.Contains(bright[0], "\x1b[0m") {
		t.Error("expected ANSI reset code")
	}
}

func TestGetMaxLineWidth(t *testing.T) {
	tests := []struct {
		lines    []string
		expected int
	}{
		{[]string{"AB", "ABCD", "A"}, 4},
		{[]string{""}, 0},
		{[]string{"█▄▀"}, 3},
	}

	for _, tt := range tests {
		got := GetMaxLineWidth(tt.lines)
		if got != tt.expected {
			t.Errorf("GetMaxLineWidth(%v) = %d, want %d", tt.lines, got, tt.expected)
		}
	}
}
