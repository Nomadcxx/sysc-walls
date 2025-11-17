package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Theme colors - RAMA theme
var (
	BgBase       = lipgloss.Color("#2b2d42")  // RAMA Space cadet
	Primary      = lipgloss.Color("#ef233c")  // RAMA Red Pantone
	Secondary    = lipgloss.Color("#d90429")  // RAMA Fire engine red
	Accent       = lipgloss.Color("#edf2f4")  // RAMA Anti-flash white
	FgPrimary    = lipgloss.Color("#edf2f4")  // RAMA Anti-flash white
	FgSecondary  = lipgloss.Color("#8d99ae")  // RAMA Cool gray
	FgMuted      = lipgloss.Color("#8d99ae")  // RAMA Cool gray
	ErrorColor   = lipgloss.Color("#d90429")  // RAMA Fire engine red
	WarningColor = lipgloss.Color("#ef233c")  // RAMA Red Pantone
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
	stepConfigPrompt
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
	step               installStep
	tasks              []installTask
	currentTaskIndex   int
	width              int
	height             int
	spinner            spinner.Model
	errors             []string
	uninstallMode      bool
	selectedOption     int  // 0 = Install, 1 = Uninstall
	configExists       bool // Whether config file already exists
	overrideConfig     bool // Whether to override existing config
	configPromptOption int  // 0 = Override, 1 = Keep existing
	binariesExist      bool // Whether binaries are already installed
}

type taskCompleteMsg struct {
	index   int
	success bool
	error   string
}

type taskProgressMsg struct {
	index       int
	description string
}

func newModel() model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(Secondary)
	s.Spinner = spinner.Dot

	// Check if binaries are already installed
	binariesExist := checkExistingBinaries()

	return model{
		step:             stepWelcome,
		currentTaskIndex: -1,
		spinner:          s,
		errors:           []string{},
		selectedOption:   0,
		binariesExist:    binariesExist,
	}
}

