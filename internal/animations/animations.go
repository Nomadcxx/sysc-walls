// animations.go - Animation handling
package animations

import (
	"fmt"
	"os/exec"
)

// Animation interface for all animations
type Animation interface {
	Update(frame int)
	Render() string
	Resize(width, height int)
}

// Factory function to create animations based on type
func CreateAnimation(effect string, width, height int, theme string) (Animation, error) {
	switch effect {
	case "matrix":
		return NewMatrixAnimation(width, height, theme)
	case "fire":
		return NewFireAnimation(width, height, theme)
	case "fireworks":
		return NewFireworksAnimation(width, height, theme)
	case "rain":
		return NewRainAnimation(width, height, theme)
	case "beams":
		return NewBeamsAnimation(width, height, theme)
	case "beam-text":
		return NewBeamTextAnimation(width, height, theme)
	case "decrypt":
		return NewDecryptAnimation(width, height, theme)
	case "pour":
		return NewPourAnimation(width, height, theme)
	case "aquarium":
		return NewAquariumAnimation(width, height, theme)
	case "print":
		return NewPrintAnimation(width, height, theme)
	default:
		return nil, fmt.Errorf("unknown animation effect: %s", effect)
	}
}

// Matrix animation using the sysc-Go library
type MatrixAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewMatrixAnimation(width, height int, theme string) (*MatrixAnimation, error) {
	return &MatrixAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (m *MatrixAnimation) Update(frame int) {
	m.frame = frame
	m.needsRender = true
}

func (m *MatrixAnimation) Render() string {
	if !m.needsRender {
		return m.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(m.command,
		"-effect", "matrix",
		"-theme", m.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering matrix animation: %v\nOutput: %s", err, string(output))
	}

	m.output = string(output)
	m.needsRender = false
	return m.output
}

func (m *MatrixAnimation) Resize(width, height int) {
	m.width = width
	m.height = height
	m.needsRender = true
}

// Fire animation using the sysc-Go library
type FireAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewFireAnimation(width, height int, theme string) (*FireAnimation, error) {
	return &FireAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (f *FireAnimation) Update(frame int) {
	f.frame = frame
	f.needsRender = true
}

func (f *FireAnimation) Render() string {
	if !f.needsRender {
		return f.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(f.command,
		"-effect", "fire",
		"-theme", f.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering fire animation: %v\nOutput: %s", err, string(output))
	}

	f.output = string(output)
	f.needsRender = false
	return f.output
}

func (f *FireAnimation) Resize(width, height int) {
	f.width = width
	f.height = height
	f.needsRender = true
}

// Fireworks animation using the sysc-Go library
type FireworksAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewFireworksAnimation(width, height int, theme string) (*FireworksAnimation, error) {
	return &FireworksAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (f *FireworksAnimation) Update(frame int) {
	f.frame = frame
	f.needsRender = true
}

func (f *FireworksAnimation) Render() string {
	if !f.needsRender {
		return f.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(f.command,
		"-effect", "fireworks",
		"-theme", f.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering fireworks animation: %v\nOutput: %s", err, string(output))
	}

	f.output = string(output)
	f.needsRender = false
	return f.output
}

func (f *FireworksAnimation) Resize(width, height int) {
	f.width = width
	f.height = height
	f.needsRender = true
}

// Rain animation using the sysc-Go library
type RainAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewRainAnimation(width, height int, theme string) (*RainAnimation, error) {
	return &RainAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (r *RainAnimation) Update(frame int) {
	r.frame = frame
	r.needsRender = true
}

func (r *RainAnimation) Render() string {
	if !r.needsRender {
		return r.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(r.command,
		"-effect", "rain",
		"-theme", r.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering rain animation: %v\nOutput: %s", err, string(output))
	}

	r.output = string(output)
	r.needsRender = false
	return r.output
}

func (r *RainAnimation) Resize(width, height int) {
	r.width = width
	r.height = height
	r.needsRender = true
}

// Beams animation using the sysc-Go library
type BeamsAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewBeamsAnimation(width, height int, theme string) (*BeamsAnimation, error) {
	return &BeamsAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (b *BeamsAnimation) Update(frame int) {
	b.frame = frame
	b.needsRender = true
}

func (b *BeamsAnimation) Render() string {
	if !b.needsRender {
		return b.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(b.command,
		"-effect", "beams",
		"-theme", b.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering beams animation: %v\nOutput: %s", err, string(output))
	}

	b.output = string(output)
	b.needsRender = false
	return b.output
}

func (b *BeamsAnimation) Resize(width, height int) {
	b.width = width
	b.height = height
	b.needsRender = true
}

// Beam Text animation using the sysc-Go library
type BeamTextAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewBeamTextAnimation(width, height int, theme string) (*BeamTextAnimation, error) {
	return &BeamTextAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (b *BeamTextAnimation) Update(frame int) {
	b.frame = frame
	b.needsRender = true
}

func (b *BeamTextAnimation) Render() string {
	if !b.needsRender {
		return b.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(b.command,
		"-effect", "beam-text",
		"-theme", b.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering beam-text animation: %v\nOutput: %s", err, string(output))
	}

	b.output = string(output)
	b.needsRender = false
	return b.output
}

func (b *BeamTextAnimation) Resize(width, height int) {
	b.width = width
	b.height = height
	b.needsRender = true
}

// Decrypt animation using the sysc-Go library
type DecryptAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewDecryptAnimation(width, height int, theme string) (*DecryptAnimation, error) {
	return &DecryptAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (d *DecryptAnimation) Update(frame int) {
	d.frame = frame
	d.needsRender = true
}

func (d *DecryptAnimation) Render() string {
	if !d.needsRender {
		return d.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(d.command,
		"-effect", "decrypt",
		"-theme", d.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering decrypt animation: %v\nOutput: %s", err, string(output))
	}

	d.output = string(output)
	d.needsRender = false
	return d.output
}

func (d *DecryptAnimation) Resize(width, height int) {
	d.width = width
	d.height = height
	d.needsRender = true
}

// Pour animation using the sysc-Go library
type PourAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewPourAnimation(width, height int, theme string) (*PourAnimation, error) {
	return &PourAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (p *PourAnimation) Update(frame int) {
	p.frame = frame
	p.needsRender = true
}

func (p *PourAnimation) Render() string {
	if !p.needsRender {
		return p.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(p.command,
		"-effect", "pour",
		"-theme", p.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering pour animation: %v\nOutput: %s", err, string(output))
	}

	p.output = string(output)
	p.needsRender = false
	return p.output
}

func (p *PourAnimation) Resize(width, height int) {
	p.width = width
	p.height = height
	p.needsRender = true
}

// Aquarium animation using the sysc-Go library
type AquariumAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewAquariumAnimation(width, height int, theme string) (*AquariumAnimation, error) {
	return &AquariumAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (a *AquariumAnimation) Update(frame int) {
	a.frame = frame
	a.needsRender = true
}

func (a *AquariumAnimation) Render() string {
	if !a.needsRender {
		return a.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(a.command,
		"-effect", "aquarium",
		"-theme", a.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering aquarium animation: %v\nOutput: %s", err, string(output))
	}

	a.output = string(output)
	a.needsRender = false
	return a.output
}

func (a *AquariumAnimation) Resize(width, height int) {
	a.width = width
	a.height = height
	a.needsRender = true
}

// Print animation using the sysc-Go library
type PrintAnimation struct {
	command     string
	width       int
	height      int
	theme       string
	frame       int
	output      string
	needsRender bool
}

func NewPrintAnimation(width, height int, theme string) (*PrintAnimation, error) {
	return &PrintAnimation{
		command:     "syscgo",
		width:       width,
		height:      height,
		theme:       theme,
		frame:       0,
		needsRender: true,
	}, nil
}

func (p *PrintAnimation) Update(frame int) {
	p.frame = frame
	p.needsRender = true
}

func (p *PrintAnimation) Render() string {
	if !p.needsRender {
		return p.output
	}

	// Use syscgo command-line tool for rendering
	cmd := exec.Command(p.command,
		"-effect", "print",
		"-theme", p.theme,
		"-duration", "1", // Just one frame
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error rendering print animation: %v\nOutput: %s", err, string(output))
	}

	p.output = string(output)
	p.needsRender = false
	return p.output
}

func (p *PrintAnimation) Resize(width, height int) {
	p.width = width
	p.height = height
	p.needsRender = true
}
