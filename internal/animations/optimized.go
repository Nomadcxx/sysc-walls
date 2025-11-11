// optimized.go - Optimized animation handling using sysc-Go library directly
package animations

import (
	"fmt"
	"strings"

	syscGo "github.com/Nomadcxx/sysc-Go/animations"
)

// CreateOptimizedAnimation creates an optimized animation using sysc-Go library directly
func CreateOptimizedAnimation(effect string, width, height int, theme string) (Animation, error) {
	return CreateOptimizedAnimationWithText(effect, width, height, theme, "")
}

// CreateOptimizedAnimationWithText creates an optimized animation with custom text for text-based effects
func CreateOptimizedAnimationWithText(effect string, width, height int, theme string, text string) (Animation, error) {
	palette := getThemePalette(theme)

	// Default text if empty
	if text == "" {
		text = "SYSC-WALLS"
	}

	switch effect {
	case "matrix":
		return newOptimizedMatrix(width, height, palette)
	case "matrix-art":
		return newOptimizedMatrixArt(width, height, palette, text)
	case "fire":
		return newOptimizedFire(width, height, palette)
	case "fireworks":
		return newOptimizedFireworks(width, height, palette)
	case "rain":
		return newOptimizedRain(width, height, palette)
	case "rain-art":
		return newOptimizedRainArt(width, height, palette, text)
	case "beams":
		return newOptimizedBeams(width, height, palette)
	case "beam-text":
		return newOptimizedBeamText(width, height, palette, text)
	case "decrypt":
		return newOptimizedDecrypt(width, height, palette)
	case "pour":
		return newOptimizedPour(width, height, palette)
	case "aquarium":
		return newOptimizedAquarium(width, height, palette)
	case "print":
		return newOptimizedPrint(width, height, palette)
	case "blackhole":
		return newOptimizedBlackhole(width, height, palette, text)
	case "ring-text":
		return newOptimizedRingText(width, height, palette, text)
	default:
		return nil, fmt.Errorf("unknown animation effect: %s", effect)
	}
}

// getThemePalette returns color palette for theme
func getThemePalette(theme string) []string {
	palettes := map[string][]string{
		"dracula":        {"#282a36", "#44475a", "#f8f8f2", "#6272a4", "#8be9fd", "#50fa7b", "#ffb86c", "#ff79c6", "#bd93f9", "#ff5555", "#f1fa8c"},
		"gruvbox":        {"#282828", "#cc241d", "#98971a", "#d79921", "#458588", "#b16286", "#689d6a", "#a89984", "#928374", "#fb4934", "#b8bb26", "#fabd2f", "#83a598", "#d3869b", "#8ec07c", "#ebdbb2"},
		"nord":           {"#2e3440", "#3b4252", "#434c5e", "#4c566a", "#d8dee9", "#e5e9f0", "#eceff4", "#8fbcbb", "#88c0d0", "#81a1c1", "#5e81ac", "#bf616a", "#d08770", "#ebcb8b", "#a3be8c", "#b48ead"},
		"tokyo-night":    {"#1a1b26", "#24283b", "#414868", "#565f89", "#787c99", "#a9b1d6", "#c0caf5", "#7aa2f7", "#bb9af7", "#7dcfff", "#73daca", "#9ece6a", "#e0af68", "#f7768e", "#ff9e64", "#db4b4b"},
		"catppuccin":     {"#1e1e2e", "#181825", "#313244", "#45475a", "#585b70", "#cdd6f4", "#f5e0dc", "#f2cdcd", "#f5c2e7", "#cba6f7", "#f38ba8", "#eba0ac", "#fab387", "#f9e2af", "#a6e3a1", "#94e2d5", "#89dceb", "#74c7ec", "#89b4fa", "#b4befe"},
		"material":       {"#263238", "#2e3c43", "#314549", "#37474f", "#607d8b", "#546e7a", "#b0bec5", "#80cbc4", "#4dd0e1", "#4fc3f7", "#29b6f6", "#039be5", "#0288d1", "#0277bd", "#01579b"},
		"solarized":      {"#002b36", "#073642", "#586e75", "#657b83", "#839496", "#93a1a1", "#eee8d5", "#fdf6e3", "#b58900", "#cb4b16", "#dc322f", "#d33682", "#6c71c4", "#268bd2", "#2aa198", "#859900"},
		"monochrome":      {"#000000", "#1a1a1a", "#333333", "#4d4d4d", "#666666", "#808080", "#999999", "#b3b3b3", "#cccccc", "#e6e6e6", "#ffffff"},
		"trainsishardjob": {"#000000", "#ff00ff", "#00ffff", "#ff0000", "#00ff00", "#0000ff", "#ffff00", "#ffffff"},
		"rama":            {"#2b2d42", "#8d99ae", "#d90429", "#ef233c", "#edf2f4", "#ef233c", "#d90429", "#8d99ae", "#edf2f4"},
		"eldritch":       {"#212337", "#292e42", "#7081d0", "#04d1f9", "#37f499", "#f16c75", "#a48cf2", "#f265b5", "#f7c67f", "#ebfafa"},
		"dark":           {"#000000", "#1a1a1a", "#333333", "#4d4d4d", "#666666", "#808080", "#999999", "#b3b3b3", "#cccccc", "#e6e6e6", "#ffffff"},
	}

	if palette, ok := palettes[theme]; ok {
		return palette
	}
	return palettes["rama"] // Default to rama
}