// checkExistingBinaries checks if sysc-walls binaries are already installed
func checkExistingBinaries() bool {
	components := []string{"daemon", "display", "client"}
	for _, component := range components {
		path := fmt.Sprintf("/usr/local/bin/sysc-walls-%s", component)
		if _, err := os.Stat(path); err != nil {
			return false // If any binary is missing, not fully installed
		}
	}
	return true
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
			if m.step == stepConfigPrompt && m.configPromptOption > 0 {
				m.configPromptOption--
			}
		case "down", "j":
			if m.step == stepWelcome && m.selectedOption < 1 {
				m.selectedOption++
			}
			if m.step == stepConfigPrompt && m.configPromptOption < 1 {
				m.configPromptOption++
			}
		case "enter":
			if m.step == stepWelcome {
				m.uninstallMode = m.selectedOption == 1

				// Check if config exists (only for install mode)
				if !m.uninstallMode {
					homeDir, err := os.UserHomeDir()
					if err == nil {
						configPath := filepath.Join(homeDir, ".config", "sysc-walls", "daemon.conf")
						if _, err := os.Stat(configPath); err == nil {
							m.configExists = true
							m.step = stepConfigPrompt
							m.configPromptOption = 1 // Default to "Keep existing"
							return m, nil
						}
					}
				}

				// No config exists or uninstall mode - proceed directly
				m.initTasks()
				m.step = stepInstalling
				m.currentTaskIndex = 0
				m.tasks[0].status = statusRunning
				return m, tea.Batch(
					m.spinner.Tick,
					executeTask(0, &m),
				)
			} else if m.step == stepConfigPrompt {
				// User has chosen whether to override config
				m.overrideConfig = m.configPromptOption == 0
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

	case taskProgressMsg:
		// Update task description with progress info
		if msg.index >= 0 && msg.index < len(m.tasks) {
			m.tasks[msg.index].description = msg.description
		}
		return m, nil

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
			{name: "Stop existing daemon", description: "Stopping existing sysc-walls daemon if running", execute: stopDaemon, status: statusPending, optional: true},
			{name: "Check sysc-Go", description: "Installing sysc-go animation library (AUR or go install)", execute: checkSyscGo, status: statusPending, optional: true},
			{name: "Build binaries", description: "Building sysc-walls components", execute: buildBinaries, status: statusPending},
			{name: "Install binaries", description: "Installing to /usr/local/bin", execute: installBinaries, status: statusPending},
			{name: "Update config", description: "Updating daemon configuration", execute: updateConfig, status: statusPending},
			{name: "Install systemd service", description: "Installing systemd service", execute: installSystemdService, status: statusPending},
			{name: "Import environment", description: "Importing Wayland environment for systemd", execute: importWaylandEnvironment, status: statusPending},
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
	headerLines := loadASCIIHeader()

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
	case stepConfigPrompt:
		mainContent = m.renderConfigPrompt()
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

	// Show installation status if binaries exist
	if m.binariesExist {
		statusStyle := lipgloss.NewStyle().Foreground(Accent).Bold(true)
		b.WriteString(statusStyle.Render("✓ sysc-walls is already installed"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("  Binaries found in /usr/local/bin"))
		b.WriteString("\n\n")
	}

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

func (m model) renderConfigPrompt() string {
	var b strings.Builder

	warningStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
	b.WriteString(warningStyle.Render("⚠ Existing Configuration Detected"))
	b.WriteString("\n\n")
	b.WriteString("An existing sysc-walls configuration file was found at:\n")
	b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("~/.config/sysc-walls/daemon.conf"))
	b.WriteString("\n\n")
	b.WriteString("What would you like to do?\n\n")

	// Override option
	overridePrefix := "  "
	if m.configPromptOption == 0 {
		overridePrefix = lipgloss.NewStyle().Foreground(Primary).Render("▸ ")
	}
	b.WriteString(overridePrefix + "Override with new default configuration\n")
	b.WriteString("    Your current config will be backed up to daemon.conf.backup\n\n")

	// Keep existing option
	keepPrefix := "  "
	if m.configPromptOption == 1 {
		keepPrefix = lipgloss.NewStyle().Foreground(Accent).Render("▸ ")
	}
	b.WriteString(keepPrefix + "Keep existing configuration\n")
	b.WriteString("    Your current settings will be preserved\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("Note: The installer will continue with your binaries update"))

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
		// Installation/Uninstallation failed
		failMsg := "Installation failed"
		if m.uninstallMode {
			failMsg = "Uninstallation failed"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(ErrorColor).Bold(true).Render(failMsg))
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

			// Installation summary
			b.WriteString(lipgloss.NewStyle().Foreground(Primary).Bold(true).Render("Installed Components:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("  • sysc-walls-daemon"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("  → /usr/local/bin/"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("  • sysc-walls-display"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render(" → /usr/local/bin/"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("  • sysc-walls-client"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("  → /usr/local/bin/"))
			b.WriteString("\n")

			// Get home directory for config path
			homeDir := os.Getenv("HOME")
			sudoUser := os.Getenv("SUDO_USER")
			if sudoUser != "" {
				homeDir = "/home/" + sudoUser
			}

			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("  • Configuration"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render(fmt.Sprintf("     → %s/.config/sysc-walls/", homeDir)))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("  • Systemd service"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render(fmt.Sprintf("   → %s/.config/systemd/user/", homeDir)))
			b.WriteString("\n\n")

			// Test first section
			b.WriteString(lipgloss.NewStyle().Foreground(Primary).Bold(true).Render("Test your installation:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("  sysc-walls-daemon -test"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("              # Quick test"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("  sysc-walls-daemon -test -debug"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("       # Test with diagnostics"))
			b.WriteString("\n\n")

			// Then start/restart service
			b.WriteString(lipgloss.NewStyle().Foreground(Primary).Bold(true).Render("Start the service:"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("  systemctl --user enable sysc-walls.service"))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(Accent).Render("  systemctl --user restart sysc-walls.service"))
			b.WriteString(lipgloss.NewStyle().Foreground(FgMuted).Render("  # Use restart to reload new binaries"))
			b.WriteString("\n\n")

			// Important note about upgrades
			b.WriteString(lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render("Note: "))
			b.WriteString(lipgloss.NewStyle().Foreground(FgSecondary).Render("If upgrading, the daemon must be restarted to use the new binaries."))
		}
	}

	return b.String()
}

func (m model) getHelpText() string {
	switch m.step {
	case stepWelcome:
		return "↑/↓: Navigate  •  Enter: Continue  •  Q/Ctrl+C: Quit"
	case stepConfigPrompt:
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
	sudoUser := os.Getenv("SUDO_USER")

	// Get actual user UID for XDG_RUNTIME_DIR
	actualUID := os.Getuid()
	if sudoUser != "" {
		cmd := exec.Command("id", "-u", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			if uid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
				actualUID = uid
			}
		}
	}

	// Stop the user daemon if it's running
	var cmd *exec.Cmd
	if sudoUser != "" {
		// Run as the actual user with proper environment
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "stop", "sysc-walls.service")
	} else {
		cmd = exec.Command("systemctl", "--user", "stop", "sysc-walls.service")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}

	// Try to stop the service (ignore errors - might not be installed/running)
	cmd.Run()

	// Give it a moment to stop gracefully
	time.Sleep(500 * time.Millisecond)

	// Note: We can't reliably detect if the daemon is still running on Linux
	// because os.WriteFile will succeed even if the binary is in use (it creates
	// a new inode). The user will be instructed to restart the service after installation.
	return nil
}

func checkSyscGo(m *model) error {
	// Check if sysc-go (syscgo) binary is already available
	if _, err := exec.LookPath("syscgo"); err == nil {
		// Already installed
		return nil
	}

	// Detect package manager and install sysc-go
	packageManager := detectPackageManager()

	switch packageManager {
	case "pacman":
		// Try AUR installation via yay/paru
		if _, err := exec.LookPath("yay"); err == nil {
			sudoUser := os.Getenv("SUDO_USER")
			var cmd *exec.Cmd
			if sudoUser != "" {
				// yay must NOT be run as root
				cmd = exec.Command("su", "-", sudoUser, "-c", "yay -S --noconfirm syscgo")
			} else {
				// If running without sudo, try directly
				cmd = exec.Command("yay", "-S", "--noconfirm", "syscgo")
			}
			if output, err := cmd.CombinedOutput(); err != nil {
				// yay failed, try go install as fallback
				return installSyscGoWithGoInstall()
			} else {
				_ = output // success
				return nil
			}
		} else if _, err := exec.LookPath("paru"); err == nil {
			sudoUser := os.Getenv("SUDO_USER")
			var cmd *exec.Cmd
			if sudoUser != "" {
				cmd = exec.Command("su", "-", sudoUser, "-c", "paru -S --noconfirm syscgo")
			} else {
				cmd = exec.Command("paru", "-S", "--noconfirm", "syscgo")
			}
			if output, err := cmd.CombinedOutput(); err != nil {
				return installSyscGoWithGoInstall()
			} else {
				_ = output
				return nil
			}
		} else {
			// No AUR helper, use go install
			return installSyscGoWithGoInstall()
		}

	default:
		// Non-Arch systems: use go install
		return installSyscGoWithGoInstall()
	}
}

func detectPackageManager() string {
	managers := map[string]string{
		"pacman": "/usr/bin/pacman",
		"apt":    "/usr/bin/apt",
		"dnf":    "/usr/bin/dnf",
	}

	for name, path := range managers {
		if _, err := os.Stat(path); err == nil {
			return name
		}
	}

	return "unknown"
}

func installSyscGoWithGoInstall() error {
	// Check if Go is available
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("sysc-go not found and Go is not installed - please install sysc-go manually via AUR or `go install github.com/Nomadcxx/sysc-Go/cmd/syscgo@latest`")
	}

	// Install via go install (run as original user, not root)
	sudoUser := os.Getenv("SUDO_USER")
	var cmd *exec.Cmd

	if sudoUser != "" {
		// Run go install as the original user so it installs to their GOPATH
		cmd = exec.Command("su", "-", sudoUser, "-c", "go install github.com/Nomadcxx/sysc-Go/cmd/syscgo@latest")
	} else {
		cmd = exec.Command("go", "install", "github.com/Nomadcxx/sysc-Go/cmd/syscgo@latest")
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install sysc-go via go install: %s", string(output))
	}

	return nil
}

func buildBinaries(m *model) error {
	components := []string{"daemon", "display", "client"}

	// Create bin directory if it doesn't exist
	if err := os.MkdirAll("bin", 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %v", err)
	}

	for _, component := range components {
		// Build each component to bin/ directory
		outputPath := filepath.Join("bin", component)
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", outputPath, fmt.Sprintf("./cmd/%s/", component))

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build failed for %s: %s", component, string(output))
		}

		// Validate binary was created
		if info, err := os.Stat(outputPath); err != nil {
			return fmt.Errorf("build validation failed - binary not found at %s after build: %v", outputPath, err)
		} else if info.Size() == 0 {
			return fmt.Errorf("build validation failed - binary at %s is empty (0 bytes)", outputPath)
		} else if info.Mode()&0111 == 0 {
			return fmt.Errorf("build validation failed - binary at %s is not executable", outputPath)
		}
	}
	return nil
}

func installBinaries(m *model) error {
	components := []string{"daemon", "display", "client"}

	// Note: daemon should already be stopped by stopDaemon task before this runs

	for _, component := range components {
		dstPath := fmt.Sprintf("/usr/local/bin/sysc-walls-%s", component)
		srcPath := filepath.Join("bin", component)

		// Read the source file from bin/ directory
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
			return fmt.Errorf("failed to install binary %s to %s: %v", component, dstPath, err)
		}

		// Validate binary was installed correctly
		if info, err := os.Stat(dstPath); err != nil {
			return fmt.Errorf("binary validation failed - %s not found after installation: %v", dstPath, err)
		} else {
			// Verify it's executable
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("binary validation failed - %s is not executable (mode: %v)", dstPath, info.Mode())
			}
			// Verify size matches source
			if info.Size() != int64(len(data)) {
				return fmt.Errorf("binary validation failed - %s size mismatch (expected %d, got %d)", dstPath, len(data), info.Size())
			}
		}
	}

	return nil
}

