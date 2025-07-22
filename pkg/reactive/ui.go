package reactive

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kdeps/kdeps/pkg/logging"
)

// ReactiveUI provides a reactive UI framework
type ReactiveUI struct {
	store        *Store[AppState]
	program      *tea.Program
	logger       *logging.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	subscription Subscription
}

// UIModel represents the reactive UI model
type UIModel struct {
	store    *Store[AppState]
	state    AppState
	spinner  spinner.Model
	progress progress.Model
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// UIMsg represents UI messages
type UIMsg struct {
	State AppState
}

// NewReactiveUI creates a new reactive UI
func NewReactiveUI(logger *logging.Logger) *ReactiveUI {
	ctx, cancel := context.WithCancel(context.Background())

	store := NewStore(InitialAppState(), AppReducer)
	store.Use(LoggingMiddleware[AppState]())

	ui := &ReactiveUI{
		store:  store,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	return ui
}

// Start starts the reactive UI
func (ui *ReactiveUI) Start() error {
	model := ui.createModel()
	ui.program = tea.NewProgram(model, tea.WithAltScreen(), tea.WithInputTTY())

	// Subscribe to state changes
	ui.subscription = ui.store.Subscribe(ui.ctx, ObserverFunc[AppState]{
		NextFunc: func(state AppState) {
			if ui.program != nil {
				ui.program.Send(UIMsg{State: state})
			}
		},
		ErrorFunc: func(err error) {
			ui.logger.Error("UI state error", "error", err)
		},
	})

	go func() {
		if _, err := ui.program.Run(); err != nil {
			ui.logger.Error("UI program error", "error", err)
		}
	}()

	return nil
}

// Stop stops the reactive UI
func (ui *ReactiveUI) Stop() {
	if ui.subscription != nil {
		ui.subscription.Unsubscribe()
	}
	if ui.program != nil {
		ui.program.Quit()
	}
	ui.cancel()
	ui.store.Close()
}

// GetStore returns the UI store
func (ui *ReactiveUI) GetStore() *Store[AppState] {
	return ui.store
}

// Dispatch dispatches an action to the store
func (ui *ReactiveUI) Dispatch(action Action) {
	ui.store.Dispatch(action)
}

// createModel creates the UI model
func (ui *ReactiveUI) createModel() UIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 50

	vp := viewport.New(80, 15)
	vp.SetContent("")

	return UIModel{
		store:    ui.store,
		state:    ui.store.GetState(),
		spinner:  s,
		progress: p,
		viewport: vp,
		width:    120,
		height:   40,
	}
}

// Bubble Tea implementation
func (m UIModel) Init() tea.Cmd {
	return tea.Batch(spinner.Tick)
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		m.viewport.Height = msg.Height - 20
		m.ready = true

	case UIMsg:
		m.state = msg.State
		m.updateViewport()

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

func (m UIModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Operations
	sections = append(sections, m.renderOperations())

	// Progress
	sections = append(sections, m.renderProgress())

	// Logs
	sections = append(sections, m.renderLogs())

	// Footer
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m UIModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		Width(m.width)

	title := "🚀 Kdeps Reactive System"
	if m.state.Loading {
		title += " " + m.spinner.View()
	}

	return titleStyle.Render(title)
}

func (m UIModel) renderOperations() string {
	if len(m.state.Operations) == 0 {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Render("No operations running")
	}

	listStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	var items []string
	items = append(items, lipgloss.NewStyle().Bold(true).Render("Operations:"))

	for _, op := range m.state.Operations {
		icon := m.getOperationIcon(op.Status)

		var style lipgloss.Style
		switch op.Status {
		case "completed":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
		case "running":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
		case "error":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		default:
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		}

		line := fmt.Sprintf("  %s %s [%s] %.0f%%", icon, op.ID, op.Type, op.Progress*100)
		items = append(items, style.Render(line))
	}

	return listStyle.Render(lipgloss.JoinVertical(lipgloss.Left, items...))
}

func (m UIModel) renderProgress() string {
	if len(m.state.Operations) == 0 {
		return ""
	}

	// Calculate overall progress
	totalProgress := 0.0
	for _, op := range m.state.Operations {
		totalProgress += op.Progress
	}
	overallProgress := totalProgress / float64(len(m.state.Operations))

	progressStyle := lipgloss.NewStyle().
		Padding(0, 2)

	progressText := fmt.Sprintf("Overall Progress: %.0f%%", overallProgress*100)
	m.progress.SetPercent(overallProgress)

	return progressStyle.Render(
		progressText + "\n" + m.progress.View(),
	)
}

func (m UIModel) renderLogs() string {
	logStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238"))

	title := lipgloss.NewStyle().Bold(true).Render("Logs:")

	if len(m.state.Logs) == 0 {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render("No logs yet...")
		return logStyle.Render(title + "\n" + content)
	}

	return logStyle.Render(title + "\n" + m.viewport.View())
}

func (m *UIModel) updateViewport() {
	if len(m.state.Logs) == 0 {
		return
	}

	var logLines []string
	for _, log := range m.state.Logs {
		timestamp := time.Unix(log.Timestamp, 0).Format("15:04:05")
		line := fmt.Sprintf("[%s] [%s] %s", timestamp, log.Level, log.Message)

		// Truncate long lines
		if len(line) > m.width-4 {
			line = line[:m.width-7] + "..."
		}

		logLines = append(logLines, line)
	}

	// Keep only last 100 lines
	if len(logLines) > 100 {
		logLines = logLines[len(logLines)-100:]
	}

	content := lipgloss.JoinVertical(lipgloss.Left, logLines...)
	m.viewport.SetContent(content)

	if m.viewport.Height > 0 && len(logLines) > 0 {
		m.viewport.GotoBottom()
	}
}

func (m UIModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		Width(m.width)

	status := "Ready"
	if m.state.Loading {
		status = "Loading..."
	}
	if m.state.Error != "" {
		status = fmt.Sprintf("Error: %s", m.state.Error)
	}

	footer := fmt.Sprintf("%s | ↑/↓: Scroll • Ctrl+C/q: Quit", status)
	return footerStyle.Render(footer)
}

func (m UIModel) getOperationIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "running":
		return "🔄"
	case "error":
		return "❌"
	default:
		return "⏳"
	}
}