// Helper function
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Matrix - uses simple constructor
type optimizedMatrix struct {
	effect *syscGo.MatrixEffect
}

func newOptimizedMatrix(width, height int, palette []string) (*optimizedMatrix, error) {
	return &optimizedMatrix{
		effect: syscGo.NewMatrixEffect(width, height, palette),
	}, nil
}

func (m *optimizedMatrix) Update(frame int) {
	m.effect.Update()
}

func (m *optimizedMatrix) Render() string {
	return m.effect.Render()
}

func (m *optimizedMatrix) Resize(width, height int) {
	m.effect.Resize(width, height)
}

// Fire - uses simple constructor
type optimizedFire struct {
	effect *syscGo.FireEffect
}

func newOptimizedFire(width, height int, palette []string) (*optimizedFire, error) {
	return &optimizedFire{
		effect: syscGo.NewFireEffect(width, height, palette),
	}, nil
}

func (f *optimizedFire) Update(frame int) {
	f.effect.Update()
}

func (f *optimizedFire) Render() string {
	return f.effect.Render()
}

func (f *optimizedFire) Resize(width, height int) {
	f.effect.Resize(width, height)
}

// Fireworks - uses simple constructor
type optimizedFireworks struct {
	effect *syscGo.FireworksEffect
}

func newOptimizedFireworks(width, height int, palette []string) (*optimizedFireworks, error) {
	return &optimizedFireworks{
		effect: syscGo.NewFireworksEffect(width, height, palette),
	}, nil
}

func (f *optimizedFireworks) Update(frame int) {
	f.effect.Update()
}

func (f *optimizedFireworks) Render() string {
	return f.effect.Render()
}

func (f *optimizedFireworks) Resize(width, height int) {
	f.effect.Resize(width, height)
}

// Rain - uses simple constructor
type optimizedRain struct {
	effect *syscGo.RainEffect
}

func newOptimizedRain(width, height int, palette []string) (*optimizedRain, error) {
	return &optimizedRain{
		effect: syscGo.NewRainEffect(width, height, palette),
	}, nil
}

func (r *optimizedRain) Update(frame int) {
	r.effect.Update()
}

func (r *optimizedRain) Render() string {
	return r.effect.Render()
}

func (r *optimizedRain) Resize(width, height int) {
	r.effect.Resize(width, height)
}

// Beams - uses config struct
type optimizedBeams struct {
	effect  *syscGo.BeamsEffect
	palette []string
}

func newOptimizedBeams(width, height int, palette []string) (*optimizedBeams, error) {
	config := syscGo.BeamsConfig{
		Width:             width,
		Height:            height,
		BeamGradientStops: palette[:minInt(len(palette), 5)],
	}
	return &optimizedBeams{
		effect:  syscGo.NewBeamsEffect(config),
		palette: palette,
	}, nil
}

func (b *optimizedBeams) Update(frame int) {
	b.effect.Update()
}

func (b *optimizedBeams) Render() string {
	return b.effect.Render()
}