func updateConfig(m *model) error {
	// Get the actual user's home directory (not root when using sudo)
	var homeDir string
	sudoUser := os.Getenv("SUDO_USER")

	if sudoUser != "" {
		// Running with sudo - get actual user's home from SUDO_USER
		// Use getent to properly get home directory (handles non-standard home dirs)
		cmd := exec.Command("getent", "passwd", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			// Format: username:x:uid:gid:gecos:home:shell
			fields := strings.Split(strings.TrimSpace(string(output)), ":")
			if len(fields) >= 6 {
				homeDir = fields[5]
			}
		}
		// Fallback to /home/$SUDO_USER if getent fails
		if homeDir == "" {
			homeDir = "/home/" + sudoUser
		}
	} else {
		// Not running with sudo - use $HOME environment variable
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			return fmt.Errorf("HOME environment variable is not set")
		}
	}

	// Config file path
	configDir := filepath.Join(homeDir, ".config", "sysc-walls")
	configPath := filepath.Join(configDir, "daemon.conf")

	// Get actual user UID/GID for proper ownership
	var uid, gid int
	if sudoUser != "" {
		// Get UID
		cmd := exec.Command("id", "-u", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			uid, _ = strconv.Atoi(strings.TrimSpace(string(output)))
		}
		// Get GID
		cmd = exec.Command("id", "-g", sudoUser)
		output, err = cmd.Output()
		if err == nil {
			gid, _ = strconv.Atoi(strings.TrimSpace(string(output)))
		}
	}

	// Validate home directory path doesn't contain literal ~ or other issues
	if strings.Contains(homeDir, "~") {
		return fmt.Errorf("home directory contains literal tilde: %s - this should not happen", homeDir)
	}
	if !filepath.IsAbs(homeDir) {
		return fmt.Errorf("home directory is not absolute path: %s", homeDir)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %v", configDir, err)
	}

	// Verify directory was created
	if info, err := os.Stat(configDir); err != nil {
		return fmt.Errorf("config directory validation failed - %s does not exist after MkdirAll: %v", configDir, err)
	} else if !info.IsDir() {
		return fmt.Errorf("config path exists but is not a directory: %s", configDir)
	}

	// Set proper ownership on config directory
	if sudoUser != "" && uid > 0 {
		if err := os.Chown(configDir, uid, gid); err != nil {
			return fmt.Errorf("failed to set ownership on config directory: %v", err)
		}
		// Also chown parent .config directory if we created it
		parentConfig := filepath.Join(homeDir, ".config")
		os.Chown(parentConfig, uid, gid) // Ignore error as it may already exist
	}

	// Create ASCII art directory
	asciiDir := filepath.Join(configDir, "ascii")
	if err := os.MkdirAll(asciiDir, 0755); err != nil {
		return fmt.Errorf("failed to create ASCII art directory %s: %v", asciiDir, err)
	}

	// Set proper ownership on ASCII directory
	if sudoUser != "" && uid > 0 {
		if err := os.Chown(asciiDir, uid, gid); err != nil {
			return fmt.Errorf("failed to set ownership on ASCII directory: %v", err)
		}
	}

	// Copy bundled ASCII art files to user config directory
	asciiSourceDir := "assets/ascii"
	if info, err := os.Stat(asciiSourceDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(asciiSourceDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
					srcPath := filepath.Join(asciiSourceDir, entry.Name())
					dstPath := filepath.Join(asciiDir, entry.Name())

					// Copy file
					srcData, err := os.ReadFile(srcPath)
					if err == nil {
						if err := os.WriteFile(dstPath, srcData, 0644); err == nil {
							// Set proper ownership
							if sudoUser != "" && uid > 0 {
								os.Chown(dstPath, uid, gid)
							}
						}
					}
				}
			}
		}
	}

	// Default config content with new defaults
	defaultConfig := `# sysc-walls daemon configuration
# Configuration file for the sysc-walls screensaver daemon

[idle]
# timeout: How long to wait after last input before activating screensaver
#          Valid formats: 30s, 5m, 1h (s=seconds, m=minutes, h=hours)
#          Default: 5m
timeout = 5m

# min_duration: Minimum time screensaver runs before it can be deactivated
#               Prevents accidental dismissal from immediate mouse movement
#               Valid formats: 30s, 5m, 1h
#               Default: 30s
min_duration = 30s

[daemon]
# debug: Enable debug logging to stderr
#        Set to true to troubleshoot issues
#        Default: false
debug = false

[animation]
# effect: The animation/screensaver effect to display
#         Text-based effects: matrix-art, fire-text, rain-art
#         Non-text effects: matrix, fire, rain, aquarium, fireworks, beams
#         Default: matrix-art
effect = matrix-art

# theme: Color theme for the animation
#        Available themes: dracula, rama, monokai, nord, gruvbox, tokyo-night, catppuccin
#        Default: rama
theme = rama

# cycle: Automatically cycle through different effects
#        Set to true to rotate effects periodically
#        Default: false
cycle = false

# datetime: Show date and time overlay on screensaver
#           IMPORTANT: Only works with non-text effects (matrix, fire, rain, aquarium, etc.)
#                      Will not work with text-based effects (matrix-art, fire-text, etc.)
#           Default: false
datetime = false

# ASCII art files are stored in ~/.config/sysc-walls/ascii/
# Text-based effects (matrix-art, rain-art, fire-text) will automatically use SYSC.txt

[datetime]
# position: Where to display the date/time overlay on screen
#           Valid values: top, center, bottom
#           - top: Display near top of screen (2 lines from edge)
#           - center: Display in vertical center of screen
#           - bottom: Display near bottom of screen (2 lines from edge)
#           Default: bottom
position = bottom

[terminal]
# kitty: Use kitty terminal graphics protocol for enhanced rendering
#        Set to false if not using kitty terminal
#        Default: true
kitty = true

# fullscreen: Launch screensaver in fullscreen mode
#             Provides immersive screensaver experience
#             Default: true
fullscreen = true
`

	// Check if config file exists
	configFileExists := false
	if _, err := os.Stat(configPath); err == nil {
		configFileExists = true
	}

	// If config exists and user chose to keep it, skip writing new config
	if configFileExists && !m.overrideConfig {
		return nil
	}

	// If config exists and we're overriding, back it up first
	if configFileExists && m.overrideConfig {
		backupPath := configPath + ".backup"
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read existing config: %v", err)
		}

		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %v", err)
		}

		// Set proper ownership on backup file
		if sudoUser != "" && uid > 0 {
			os.Chown(backupPath, uid, gid)
		}
	}

	// Write new config
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %v", configPath, err)
	}

	// Set proper ownership on config file
	if sudoUser != "" && uid > 0 {
		if err := os.Chown(configPath, uid, gid); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %v", configPath, err)
		}
	}

	// Validate that config file was created correctly
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config validation failed - file not found at %s: %v", configPath, err)
	}

	// Validate config directory ownership
	if sudoUser != "" && uid > 0 {
		info, err := os.Stat(configPath)
		if err != nil {
			return fmt.Errorf("failed to validate config file: %v", err)
		}
		stat := info.Sys().(*syscall.Stat_t)
		if int(stat.Uid) != uid {
			return fmt.Errorf("config ownership validation failed - file at %s is owned by UID %d, expected %d", configPath, stat.Uid, uid)
		}
	}

	return nil
}

