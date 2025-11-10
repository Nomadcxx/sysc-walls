// clock.go - ASCII clock rendering for datetime overlay
package clock

import (
	"strings"
	"time"
)

// ClockStyle represents a specific ASCII clock style
type ClockStyle string

const (
	StyleKompaktblk ClockStyle = "kompaktblk"
)

// Clock styles digits (using kompaktblk style from sysc-greet)
var clockDigits = map[rune][]string{
	'0': {
		"▄▀▀█▄ ",
		"█▄▀ █ ",
		" ▀▀▀  ",
	},
	'1': {
		" ▄█   ",
		"  █   ",
		"▀▀▀▀▀ ",
	},
	'2': {
		"▀▀▀▀█ ",
		"█▀▀▀▀ ",
		"▀▀▀▀▀ ",
	},
	'3': {
		"▀▀▀▀▄ ",
		"  ▀▀▄ ",
		"▀▀▀▀  ",
	},
	'4': {
		"█   █ ",
		"▀▀▀▀█ ",
		"    ▀ ",
	},
	'5': {
		"█▀▀▀▀ ",
		"▀▀▀▀█ ",
		"▀▀▀▀▀ ",
	},
	'6': {
		"█▀▀▀▀ ",
		"█▀▀▀█ ",
		"▀▀▀▀▀ ",
	},
	'7': {
		"▀▀▀▀█ ",
		"   █▀ ",
		"   ▀  ",
	},
	'8': {
		"█▀▀▀█ ",
		"█▀▀▀█ ",
		"▀▀▀▀▀ ",
	},
	'9': {
		"█▀▀▀█ ",
		"▀▀▀▀█ ",
		"▀▀▀▀▀ ",
	},
	':': {
		"  ▄   ",
		"  ▄   ",
		"      ",
	},
	' ': {
		"      ",
		"      ",
		"      ",
	},
	'A': {
		"▄▀▀▀▄ ",
		"█▀▀▀█ ",
		"▀   ▀ ",
	},
	'M': {
		"█▀▄▀█ ",
		"█   █ ",
		"▀   ▀ ",
	},
	'P': {
		"█▀▀▀▄ ",
		"█▀▀▀  ",
		"▀     ",
	},
}

// RenderClock renders time string using ASCII art
func RenderClock(timeStr string) []string {
	// Get the height from first digit
	if len(clockDigits['0']) == 0 {
		return []string{timeStr}
	}
	height := len(clockDigits['0'])

	// Build each line of the clock
	var lines []string
	for row := 0; row < height; row++ {
		var line strings.Builder
		for _, ch := range timeStr {
			digitLines, ok := clockDigits[ch]
			if !ok {
				// Unknown character, use space
				digitLines = clockDigits[' ']
			}
			if row < len(digitLines) {
				line.WriteString(digitLines[row])
			}
		}
		lines = append(lines, line.String())
	}
	return lines
}

// GetDateTime returns formatted time and date strings
func GetDateTime() (timeStr string, dateStr string) {
	now := time.Now()
	// Format time like "3:04:05 PM"
	timeStr = now.Format("3:04:05 PM")
	// Pad single-digit hours for consistent width
	if len(timeStr) > 1 && timeStr[0] != '1' && timeStr[1] == ':' {
		timeStr = " " + timeStr
	}
	// Format date like "MONDAY, JANUARY 2, 2006"
	dateStr = strings.ToUpper(now.Format("Monday, January 2, 2006"))
	return
}

// RenderDateTime renders the complete date-time overlay
func RenderDateTime() []string {
	timeStr, dateStr := GetDateTime()
	clockLines := RenderClock(timeStr)

	// Combine clock lines and date
	result := make([]string, 0, len(clockLines)+2)
	result = append(result, clockLines...)
	result = append(result, "") // Blank line
	result = append(result, dateStr)

	return result
}

// CenterLines centers each line in the given width
func CenterLines(lines []string, width int) []string {
	centered := make([]string, len(lines))
	for i, line := range lines {
		lineLen := len([]rune(line))
		if lineLen >= width {
			centered[i] = line
		} else {
			padding := (width - lineLen) / 2
			centered[i] = strings.Repeat(" ", padding) + line
		}
	}
	return centered
}

// GetMaxLineWidth returns the maximum width of all lines
func GetMaxLineWidth(lines []string) int {
	maxWidth := 0
	for _, line := range lines {
		width := len([]rune(line))
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}