func (b *optimizedBeams) Resize(width, height int) {
	config := syscGo.BeamsConfig{
		Width:             width,
		Height:            height,
		BeamGradientStops: b.palette[:minInt(len(b.palette), 5)],
	}
	b.effect = syscGo.NewBeamsEffect(config)
}

// BeamText - uses config struct with auto-sizing and centering wrapper
type optimizedBeamText struct {
	effect       *syscGo.BeamTextEffect
	palette      []string
	text         string
	termWidth    int
	termHeight   int
}

func newOptimizedBeamText(width, height int, palette []string, text string) (*optimizedBeamText, error) {
	config := syscGo.BeamTextConfig{
		Width:             0, // Not used when Auto=true
		Height:            0, // Not used when Auto=true
		Text:              text,
		Auto:              true, // Auto-size to text dimensions
		BeamGradientStops: palette[:minInt(len(palette), 5)],
	}
	return &optimizedBeamText{
		effect:     syscGo.NewBeamTextEffect(config),
		palette:    palette,
		text:       text,
		termWidth:  width,
		termHeight: height,
	}, nil
}

func (b *optimizedBeamText) Update(frame int) {
	b.effect.Update()
}

func (b *optimizedBeamText) Render() string {
	// Get the effect output (auto-sized to text)
	output := b.effect.Render()

	// Center it in the terminal
	return centerOutput(output, b.termWidth, b.termHeight)
}

func (b *optimizedBeamText) Resize(width, height int) {
	b.termWidth = width
	b.termHeight = height
	config := syscGo.BeamTextConfig{
		Width:             0,
		Height:            0,
		Text:              b.text,
		Auto:              true,
		BeamGradientStops: b.palette[:minInt(len(b.palette), 5)],
	}
	b.effect = syscGo.NewBeamTextEffect(config)
}

// Decrypt - uses config struct
type optimizedDecrypt struct {
	effect  *syscGo.DecryptEffect
	palette []string
}

func newOptimizedDecrypt(width, height int, palette []string) (*optimizedDecrypt, error) {
	config := syscGo.DecryptConfig{
		Width:   width,
		Height:  height,
		Palette: palette,
	}
	return &optimizedDecrypt{
		effect:  syscGo.NewDecryptEffect(config),
		palette: palette,
	}, nil
}

func (d *optimizedDecrypt) Update(frame int) {
	d.effect.Update()
}

func (d *optimizedDecrypt) Render() string {
	return d.effect.Render()
}

func (d *optimizedDecrypt) Resize(width, height int) {
	config := syscGo.DecryptConfig{
		Width:   width,
		Height:  height,
		Palette: d.palette,
	}
	d.effect = syscGo.NewDecryptEffect(config)
}

// Pour - uses config struct
type optimizedPour struct {
	effect  *syscGo.PourEffect
	palette []string
}

func newOptimizedPour(width, height int, palette []string) (*optimizedPour, error) {
	config := syscGo.PourConfig{
		Width:  width,
		Height: height,
	}
	return &optimizedPour{
		effect:  syscGo.NewPourEffect(config),
		palette: palette,
	}, nil
}

func (p *optimizedPour) Update(frame int) {
	p.effect.Update()
}

func (p *optimizedPour) Render() string {
	return p.effect.Render()
}

func (p *optimizedPour) Resize(width, height int) {
	config := syscGo.PourConfig{
		Width:  width,
		Height: height,
	}
	p.effect = syscGo.NewPourEffect(config)
}

// Aquarium - uses config struct
type optimizedAquarium struct {
	effect  *syscGo.AquariumEffect
	palette []string
}

func newOptimizedAquarium(width, height int, palette []string) (*optimizedAquarium, error) {
	// Split palette into appropriate color groups
	fishColors := palette[:minInt(len(palette), 3)]
	waterColors := []string{"#2e3440", "#3b4252", "#434c5e"}
	if len(palette) > 3 {
		waterColors = palette[3:minInt(len(palette), 6)]
	}
	seaweedColors := []string{"#a3be8c", "#8fbcbb"}
	if len(palette) > 6 {
		seaweedColors = palette[6:minInt(len(palette), 8)]
	}

	config := syscGo.AquariumConfig{
		Width:         width,
		Height:        height,
		FishColors:    fishColors,
		WaterColors:   waterColors,
		SeaweedColors: seaweedColors,
		BubbleColor:   "#88c0d0",
		DiverColor:    "#d08770",
		BoatColor:     "#bf616a",
		MermaidColor:  "#b48ead",
		AnchorColor:   "#5e81ac",
	}
	return &optimizedAquarium{
		effect:  syscGo.NewAquariumEffect(config),
		palette: palette,
	}, nil
}

