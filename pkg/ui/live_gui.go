package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Operation represents a single operation in the workflow
type Operation struct {
	Name        string
	Description string
	Status      OperationStatus
	Progress    float64
	StartTime   time.Time
	EndTime     time.Time
	Logs        []string
}

// OperationStatus represents the status of an operation
type OperationStatus int

const (
	StatusPending OperationStatus = iota
	StatusRunning
	StatusCompleted
	StatusError
)

// LiveGUIModel represents the live CLI GUI model
type LiveGUIModel struct {
	// UI Components
	spinner  spinner.Model
	progress progress.Model
	viewport viewport.Model

	// State
	operations     []Operation
	currentOp      int
	globalProgress float64
	done           bool
	success        bool
	err            error
	completionTime time.Time
	containerStats *ContainerStats

	// Logs
	dockerLogs []string
	maxLogs    int

	// Styling
	width  int
	height int

	// Context
	ctx context.Context
}

// OperationMsg represents an operation update message
type OperationMsg struct {
	OperationIndex int
	Status         OperationStatus
	Description    string
	Progress       float64
	Error          error
}

// LogMsg represents a log message
type LogMsg struct {
	Message string
	IsError bool
}

// RouteInfo holds information about a specific route
type RouteInfo struct {
	Path       string
	Methods    []string
	ServerType string // "api", "static", "app"
	ActionID   string // for API routes
	AppPort    string // for app routes
}

// ContainerStats holds information about the built/running container
type ContainerStats struct {
	ImageName     string
	ImageVersion  string
	ContainerID   string
	APIServerMode bool
	WebServerMode bool
	HostIP        string
	HostPort      string
	WebHostIP     string
	WebHostPort   string
	GPUType       string
	Command       string // build, run, package
	Routes        []RouteInfo
}

// CompletionMsg represents completion of all operations
type CompletionMsg struct {
	Success        bool
	Error          error
	ContainerStats *ContainerStats
}

// NewLiveGUI creates a new live GUI instance
func NewLiveGUI(ctx context.Context, operationNames []string) LiveGUIModel {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize progress bar
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 50

	// Initialize viewport for logs
	vp := viewport.New(80, 15)
	vp.SetContent("")

	// Create operations
	operations := make([]Operation, len(operationNames))
	for i, name := range operationNames {
		operations[i] = Operation{
			Name:        name,
			Description: fmt.Sprintf("Preparing %s...", name),
			Status:      StatusPending,
			Progress:    0.0,
			Logs:        []string{},
		}
	}

	return LiveGUIModel{
		spinner:        s,
		progress:       p,
		viewport:       vp,
		operations:     operations,
		currentOp:      0,
		globalProgress: 0.0,
		dockerLogs:     []string{},
		maxLogs:        100,
		width:          120,
		height:         40,
		ctx:            ctx,
	}
}

// Init initializes the model
func (m LiveGUIModel) Init() tea.Cmd {
	return tea.Batch(spinner.Tick, m.waitForActivity())
}