func importWaylandEnvironment(m *model) error {
	sudoUser := os.Getenv("SUDO_USER")

	// Get actual user UID for XDG_RUNTIME_DIR
	actualUID := os.Getuid()
	if sudoUser != "" {
		cmd := exec.Command("id", "-u", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			if uid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
				actualUID = uid
			}
		}
	}

	// Import WAYLAND_DISPLAY for systemd user services
	// This is critical for compositor detection to work
	var cmd *exec.Cmd
	if sudoUser != "" {
		// Run as the actual user with proper environment
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "import-environment", "WAYLAND_DISPLAY")
	} else {
		cmd = exec.Command("systemctl", "--user", "import-environment", "WAYLAND_DISPLAY")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}

	// Run the command, but don't fail if it doesn't work
	// (user might be on X11 or environment might be set already)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to import WAYLAND_DISPLAY for systemd: %v\n", err)
		fmt.Fprintf(os.Stderr, "Output: %s\n", string(output))
		fmt.Fprintf(os.Stderr, "This may affect compositor detection in the daemon\n")
	}

	return nil
}

func installSystemdService(m *model) error {
	srcPath := "systemd/sysc-walls-user.service"
	
	// Get the actual user's home directory and UID (not root when using sudo)
	homeDir := os.Getenv("HOME")
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		homeDir = "/home/" + sudoUser
	}
	
	// Create user systemd directory
	userSystemdDir := filepath.Join(homeDir, ".config", "systemd", "user")
	if err := os.MkdirAll(userSystemdDir, 0755); err != nil {
		return fmt.Errorf("failed to create user systemd directory: %v", err)
	}
	
	dstPath := filepath.Join(userSystemdDir, "sysc-walls.service")

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

	// Get actual user UID for systemctl commands
	actualUID := os.Getuid()
	if sudoUser != "" {
		// Get the UID of the sudo user
		cmd := exec.Command("id", "-u", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			if uid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
				actualUID = uid
			}
		}
	}

	// Reload user systemd as the actual user
	var cmd *exec.Cmd
	if sudoUser != "" {
		// Run as the actual user with proper environment
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "daemon-reload")
	} else {
		cmd = exec.Command("systemctl", "--user", "daemon-reload")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to reload systemd daemon: %v\n", err)
		fmt.Fprintf(os.Stderr, "Output: %s\n", string(output))
		fmt.Fprintf(os.Stderr, "You may need to run: systemctl --user daemon-reload\n")
	}

	return nil
}