func (a *optimizedAquarium) Update(frame int) {
	a.effect.Update()
}

func (a *optimizedAquarium) Render() string {
	return a.effect.Render()
}

func (a *optimizedAquarium) Resize(width, height int) {
	// Aquarium resize needs full reconfiguration
	fishColors := a.palette[:minInt(len(a.palette), 3)]
	waterColors := []string{"#2e3440", "#3b4252", "#434c5e"}
	if len(a.palette) > 3 {
		waterColors = a.palette[3:minInt(len(a.palette), 6)]
	}
	seaweedColors := []string{"#a3be8c", "#8fbcbb"}
	if len(a.palette) > 6 {
		seaweedColors = a.palette[6:minInt(len(a.palette), 8)]
	}

	config := syscGo.AquariumConfig{
		Width:         width,
		Height:        height,
		FishColors:    fishColors,
		WaterColors:   waterColors,
		SeaweedColors: seaweedColors,
		BubbleColor:   "#88c0d0",
		DiverColor:    "#d08770",
		BoatColor:     "#bf616a",
		MermaidColor:  "#b48ead",
		AnchorColor:   "#5e81ac",
	}
	a.effect = syscGo.NewAquariumEffect(config)
}

// Print - uses config struct
type optimizedPrint struct {
	effect  *syscGo.PrintEffect
	palette []string
}

func newOptimizedPrint(width, height int, palette []string) (*optimizedPrint, error) {
	config := syscGo.PrintConfig{
		Width:  width,
		Height: height,
	}
	return &optimizedPrint{
		effect:  syscGo.NewPrintEffect(config),
		palette: palette,
	}, nil
}

func (p *optimizedPrint) Update(frame int) {
	p.effect.Update()
}

func (p *optimizedPrint) Render() string {
	return p.effect.Render()
}

func (p *optimizedPrint) Resize(width, height int) {
	config := syscGo.PrintConfig{
		Width:  width,
		Height: height,
	}
	p.effect = syscGo.NewPrintEffect(config)
}

// MatrixArt - Matrix rain that crystallizes into ASCII art
type optimizedMatrixArt struct {
	effect  *syscGo.MatrixArtEffect
	palette []string
	text    string
}

func newOptimizedMatrixArt(width, height int, palette []string, text string) (*optimizedMatrixArt, error) {
	return &optimizedMatrixArt{
		effect:  syscGo.NewMatrixArtEffect(width, height, palette, text),
		palette: palette,
		text:    text,
	}, nil
}

func (m *optimizedMatrixArt) Update(frame int) {
	m.effect.Update()
}

func (m *optimizedMatrixArt) Render() string {
	return m.effect.Render()
}

func (m *optimizedMatrixArt) Resize(width, height int) {
	m.effect = syscGo.NewMatrixArtEffect(width, height, m.palette, m.text)
}

// RainArt - Rain drops that freeze to form ASCII art
type optimizedRainArt struct {
	effect  *syscGo.RainArtEffect
	palette []string
	text    string
}

func newOptimizedRainArt(width, height int, palette []string, text string) (*optimizedRainArt, error) {
	return &optimizedRainArt{
		effect:  syscGo.NewRainArtEffect(width, height, palette, text),
		palette: palette,
		text:    text,
	}, nil
}

func (r *optimizedRainArt) Update(frame int) {
	r.effect.Update()
}

func (r *optimizedRainArt) Render() string {
	return r.effect.Render()
}

func (r *optimizedRainArt) Resize(width, height int) {
	r.effect = syscGo.NewRainArtEffect(width, height, r.palette, r.text)
}

// Blackhole - Text gets consumed by a blackhole and explodes
type optimizedBlackhole struct {
	effect  *syscGo.BlackholeEffect
	palette []string
	text    string
}