// Update handles model updates
func (m LiveGUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If done, any key quits
		if m.done {
			return m, tea.Quit
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			m.viewport.LineUp(1)
		case "down":
			m.viewport.LineDown(1)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 20 // Leave space for header and footer

	case OperationMsg:
		if msg.OperationIndex >= 0 && msg.OperationIndex < len(m.operations) {
			op := &m.operations[msg.OperationIndex]
			op.Status = msg.Status
			op.Description = msg.Description
			op.Progress = msg.Progress

			if msg.Status == StatusRunning && op.StartTime.IsZero() {
				op.StartTime = time.Now()
				m.currentOp = msg.OperationIndex
			}

			if msg.Status == StatusCompleted || msg.Status == StatusError {
				op.EndTime = time.Now()
				if msg.Status == StatusError {
					m.err = msg.Error
				}
			}

			// Update global progress
			m.updateGlobalProgress()

			// Update progress bar with new value
			progressCmd := m.progress.SetPercent(m.globalProgress)
			return m, tea.Batch(m.waitForActivity(), progressCmd)
		}
		return m, m.waitForActivity()

	case LogMsg:
		// Add log to Docker logs
		m.dockerLogs = append(m.dockerLogs, msg.Message)

		// Trim logs if too many
		if len(m.dockerLogs) > m.maxLogs {
			m.dockerLogs = m.dockerLogs[len(m.dockerLogs)-m.maxLogs:]
		}

		// Update viewport content
		m.updateViewport()
		return m, m.waitForActivity()

	case CompletionMsg:
		m.done = true
		m.success = msg.Success
		m.completionTime = time.Now()
		m.containerStats = msg.ContainerStats
		if msg.Error != nil {
			m.err = msg.Error
		}
		// Don't quit immediately - wait for user input
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		newProgressModel, cmd := m.progress.Update(msg)
		m.progress = newProgressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// View renders the GUI
func (m LiveGUIModel) View() string {
	if m.done {
		return m.renderCompletion()
	}

	return m.renderProgress()
}

// renderProgress renders the main progress view
func (m LiveGUIModel) renderProgress() string {
	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Current operation
	sections = append(sections, m.renderCurrentOperation())

	// Progress bar
	sections = append(sections, m.renderProgressBar())

	// Operations list
	sections = append(sections, m.renderOperationsList())

	// Logs section
	sections = append(sections, m.renderLogs())

	// Footer
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header section
func (m LiveGUIModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		Width(m.width)

	return titleStyle.Render("🚀 Kdeps Modern CLI - AI Agent Operations")
}

// renderCurrentOperation renders the current operation
func (m LiveGUIModel) renderCurrentOperation() string {
	if m.currentOp >= len(m.operations) {
		return ""
	}

	op := m.operations[m.currentOp]

	spinnerStr := ""
	if op.Status == StatusRunning {
		spinnerStr = m.spinner.View()
	}

	statusIcon := m.getStatusIcon(op.Status)

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Padding(1, 2)

	content := fmt.Sprintf("%s %s %s\n%s",
		spinnerStr, statusIcon, op.Name, op.Description)

	return currentStyle.Render(content)
}

// renderProgressBar renders the global progress bar
func (m LiveGUIModel) renderProgressBar() string {
	progressStyle := lipgloss.NewStyle().
		Padding(0, 2)

	progressText := fmt.Sprintf("Overall Progress: %.0f%%", m.globalProgress*100)

	return progressStyle.Render(
		progressText + "\n" + m.progress.View(),
	)
}

// renderOperationsList renders the list of all operations
func (m LiveGUIModel) renderOperationsList() string {
	listStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	var items []string
	items = append(items, lipgloss.NewStyle().Bold(true).Render("Operations:"))

	for _, op := range m.operations {
		icon := m.getStatusIcon(op.Status)

		var style lipgloss.Style
		switch op.Status {
		case StatusCompleted:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
		case StatusRunning:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
		case StatusError:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		default:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		}

		line := fmt.Sprintf("  %s %s", icon, op.Name)

		// Add timing info for completed operations
		if op.Status == StatusCompleted && !op.StartTime.IsZero() && !op.EndTime.IsZero() {
			duration := op.EndTime.Sub(op.StartTime)
			line += fmt.Sprintf(" (%s)", duration.Round(time.Millisecond))
		}

		items = append(items, style.Render(line))
	}

	return listStyle.Render(strings.Join(items, "\n"))
}

// renderLogs renders the logs section
func (m LiveGUIModel) renderLogs() string {
	logStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238"))

	title := lipgloss.NewStyle().Bold(true).Render("Live Output:")

	if len(m.dockerLogs) == 0 {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render("Waiting for output...")
		return logStyle.Render(title + "\n" + content)
	}

	return logStyle.Render(title + "\n" + m.viewport.View())
}

// renderLogsCompletion renders logs for the completion screen
func (m LiveGUIModel) renderLogsCompletion() string {
	logStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Height(10) // Fixed height for completion screen

	title := lipgloss.NewStyle().Bold(true).Render("Build Output Summary:")

	// Show last 20 lines of logs for completion screen
	maxLines := 20
	startIdx := 0
	if len(m.dockerLogs) > maxLines {
		startIdx = len(m.dockerLogs) - maxLines
	}

	var logsToShow []string
	for i := startIdx; i < len(m.dockerLogs); i++ {
		log := m.dockerLogs[i]
		// Truncate long lines
		if len(log) > m.width-8 {
			log = log[:m.width-11] + "..."
		}
		logsToShow = append(logsToShow, log)
	}

	if len(logsToShow) == 0 {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render("No build output captured")
		return logStyle.Render(title + "\n" + content)
	}

	content := strings.Join(logsToShow, "\n")
	return logStyle.Render(title + "\n" + content)
}

// renderContainerStats renders container statistics for the completion screen
func (m LiveGUIModel) renderContainerStats() string {
	statsStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39"))

	title := lipgloss.NewStyle().Bold(true).Render("📦 Container Details:")

	stats := m.containerStats
	var details []string

	// Image information or package path
	if stats.Command == "package" {
		if stats.ImageName != "" {
			details = append(details, fmt.Sprintf("📦 Package: %s", stats.ImageName))
		}
	} else if stats.ImageName != "" {
		if stats.ImageVersion != "" {
			details = append(details, fmt.Sprintf("🐳 Image: %s", stats.ImageVersion))
		} else {
			details = append(details, fmt.Sprintf("🐳 Image: %s", stats.ImageName))
		}
	}

	// Container ID (for run command)
	if stats.ContainerID != "" {
		details = append(details, fmt.Sprintf("📋 Container ID: %s", stats.ContainerID[:12]))
	}

	// Command type
	if stats.Command != "" {
		details = append(details, fmt.Sprintf("⚙️  Command: kdeps %s", stats.Command))
	}

	// GPU information
	if stats.GPUType != "" && stats.GPUType != "none" {
		details = append(details, fmt.Sprintf("🎮 GPU: %s", stats.GPUType))
	}

	// Server configurations and ports
	if stats.APIServerMode {
		if stats.HostIP != "" && stats.HostPort != "" {
			details = append(details, fmt.Sprintf("🌐 API Server: http://%s:%s", stats.HostIP, stats.HostPort))
		} else {
			details = append(details, "🌐 API Server: Enabled")
		}
	}

	if stats.WebServerMode {
		if stats.WebHostIP != "" && stats.WebHostPort != "" {
			details = append(details, fmt.Sprintf("🌍 Web Server: http://%s:%s", stats.WebHostIP, stats.WebHostPort))
		} else {
			details = append(details, "🌍 Web Server: Enabled")
		}
	}

	// If no specific server info, show general access info
	if !stats.APIServerMode && !stats.WebServerMode && stats.HostIP != "" && stats.HostPort != "" {
		details = append(details, fmt.Sprintf("🔗 Access: http://%s:%s", stats.HostIP, stats.HostPort))
	}

	// Route information - only show routes for enabled server modes
	var apiRoutes []RouteInfo
	var webRoutes []RouteInfo
	
	for _, route := range stats.Routes {
		if route.ServerType == "api" && stats.APIServerMode {
			apiRoutes = append(apiRoutes, route)
		} else if (route.ServerType == "static" || route.ServerType == "app") && stats.WebServerMode {
			webRoutes = append(webRoutes, route)
		}
	}
	
	// Display API routes if API server is enabled
	if len(apiRoutes) > 0 && stats.APIServerMode {
		details = append(details, "")
		details = append(details, "🌐 API Routes:")
		for _, route := range apiRoutes {
			methodsStr := strings.Join(route.Methods, ", ")
			if stats.HostIP != "" && stats.HostPort != "" {
				routeLine := fmt.Sprintf("   • %s [%s] → http://%s:%s%s", route.Path, methodsStr, stats.HostIP, stats.HostPort, route.Path)
				details = append(details, routeLine)
			} else {
				routeLine := fmt.Sprintf("   • %s [%s]", route.Path, methodsStr)
				details = append(details, routeLine)
			}
		}
	}
	
	// Display Web routes if Web server is enabled
	if len(webRoutes) > 0 && stats.WebServerMode {
		details = append(details, "")
		details = append(details, "🌍 Web Routes:")
		for _, route := range webRoutes {
			var routeLine string
			
			switch route.ServerType {
			case "static":
				if stats.WebHostIP != "" && stats.WebHostPort != "" {
					routeLine = fmt.Sprintf("   • %s [Static] → http://%s:%s%s", route.Path, stats.WebHostIP, stats.WebHostPort, route.Path)
				} else {
					routeLine = fmt.Sprintf("   • %s [Static]", route.Path)
				}
			case "app":
				if stats.WebHostIP != "" && stats.WebHostPort != "" {
					routeLine = fmt.Sprintf("   • %s [App] → http://%s:%s%s", route.Path, stats.WebHostIP, stats.WebHostPort, route.Path)
				} else {
					routeLine = fmt.Sprintf("   • %s [App]", route.Path)
				}
				if route.AppPort != "" {
					routeLine += fmt.Sprintf(" (Port: %s)", route.AppPort)
				}
			default:
				routeLine = fmt.Sprintf("   • %s", route.Path)
			}
			
			details = append(details, routeLine)
		}
	}

	if len(details) == 0 {
		details = append(details, "No container details available")
	}

	content := strings.Join(details, "\n")
	return statsStyle.Render(title + "\n" + content)
}

// renderFooter renders the footer with controls
func (m LiveGUIModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		Width(m.width)

	return footerStyle.Render("↑/↓: Scroll logs • Ctrl+C: Cancel • q: Quit")
}

// renderCompletion renders the completion screen
func (m LiveGUIModel) renderCompletion() string {
	var sections []string

	if m.success {
		// Success screen
		successStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")).
			Background(lipgloss.Color("235")).
			Padding(2, 4).
			Margin(2, 0).
			Width(m.width)

		sections = append(sections, successStyle.Render("✅ Operation Completed Successfully!"))

		// Show summary
		summaryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Padding(1, 2)

		totalTime := m.completionTime.Sub(m.operations[0].StartTime)
		summary := fmt.Sprintf("🎉 All operations completed in %s", totalTime.Round(time.Millisecond))
		sections = append(sections, summaryStyle.Render(summary))

	} else {
		// Error screen
		errorStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("235")).
			Padding(2, 4).
			Margin(2, 0).
			Width(m.width)

		sections = append(sections, errorStyle.Render("❌ Operation Failed"))

		if m.err != nil {
			errorDetailStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("203")).
				Padding(1, 2)

			sections = append(sections, errorDetailStyle.Render(fmt.Sprintf("Error: %s", m.err.Error())))
		}
	}

	// Operations summary
	sections = append(sections, m.renderOperationsList())

	// Show container stats if available
	if m.containerStats != nil {
		sections = append(sections, m.renderContainerStats())
	}

	// Show logs if there are any
	if len(m.dockerLogs) > 0 {
		sections = append(sections, m.renderLogsCompletion())
	}

	// Add instruction to continue
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Background(lipgloss.Color("235")).
		Padding(1, 2).
		Width(m.width).
		Align(lipgloss.Center)

	sections = append(sections, instructionStyle.Render("Press any key to continue..."))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// Helper methods

func (m *LiveGUIModel) updateGlobalProgress() {
	if len(m.operations) == 0 {
		return
	}

	totalProgress := 0.0
	for _, op := range m.operations {
		switch op.Status {
		case StatusCompleted:
			totalProgress += 1.0
		case StatusRunning:
			totalProgress += op.Progress
		}
	}

	m.globalProgress = totalProgress / float64(len(m.operations))
}

func (m *LiveGUIModel) updateViewport() {
	// Process logs to handle long lines and improve readability
	var processedLogs []string
	for _, log := range m.dockerLogs {
		// Truncate very long lines to prevent viewport issues
		if len(log) > m.width-4 {
			log = log[:m.width-7] + "..."
		}

		// Skip empty lines
		if strings.TrimSpace(log) == "" {
			continue
		}

		processedLogs = append(processedLogs, log)
	}

	// Limit the number of lines to prevent memory issues
	maxLines := 1000
	if len(processedLogs) > maxLines {
		processedLogs = processedLogs[len(processedLogs)-maxLines:]
	}

	content := strings.Join(processedLogs, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m LiveGUIModel) getStatusIcon(status OperationStatus) string {
	switch status {
	case StatusCompleted:
		return "✅"
	case StatusRunning:
		return "🔄"
	case StatusError:
		return "❌"
	default:
		return "⏳"
	}
}

func (m LiveGUIModel) waitForActivity() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
}

// GUI Controller

// GUIControllerInterface defines the interface for GUI controllers
type GUIControllerInterface interface {
	Start() error
	Wait()
	UpdateOperation(index int, status OperationStatus, description string, progress float64)
	UpdateOperationError(index int, err error)
	AddLog(message string, isError bool)
	Complete(success bool, err error)
	CompleteWithStats(success bool, err error, stats *ContainerStats)
	Stop()
}

// GUIController manages the modern GUI
// It implements GUIControllerInterface
type GUIController struct {
	program *tea.Program
	model   *LiveGUIModel
	done    chan struct{}
}

// Compile-time interface compliance check
var _ GUIControllerInterface = (*GUIController)(nil)

// NewGUIController creates a new GUI controller
func NewGUIController(ctx context.Context, operations []string) *GUIController {
	model := NewLiveGUI(ctx, operations)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInputTTY())

	return &GUIController{
		program: program,
		model:   &model,
		done:    make(chan struct{}),
	}
}

// Start starts the GUI
func (gc *GUIController) Start() error {
	go func() {
		if _, err := gc.program.Run(); err != nil {
			// Handle error silently for now
		}
		close(gc.done)
	}()
	return nil
}

// Wait waits for the GUI to finish (blocking call)
func (gc *GUIController) Wait() {
	<-gc.done
}

// UpdateOperation updates an operation's status
func (gc *GUIController) UpdateOperation(index int, status OperationStatus, description string, progress float64) {
	gc.program.Send(OperationMsg{
		OperationIndex: index,
		Status:         status,
		Description:    description,
		Progress:       progress,
	})
}

// UpdateOperationError updates an operation with an error
func (gc *GUIController) UpdateOperationError(index int, err error) {
	gc.program.Send(OperationMsg{
		OperationIndex: index,
		Status:         StatusError,
		Description:    err.Error(),
		Error:          err,
	})
}

// AddLog adds a log message
func (gc *GUIController) AddLog(message string, isError bool) {
	gc.program.Send(LogMsg{
		Message: message,
		IsError: isError,
	})
}

// Complete marks all operations as complete
func (gc *GUIController) Complete(success bool, err error) {
	gc.program.Send(CompletionMsg{
		Success: success,
		Error:   err,
	})
}

// CompleteWithStats marks all operations as complete with container statistics
func (gc *GUIController) CompleteWithStats(success bool, err error, stats *ContainerStats) {
	gc.program.Send(CompletionMsg{
		Success:        success,
		Error:          err,
		ContainerStats: stats,
	})
}

// Stop stops the GUI
func (gc *GUIController) Stop() {
	gc.program.Quit()
}

// CreateGUILogger creates a logger that outputs to the GUI instead of stdout/stderr
func (gc *GUIController) CreateGUILogger() *GUILogger {
	return &GUILogger{gui: gc}
}

// GUILogger is a logger that redirects output to the GUI
type GUILogger struct {
	gui *GUIController
}

// Info logs an info message to the GUI
func (gl *GUILogger) Info(msg string, keysAndValues ...interface{}) {
	gl.gui.AddLog(fmt.Sprintf("ℹ️  %s", msg), false)
}

// Debug logs a debug message to the GUI
func (gl *GUILogger) Debug(msg string, keysAndValues ...interface{}) {
	// Skip debug messages to reduce noise
}

// Error logs an error message to the GUI
func (gl *GUILogger) Error(msg string, keysAndValues ...interface{}) {
	gl.gui.AddLog(fmt.Sprintf("🔴 %s", msg), true)
}

// Warn logs a warning message to the GUI
func (gl *GUILogger) Warn(msg string, keysAndValues ...interface{}) {
	gl.gui.AddLog(fmt.Sprintf("⚠️  %s", msg), false)
}

// Errorf logs a formatted error message to the GUI
func (gl *GUILogger) Errorf(format string, args ...interface{}) {
	gl.gui.AddLog(fmt.Sprintf("🔴 "+format, args...), true)
}