func enableSystemdService(m *model) error {
	sudoUser := os.Getenv("SUDO_USER")

	// Get actual user UID for XDG_RUNTIME_DIR
	actualUID := os.Getuid()
	if sudoUser != "" {
		cmd := exec.Command("id", "-u", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			if uid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
				actualUID = uid
			}
		}
	}

	var cmd *exec.Cmd
	if sudoUser != "" {
		// Run as the actual user with proper environment
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "enable", "sysc-walls.service")
	} else {
		cmd = exec.Command("systemctl", "--user", "enable", "sysc-walls.service")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail, but warn user
		fmt.Fprintf(os.Stderr, "\nWarning: Failed to enable service automatically: %v\n", err)
		fmt.Fprintf(os.Stderr, "Output: %s\n", string(output))
		fmt.Fprintf(os.Stderr, "You may need to run manually:\n")
		fmt.Fprintf(os.Stderr, "  systemctl --user enable sysc-walls.service\n")
		fmt.Fprintf(os.Stderr, "  systemctl --user start sysc-walls.service\n\n")
	}

	return nil
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
	// Get the actual user's home directory (not root when using sudo)
	homeDir := os.Getenv("HOME")
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		homeDir = "/home/" + sudoUser
	}

	// Get actual user UID for XDG_RUNTIME_DIR
	actualUID := os.Getuid()
	if sudoUser != "" {
		cmd := exec.Command("id", "-u", sudoUser)
		output, err := cmd.Output()
		if err == nil {
			if uid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
				actualUID = uid
			}
		}
	}

	// Stop the user service first (ignore errors)
	var cmd *exec.Cmd
	if sudoUser != "" {
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "stop", "sysc-walls.service")
	} else {
		cmd = exec.Command("systemctl", "--user", "stop", "sysc-walls.service")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}
	cmd.Run()

	// Disable the user service (ignore errors)
	if sudoUser != "" {
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "disable", "sysc-walls.service")
	} else {
		cmd = exec.Command("systemctl", "--user", "disable", "sysc-walls.service")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}
	cmd.Run()

	// Remove the user service file
	servicePath := filepath.Join(homeDir, ".config", "systemd", "user", "sysc-walls.service")
	err := os.Remove(servicePath)
	if err != nil && !os.IsNotExist(err) {
		// Service file doesn't exist or we can't remove it - not critical, just log it
		fmt.Printf("Note: Could not remove service file at %s: %v\n", servicePath, err)
	}

	// Reload user systemd (ignore errors - might not be running)
	if sudoUser != "" {
		cmd = exec.Command("sudo", "-u", sudoUser, "env", fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID), "systemctl", "--user", "daemon-reload")
	} else {
		cmd = exec.Command("systemctl", "--user", "daemon-reload")
		cmd.Env = append(os.Environ(), fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/%d", actualUID))
	}
	cmd.Run()

	return nil
}