func newOptimizedBlackhole(width, height int, palette []string, text string) (*optimizedBlackhole, error) {
	config := syscGo.BlackholeConfig{
		Width:               width,
		Height:              height,
		Text:                text,
		BlackholeColor:      "#ffffff",
		StarColors:          palette[:minInt(len(palette), 6)],
		FinalGradientStops:  palette[:minInt(len(palette), 3)],
		StaticGradientStops: palette[:minInt(len(palette), 6)],
		StaticGradientDir:   syscGo.GradientHorizontal,
	}
	return &optimizedBlackhole{
		effect:  syscGo.NewBlackholeEffect(config),
		palette: palette,
		text:    text,
	}, nil
}

func (b *optimizedBlackhole) Update(frame int) {
	b.effect.Update()
}

func (b *optimizedBlackhole) Render() string {
	return b.effect.Render()
}

func (b *optimizedBlackhole) Resize(width, height int) {
	config := syscGo.BlackholeConfig{
		Width:               width,
		Height:              height,
		Text:                b.text,
		BlackholeColor:      "#ffffff",
		StarColors:          b.palette[:minInt(len(b.palette), 6)],
		FinalGradientStops:  b.palette[:minInt(len(b.palette), 3)],
		StaticGradientStops: b.palette[:minInt(len(b.palette), 6)],
		StaticGradientDir:   syscGo.GradientHorizontal,
	}
	b.effect = syscGo.NewBlackholeEffect(config)
}

// RingText - Text spins on concentric rings with vortex motion
type optimizedRingText struct {
	effect  *syscGo.RingTextEffect
	palette []string
	text    string
}

func newOptimizedRingText(width, height int, palette []string, text string) (*optimizedRingText, error) {
	config := syscGo.RingTextConfig{
		Width:               width,
		Height:              height,
		Text:                text,
		RingColors:          palette[:minInt(len(palette), 3)],
		FinalGradientStops:  palette[:minInt(len(palette), 3)],
		StaticGradientStops: palette[:minInt(len(palette), 3)],
		StaticGradientDir:   syscGo.GradientHorizontal,
	}
	return &optimizedRingText{
		effect:  syscGo.NewRingTextEffect(config),
		palette: palette,
		text:    text,
	}, nil
}

func (r *optimizedRingText) Update(frame int) {
	r.effect.Update()
}

func (r *optimizedRingText) Render() string {
	return r.effect.Render()
}

func (r *optimizedRingText) Resize(width, height int) {
	config := syscGo.RingTextConfig{
		Width:               width,
		Height:              height,
		Text:                r.text,
		RingColors:          r.palette[:minInt(len(r.palette), 3)],
		FinalGradientStops:  r.palette[:minInt(len(r.palette), 3)],
		StaticGradientStops: r.palette[:minInt(len(r.palette), 3)],
		StaticGradientDir:   syscGo.GradientHorizontal,
	}
	r.effect = syscGo.NewRingTextEffect(config)
}

// centerOutput centers smaller animation output in a larger terminal
func centerOutput(output string, termWidth, termHeight int) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return output
	}

	// Calculate vertical offset to center
	outputHeight := len(lines)
	verticalOffset := (termHeight - outputHeight) / 2
	if verticalOffset < 0 {
		verticalOffset = 0
	}

	// Find max line width (ignoring ANSI codes for width calculation)
	maxWidth := 0
	for _, line := range lines {
		// Simple width calculation - could be improved to strip ANSI
		visualWidth := len([]rune(line))
		if visualWidth > maxWidth {
			maxWidth = visualWidth
		}
	}

	// Calculate horizontal offset to center
	horizontalOffset := (termWidth - maxWidth) / 2
	if horizontalOffset < 0 {
		horizontalOffset = 0
	}

	// Build centered output
	var result strings.Builder

	// Add top padding
	for i := 0; i < verticalOffset; i++ {
		result.WriteString("\n")
	}

	// Add horizontally centered lines
	padding := strings.Repeat(" ", horizontalOffset)
	for _, line := range lines {
		result.WriteString(padding)
		result.WriteString(line)
		result.WriteString("\n")
	}

	// Add bottom padding to fill terminal
	remainingLines := termHeight - verticalOffset - outputHeight
	for i := 0; i < remainingLines; i++ {
		result.WriteString("\n")
	}

	return result.String()
}
