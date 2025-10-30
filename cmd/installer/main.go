package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Theme colors - sysc-walls themed colors
var (
	BgBase       = lipgloss.Color("#0a0e1a")
	Primary      = lipgloss.Color("#00c2ff")
	Secondary    = lipgloss.Color("#8b95ff")
	Accent       = lipgloss.Color("#00ff88")
	FgPrimary    = lipgloss.Color("#e6e6e6")
	FgSecondary  = lipgloss.Color("#b0b0b0")
	FgMuted      = lipgloss.Color("#666666")
	ErrorColor   = lipgloss.Color("#ff4d4d")
	WarningColor = lipgloss.Color("#ffbb33")
)

// Styles
var (
	checkMark   = lipgloss.NewStyle().Foreground(Accent).SetString("[OK]")
	failMark    = lipgloss.NewStyle().Foreground(ErrorColor).SetString("[FAIL]")
	skipMark    = lipgloss.NewStyle().Foreground(WarningColor).SetString("[SKIP]")
	headerStyle = lipgloss.NewStyle().Foreground(Primary).Bold(true)
)

type installStep int

const (
	stepWelcome installStep = iota
	stepInstalling
	stepComplete
)

type taskStatus int

const (
	statusPending taskStatus = iota
	statusRunning
	statusComplete
	statusFailed
	statusSkipped
)

type installTask struct {
	name        string
	description string
	execute     func(*model) error
	optional    bool
	status      taskStatus
}

type model struct {
	step             installStep
	tasks            []installTask
	currentTaskIndex int
	width            int
	height           int
	spinner          spinner.Model
	errors           []string
	uninstallMode    bool
	selectedOption   int // 0 = Install, 1 = Uninstall
}

type taskCompleteMsg struct {
	index   int
	success bool
	error   string
}

func newModel() model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(Secondary)
	s.Spinner = spinner.Dot

	return model{
		step:             stepWelcome,
		currentTaskIndex: -1,
		spinner:          s,
		errors:           []string{},
		selectedOption:   0,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Allow exit from any step except during installation
			if m.step != stepInstalling {
				return m, tea.Quit
			}
		case "up", "k":
			if m.step == stepWelcome && m.selectedOption > 0 {
				m.selectedOption--
			}
		case "down", "j":
			if m.step == stepWelcome && m.selectedOption < 1 {
				m.selectedOption++
			}
		case "enter":
			if m.step == stepWelcome {
				m.uninstallMode = m.selectedOption == 1
				m.initTasks()
				m.step = stepInstalling
				m.currentTaskIndex = 0
				m.tasks[0].status = statusRunning
				return m, tea.Batch(
					m.spinner.Tick,
					executeTask(0, &m),
				)
			} else if m.step == stepComplete {
				return m, tea.Quit
			}
		}

	case taskCompleteMsg:
		// Update task status
		if msg.success {
			m.tasks[msg.index].status = statusComplete
		} else {
			if m.tasks[msg.index].optional {
				m.tasks[msg.index].status = statusSkipped
				m.errors = append(m.errors, fmt.Sprintf("%s (skipped): %s", m.tasks[msg.index].name, msg.error))
			} else {
				m.tasks[msg.index].status = statusFailed
				m.errors = append(m.errors, fmt.Sprintf("%s: %s", m.tasks[msg.index].name, msg.error))
				m.step = stepComplete
				return m, nil
			}
		}

		// Move to next task
		m.currentTaskIndex++
		if m.currentTaskIndex >= len(m.tasks) {
			m.step = stepComplete
			return m, nil
		}

		// Start next task
		m.tasks[m.currentTaskIndex].status = statusRunning
		return m, executeTask(m.currentTaskIndex, &m)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) initTasks() {
	if m.uninstallMode {
		m.tasks = []installTask{
			{name: "Check privileges", description: "Checking root access", execute: checkPrivileges, status: statusPending},
			{name: "Stop daemon", description: "Stopping sysc-walls daemon if running", execute: stopDaemon, status: statusPending},
			{name: "Remove binaries", description: "Removing /usr/local/bin/sysc-walls-*", execute: removeBinaries, status: statusPending},
			{name: "Remove systemd service", description: "Removing systemd service", execute: removeSystemdService, status: statusPending},
		}
	} else {
		m.tasks = []installTask{
			{name: "Check privileges", description: "Checking root access", execute: checkPrivileges, status: statusPending},
			{name: "Build binaries", description: "Building sysc-walls components", execute: buildBinaries, status: statusPending},
			{name: "Install binaries", description: "Installing to /usr/local/bin", execute: installBinaries, status: statusPending},
			{name: "Install systemd service", description: "Installing systemd service", execute: installSystemdService, status: statusPending},
			{name: "Enable systemd service", description: "Enabling systemd service", execute: enableSystemdService, status: statusPending, optional: true},
		}
	}
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content strings.Builder

	// ASCII Header - sysc-walls themed ASCII art
	headerLines := []string{
		"▄▀▀▀▀ █   █ ▄▀▀▀▀ ▄▀▀▀▀          ▄▀ █   █ ▄▀▀▀▀ ▄▀▀▀▀      ",
		" ▀▀▀▄ ▀▀▀▀█  ▀▀▀▄ █     ▀▀▀▀▀  ▄▀   █ █ █  ▀▀▀▄ █     ▀▀▀▀▀ ",
		"▀▀▀▀  ▀▀▀▀▀ ▀▀▀▀   ▀▀▀▀       ▀      ▀ ▀ ▀▀▀▀  ▀▀▀▀   ▀▀▀▀▀  ",
		"            TERMINAL SCREENSAVER INSTALLER           ",
	}

	for _, line := range headerLines {
		content.WriteString(headerStyle.Render(line))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Main content based on step
	var mainContent string
	switch m.step {
	case stepWelcome:
		mainContent = m.renderWelcome()
	case stepInstalling:
		mainContent = m.renderInstalling()
	case stepComplete:
		mainContent = m.renderComplete()
	}

	// Wrap in border
	mainStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Width(m.width - 4)
	content.WriteString(mainStyle.Render(mainContent))
	content.WriteString("\n")

	// Help text
	helpText := m.getHelpText()
	if helpText != "" {
		helpStyle := lipgloss.NewStyle().
			Foreground(FgMuted).
			Italic(true).
			Align(lipgloss.Center)
		content.WriteString("\n" + helpStyle.Render(helpText))
	}

	// Wrap everything in background with centering
	bgStyle := lipgloss.NewStyle().
		Background(BgBase).
		Foreground(FgPrimary).
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Top)

	return bgStyle.Render(content.String())
}