// loadASCIIHeader loads ASCII art from file or returns default
func loadASCIIHeader() []string {
	// Try to load from ascii.txt
	data, err := os.ReadFile("ascii.txt")
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		// Pad all lines to same width so lipgloss centering doesn't mangle them
		maxWidth := 0
		for _, line := range lines {
			if len([]rune(line)) > maxWidth {
				maxWidth = len([]rune(line))
			}
		}
		for i, line := range lines {
			lineRunes := []rune(line)
			if len(lineRunes) < maxWidth {
				// Pad with spaces on the right
				lines[i] = line + strings.Repeat(" ", maxWidth-len(lineRunes))
			}
		}
		// Add subtitle padded to same width
		subtitle := "       TERMINAL SCREENSAVER INSTALLER"
		subtitleRunes := []rune(subtitle)
		if len(subtitleRunes) < maxWidth {
			subtitle = subtitle + strings.Repeat(" ", maxWidth-len(subtitleRunes))
		}
		lines = append(lines, subtitle)
		return lines
	}

	// Fallback to embedded ASCII art - SYSCWALL
	return []string{
		" ▄▄▄▄▄▄▄ ▄▄    ▄▄   ▄▄▄▄▄▄▄  ▄▄▄▄▄▄▄     ▄▄ ▄▄    ▄▄  ▄▄▄▄▄▄  ▄▄        ▄▄      ",
		"██▀▀▀▀▀▀ ██▄  ▄██  ██▀▀▀▀▀▀ ██▀▀▀▀▀▀    ▄██ ██    ██ ██▀▀▀▀██ ██        ██      ",
		"▀██████▄  ▀████▀   ▀██████▄ ██        ▄██▀  ██▄██▄██ ██▄▄▄▄██ ██        ██      ",
		"▄▄▄▄▄▄██    ██     ▄▄▄▄▄▄██ ██▄▄▄▄▄▄ ██▀    ███▀▀███ ██▀▀▀▀██ ██▄▄▄▄▄▄  ██▄▄▄▄▄▄",
		"▀▀▀▀▀▀▀     ▀▀     ▀▀▀▀▀▀▀   ▀▀▀▀▀▀▀ ▀▀     ▀▀    ▀▀ ▀▀    ▀▀ ▀▀▀▀▀▀▀▀  ▀▀▀▀▀▀▀▀",
		"                    TERMINAL SCREENSAVER INSTALLER                               ",
	}
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
