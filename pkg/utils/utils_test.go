package utils

import "testing"

func TestTerminalCellSizeUsesPixelGeometry(t *testing.T) {
	gotWidth, gotHeight := terminalCellSize(&winsize{Col: 110, Row: 32, Xpixel: 990, Ypixel: 640})
	if gotWidth != 9 || gotHeight != 20 {
		t.Fatalf("cell size = %dx%d, want 9x20", gotWidth, gotHeight)
	}

	if gotWidth, gotHeight = terminalCellSize(&winsize{Col: 110, Row: 32}); gotWidth != 0 || gotHeight != 0 {
		t.Fatalf("cell size without pixel geometry = %dx%d, want 0x0", gotWidth, gotHeight)
	}
}