func (m model) renderWelcome() string {
	var b strings.Builder

	b.WriteString("Select an option:\n\n")

	// Install option
	installPrefix := "  "
	if m.selectedOption == 0 {
		installPrefix = lipgloss.NewStyle().Foreground(Primary).Render("▸ ")
	}
	b.WriteString(installPrefix + "Install sysc-walls\n")
	b.WriteString("    Builds binaries and installs system-wide to /usr/local/bin\n\n")

	// Uninstall option
	uninstallPrefix := "  "
	if m.selectedOption == 1 {
		uninstallPrefix = lipgloss.NewStyle().Foreground(Primary).Render("▸ ")
	}
	b.WriteString(uninstallPrefix + "Uninstall sysc-walls\n")
	b.WriteString("    Removes sysc-walls from your system\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("Requires root privileges"))

	return b.String()
}

func (m model) renderInstalling() string {
	var b strings.Builder

	// Render all tasks with their current status
	for i, task := range m.tasks {
		var line string
		switch task.status {
		case statusPending:
			line = lipgloss.NewStyle().Foreground(FgMuted).Render("  " + task.name)
		case statusRunning:
			line = m.spinner.View() + " " + lipgloss.NewStyle().Foreground(Secondary).Render(task.description)
		case statusComplete:
			line = checkMark.String() + " " + task.name
		case statusFailed:
			line = failMark.String() + " " + task.name
		case statusSkipped:
			line = skipMark.String() + " " + task.name
		}

		b.WriteString(line)
		if i < len(m.tasks)-1 {
			b.WriteString("\n")
		}
	}

	// Show errors at bottom if any
	if len(m.errors) > 0 {
		b.WriteString("\n\n")
		for _, err := range m.errors {
			b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Render(err))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m model) renderComplete() string {
	var b strings.Builder

	if len(m.errors) > 0 {
		// Installation failed
		b.WriteString(lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render("Installation failed"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("Errors encountered:"))
		b.WriteString("\n")
		for _, err := range m.errors {
			b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Render("• " + err))
			b.WriteString("\n")
		}
	} else {
		// Installation succeeded
		if m.uninstallMode {
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Bold(true).Render("✓ Uninstallation complete!"))
			b.WriteString("\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("sysc-walls has been removed from your system."))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Bold(true).Render("✓ Installation complete!"))
			b.WriteString("\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("sysc-walls is now installed at /usr/local/bin"))
			b.WriteString("\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("Try it out:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("  systemctl enable sysc-walls.service"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("  systemctl start sysc-walls.service"))
		}
	}

	return b.String()
}

func (m model) getHelpText() string {
	switch m.step {
	case stepWelcome:
		return "↑/↓: Navigate  •  Enter: Continue  •  Q/Ctrl+C: Quit"
	case stepComplete:
		return "Enter: Exit  •  Q/Ctrl+C: Quit"
	default:
		return "Q/Ctrl+C: Cancel"
	}
}

func executeTask(index int, m *model) tea.Cmd {
	return func() tea.Msg {
		// Simulate work delay for visibility
		time.Sleep(200 * time.Millisecond)

		err := m.tasks[index].execute(m)

		if err != nil {
			return taskCompleteMsg{
				index:   index,
				success: false,
				error:   err.Error(),
			}
		}

		return taskCompleteMsg{
			index:   index,
			success: true,
		}
	}
}

// Task functions

func checkPrivileges(m *model) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("installer must be run with sudo or as root")
	}
	return nil
}

func stopDaemon(m *model) error {
	// Stop the daemon if it's running
	cmd := exec.Command("systemctl", "stop", "sysc-walls.service")
	return cmd.Run()
}

func buildBinaries(m *model) error {
	components := []string{"daemon", "display", "client"}

	for _, component := range components {
		// Build each component
		cmd := exec.Command("go", "build", "-o", component, fmt.Sprintf("./cmd/%s/", component))
		cmd.Dir = getProjectRoot()

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build failed for %s: %s", component, string(output))
		}
	}
	return nil
}

func installBinaries(m *model) error {
	projectRoot := getProjectRoot()
	components := []string{"daemon", "display", "client"}

	// Stop the daemon if it's running to avoid "text file busy" error
	fmt.Println("Stopping sysc-walls service if running...")
	stopCmd := exec.Command("systemctl", "stop", "sysc-walls.service")
	stopCmd.Run() // Ignore errors - service might not be running

	for _, component := range components {
		srcPath := filepath.Join(projectRoot, component)
		dstPath := fmt.Sprintf("/usr/local/bin/sysc-walls-%s", component)

		// Read the source file
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read binary %s: %v", component, err)
		}

		// Remove existing file first (if it exists) to avoid busy file error
		if _, err := os.Stat(dstPath); err == nil {
			if err := os.Remove(dstPath); err != nil {
				return fmt.Errorf("failed to remove existing binary %s: %v", component, err)
			}
		}

		// Write to destination
		err = os.WriteFile(dstPath, data, 0755)
		if err != nil {
			return fmt.Errorf("failed to install binary %s: %v", component, err)
		}
	}

	return nil
}

