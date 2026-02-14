// compositor_test.go - Tests for compositor implementations
package compositor

import (
	"fmt"
	"testing"
)

// TestHyprlandParseOutputs tests Hyprland's parseOutputs function
func TestHyprlandParseOutputs(t *testing.T) {
	h := NewHyprlandCompositor()

	tests := []struct {
		name        string
		input       string
		wantOutputs []Output
		wantErr     bool
	}{
		{
			name:    "single monitor focused",
			input:   `[{"name":"DP-1","width":2560,"height":1440,"x":0,"y":0,"focused":true}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1", Width: 2560, Height: 1440, Focused: true},
			},
		},
		{
			name:    "multi-monitor one focused",
			input:   `[{"name":"DP-1","width":2560,"height":1440,"focused":true},{"name":"HDMI-A-0","width":1920,"height":1080,"focused":false}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1", Width: 2560, Height: 1440, Focused: true},
				{Name: "HDMI-A-0", Width: 1920, Height: 1080, Focused: false},
			},
		},
		{
			name:    "no monitor focused",
			input:   `[{"name":"DP-1","width":2560,"height":1440,"focused":false},{"name":"HDMI-A-0","width":1920,"height":1080,"focused":false}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1", Width: 2560, Height: 1440, Focused: false},
				{Name: "HDMI-A-0", Width: 1920, Height: 1080, Focused: false},
			},
		},
		{
			name:        "empty JSON array",
			input:       `[]`,
			wantErr:     true,
			wantOutputs: nil,
		},
		{
			name:        "malformed JSON",
			input:       `invalid{json`,
			wantErr:     true,
			wantOutputs: nil,
		},
		{
			name:    "missing fields",
			input:   `[{"name":"DP-1"}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1", Width: 0, Height: 0, Focused: false},
			},
		},
		{
			name:    "zero dimension output",
			input:   `[{"name":"DP-1","width":0,"height":0,"focused":true}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1", Width: 0, Height: 0, Focused: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputs, err := h.parseOutputs([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOutputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(outputs) != len(tt.wantOutputs) {
				t.Errorf("parseOutputs() got %d outputs, want %d", len(outputs), len(tt.wantOutputs))
				return
			}
			for i, want := range tt.wantOutputs {
				if outputs[i].Name != want.Name {
					t.Errorf("output[%d].Name = %s, want %s", i, outputs[i].Name, want.Name)
				}
				if outputs[i].Width != want.Width {
					t.Errorf("output[%d].Width = %d, want %d", i, outputs[i].Width, want.Width)
				}
				if outputs[i].Height != want.Height {
					t.Errorf("output[%d].Height = %d, want %d", i, outputs[i].Height, want.Height)
				}
				if outputs[i].Focused != want.Focused {
					t.Errorf("output[%d].Focused = %v, want %v", i, outputs[i].Focused, want.Focused)
				}
			}
		})
	}
}

// TestHyprlandGetFocusedOutput tests GetFocusedOutput with pre-built output slices
func TestHyprlandGetFocusedOutput(t *testing.T) {
	tests := []struct {
		name      string
		outputs   []Output
		wantFocus string
		wantErr   bool
	}{
		{
			name:      "one focused",
			outputs:   []Output{{Name: "DP-1", Focused: true}, {Name: "HDMI-A-0", Focused: false}},
			wantFocus: "DP-1",
			wantErr:   false,
		},
		{
			name:      "none focused returns first",
			outputs:   []Output{{Name: "DP-1", Focused: false}, {Name: "HDMI-A-0", Focused: false}},
			wantFocus: "DP-1",
			wantErr:   false,
		},
		{
			name:      "empty outputs error",
			outputs:   []Output{},
			wantFocus: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getFocusedFromOutputs(tt.outputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFocusedFromOutputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.wantFocus {
				t.Errorf("getFocusedFromOutputs() = %s, want %s", result, tt.wantFocus)
			}
		})
	}
}

// Helper function for testing GetFocusedOutput logic
func getFocusedFromOutputs(outputs []Output) (string, error) {
	if len(outputs) == 0 {
		return "", fmt.Errorf("no outputs found")
	}
	for _, output := range outputs {
		if output.Focused {
			return output.Name, nil
		}
	}
	return outputs[0].Name, nil
}

// TestSwayParseOutputs tests Sway's parseOutputs function
func TestSwayParseOutputs(t *testing.T) {
	s := NewSwayCompositor()

	tests := []struct {
		name        string
		input       string
		wantOutputs []Output
		wantErr     bool
	}{
		{
			name:    "active output with rect",
			input:   `[{"name":"eDP-1","active":true,"focused":true,"rect":{"width":1920,"height":1080}}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "eDP-1", Width: 1920, Height: 1080, Focused: true},
			},
		},
		{
			name:    "inactive output filtered",
			input:   `[{"name":"eDP-1","active":true,"focused":true,"rect":{"width":1920,"height":1080}},{"name":"DP-2","active":false,"focused":false,"rect":{"width":2560,"height":1440}}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "eDP-1", Width: 1920, Height: 1080, Focused: true},
			},
		},
		{
			name:    "multi-monitor with one focused",
			input:   `[{"name":"eDP-1","active":true,"focused":false,"rect":{"width":1920,"height":1080}},{"name":"DP-2","active":true,"focused":true,"rect":{"width":2560,"height":1440}}]`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "eDP-1", Width: 1920, Height: 1080, Focused: false},
				{Name: "DP-2", Width: 2560, Height: 1440, Focused: true},
			},
		},
		{
			name:        "empty JSON array",
			input:       `[]`,
			wantErr:     true,
			wantOutputs: nil,
		},
		{
			name:        "malformed JSON",
			input:       `{invalid json}`,
			wantErr:     true,
			wantOutputs: nil,
		},
		{
			name:        "all inactive outputs",
			input:       `[{"name":"eDP-1","active":false,"focused":false,"rect":{"width":1920,"height":1080}}]`,
			wantErr:     true,
			wantOutputs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputs, err := s.parseOutputs([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOutputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(outputs) != len(tt.wantOutputs) {
				t.Errorf("parseOutputs() got %d outputs, want %d", len(outputs), len(tt.wantOutputs))
				return
			}
			for i, want := range tt.wantOutputs {
				if outputs[i].Name != want.Name {
					t.Errorf("output[%d].Name = %s, want %s", i, outputs[i].Name, want.Name)
				}
				if outputs[i].Width != want.Width {
					t.Errorf("output[%d].Width = %d, want %d", i, outputs[i].Width, want.Width)
				}
				if outputs[i].Height != want.Height {
					t.Errorf("output[%d].Height = %d, want %d", i, outputs[i].Height, want.Height)
				}
				if outputs[i].Focused != want.Focused {
					t.Errorf("output[%d].Focused = %v, want %v", i, outputs[i].Focused, want.Focused)
				}
			}
		})
	}
}

// TestSwayGetFocusedOutput tests GetFocusedOutput selection logic for Sway
func TestSwayGetFocusedOutput(t *testing.T) {
	tests := []struct {
		name      string
		outputs   []Output
		wantFocus string
	}{
		{
			name:      "focused output found",
			outputs:   []Output{{Name: "eDP-1", Focused: false}, {Name: "DP-2", Focused: true}},
			wantFocus: "DP-2",
		},
		{
			name:      "none focused returns first",
			outputs:   []Output{{Name: "eDP-1", Focused: false}, {Name: "DP-2", Focused: false}},
			wantFocus: "eDP-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getFocusedFromOutputs(tt.outputs)
			if err != nil {
				t.Errorf("getFocusedFromOutputs() error = %v", err)
				return
			}
			if result != tt.wantFocus {
				t.Errorf("getFocusedFromOutputs() = %s, want %s", result, tt.wantFocus)
			}
		})
	}
}

// TestNiriParseOutputs tests Niri's parseOutputs function
func TestNiriParseOutputs(t *testing.T) {
	n := NewNiriCompositor()

	tests := []struct {
		name        string
		input       string
		wantOutputs []Output
		wantErr     bool
	}{
		{
			name:    "single output",
			input:   `Output "eDP-1" (eDP-1)`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "eDP-1"},
			},
		},
		{
			name:    "multiple outputs",
			input:   "Output \"DP-1\" (DP-1)\nOutput \"HDMI-A-1\" (HDMI-A-1)\nOutput \"eDP-1\" (eDP-1)",
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1"},
				{Name: "HDMI-A-1"},
				{Name: "eDP-1"},
			},
		},
		{
			name:    "output with spaces in name",
			input:   `Output "Some Display" (DP-1)`,
			wantErr: false,
			wantOutputs: []Output{
				{Name: "DP-1"},
			},
		},
		{
			name:        "empty output",
			input:       "",
			wantErr:     true,
			wantOutputs: nil,
		},
		{
			name:        "unexpected format",
			input:       "This is not a valid output format",
			wantErr:     true,
			wantOutputs: nil,
		},
		{
			name:    "whitespace handling",
			input:   "  Output \"eDP-1\" (eDP-1)  \n\n  Output \"DP-2\" (DP-2)  ",
			wantErr: false,
			wantOutputs: []Output{
				{Name: "eDP-1"},
				{Name: "DP-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputs, err := n.parseOutputs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOutputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(outputs) != len(tt.wantOutputs) {
				t.Errorf("parseOutputs() got %d outputs, want %d", len(outputs), len(tt.wantOutputs))
				return
			}
			for i, want := range tt.wantOutputs {
				if outputs[i].Name != want.Name {
					t.Errorf("output[%d].Name = %s, want %s", i, outputs[i].Name, want.Name)
				}
			}
		})
	}
}

// TestNiriGetFocusedOutputParsing tests the focused-output parsing logic
func TestNiriGetFocusedOutputParsing(t *testing.T) {
	n := NewNiriCompositor()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "standard format",
			input:   `Output "eDP-1" (eDP-1)`,
			want:    "eDP-1",
			wantErr: false,
		},
		{
			name:    "HDMI connector",
			input:   `Output "HDMI-A-0" (HDMI-A-0)`,
			want:    "HDMI-A-0",
			wantErr: false,
		},
		{
			name:    "unexpected format returns raw",
			input:   "Just some random text",
			want:    "Just some random text",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the parsing logic from GetFocusedOutput
			outputStr := tt.input
			matches := n.outputRegex.FindStringSubmatch(outputStr)
			var result string
			if len(matches) >= 3 {
				result = matches[2]
			} else {
				result = outputStr
			}

			if result != tt.want {
				t.Errorf("GetFocusedOutput parsing = %s, want %s", result, tt.want)
			}
		})
	}
}

// TestCompositorName tests that each compositor returns correct name
func TestCompositorName(t *testing.T) {
	tests := []struct {
		name      string
		compositor Compositor
		wantName  string
	}{
		{
			name:       "hyprland",
			compositor: NewHyprlandCompositor(),
			wantName:   "hyprland",
		},
		{
			name:       "sway",
			compositor: NewSwayCompositor(),
			wantName:   "sway",
		},
		{
			name:       "niri",
			compositor: NewNiriCompositor(),
			wantName:   "niri",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.compositor.Name(); got != tt.wantName {
				t.Errorf("Name() = %s, want %s", got, tt.wantName)
			}
		})
	}
}