// Helper functions for UI actions

// UIOperationStarted creates an action for when an operation starts
func UIOperationStarted(id, opType string) Action {
	op := Operation{
		ID:        id,
		Type:      opType,
		Status:    "running",
		Progress:  0.0,
		StartTime: time.Now().Unix(),
		Metadata:  map[string]interface{}{},
	}
	return AddOperation(op)
}

// UIOperationProgress creates an action for operation progress
func UIOperationProgress(id string, progress float64) Action {
	// This would need to be implemented in the reducer to find and update the operation
	return NewAction("UPDATE_OPERATION_PROGRESS", map[string]interface{}{
		"id":       id,
		"progress": progress,
	})
}

// UIOperationCompleted creates an action for when an operation completes
func UIOperationCompleted(id string, result interface{}) Action {
	return NewAction("COMPLETE_OPERATION", map[string]interface{}{
		"id":     id,
		"result": result,
	})
}

// UIOperationFailed creates an action for when an operation fails
func UIOperationFailed(id string, err error) Action {
	return NewAction("FAIL_OPERATION", map[string]interface{}{
		"id":    id,
		"error": err.Error(),
	})
}

// UILogMessage creates an action for logging a message
func UILogMessage(level, message, source string, data interface{}) Action {
	log := LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now().Unix(),
		Source:    source,
		Data:      data,
	}
	return AddLog(log)
}

// UISetLoading creates an action for setting loading state
func UISetLoading(loading bool) Action {
	return SetLoading(loading)
}

// UISetError creates an action for setting error state
func UISetError(err string) Action {
	return SetError(err)
}

// UIClearError creates an action for clearing error state
func UIClearError() Action {
	return ClearError()
}