func installSystemdService(m *model) error {
	projectRoot := getProjectRoot()
	srcPath := filepath.Join(projectRoot, "systemd", "sysc-walls.service")
	dstPath := "/etc/systemd/system/sysc-walls.service"

	// Read the source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read systemd service file: %v", err)
	}

	// Write to destination
	err = os.WriteFile(dstPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to install systemd service: %v", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	return cmd.Run()
}

func enableSystemdService(m *model) error {
	cmd := exec.Command("systemctl", "enable", "sysc-walls.service")
	return cmd.Run()
}

func removeBinaries(m *model) error {
	components := []string{"daemon", "display", "client"}

	for _, component := range components {
		path := fmt.Sprintf("/usr/local/bin/sysc-walls-%s", component)
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove binary %s: %v", component, err)
		}
	}

	return nil
}

func removeSystemdService(m *model) error {
	// Stop and disable the service first
	cmd := exec.Command("systemctl", "stop", "sysc-walls.service")
	cmd.Run()

	cmd = exec.Command("systemctl", "disable", "sysc-walls.service")
	cmd.Run()

	// Remove the service file
	err := os.Remove("/etc/systemd/system/sysc-walls.service")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove systemd service: %v", err)
	}

	// Reload systemd
	cmd = exec.Command("systemctl", "daemon-reload")
	return cmd.Run()
}

func getProjectRoot() string {
	// Get the directory where the installer is located
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		return "."
	}

	// Go up from cmd/installer to project root
	root := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))

	// Check if go.mod exists to verify this is the project root
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return root
	}

	// Fallback: try to find go.mod by walking up from current directory
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "."
}

func main() {
	// Check if go is installed
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Println("Error: Go is not installed or not in PATH")
		fmt.Println("Please install Go from https://golang.org/dl/")
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
