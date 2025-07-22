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
	"github.com/kdeps/kdeps/pkg/reactive"
)

// ReactiveGUI replaces LiveGUIModel with a purely reactive interface
type ReactiveGUI struct {
	// Core reactive system
	reactiveUI   *reactive.ReactiveUI
	program      *tea.Program
	model        ReactiveGUIModel
	ctx          context.Context
	cancel       context.CancelFunc
	subscription reactive.Subscription
}

// ReactiveGUIModel implements the Bubble Tea model with reactive state
type ReactiveGUIModel struct {
	// UI Components
	spinner  spinner.Model
	progress progress.Model
	viewport viewport.Model

	// Reactive state
	state reactive.AppState
	store *reactive.Store[reactive.AppState]

	// Display state
	width  int
	height int
	ready  bool

	// Context
	ctx context.Context
}

// ReactiveStateMsg represents reactive state updates
type ReactiveStateMsg struct {
	State reactive.AppState
}

// NewReactiveGUI creates a new reactive GUI
func NewReactiveGUI(ctx context.Context, operations []string, logger interface{}) *ReactiveGUI {
	ctx, cancel := context.WithCancel(ctx)

	// Create reactive UI
	reactiveUI := reactive.NewReactiveUI(nil) // logger would be typed properly in real implementation

	// Initialize operations in the store
	for _, opName := range operations {
		reactiveUI.Dispatch(reactive.UIOperationStarted(opName, "initialization"))
	}

	gui := &ReactiveGUI{
		reactiveUI: reactiveUI,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Create the model
	gui.model = gui.createModel()

	return gui
}

// createModel creates the reactive GUI model
func (rg *ReactiveGUI) createModel() ReactiveGUIModel {
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

	store := rg.reactiveUI.GetStore()

	model := ReactiveGUIModel{
		spinner:  s,
		progress: p,
		viewport: vp,
		state:    store.GetState(),
		store:    store,
		width:    120,
		height:   40,
		ctx:      rg.ctx,
	}

	return model
}

// Start starts the reactive GUI
func (rg *ReactiveGUI) Start() error {
	// Start the reactive UI
	if err := rg.reactiveUI.Start(); err != nil {
		return err
	}

	// Subscribe to state changes
	rg.subscription = rg.model.store.Subscribe(rg.ctx, reactive.ObserverFunc[reactive.AppState]{
		NextFunc: func(state reactive.AppState) {
			if rg.program != nil {
				rg.program.Send(ReactiveStateMsg{State: state})
			}
		},
		ErrorFunc: func(err error) {
			// Handle state errors
		},
		CompleteFunc: func() {
			// Handle state completion
		},
	})

	// Start the Bubble Tea program
	rg.program = tea.NewProgram(rg.model, tea.WithAltScreen(), tea.WithInputTTY())

	go func() {
		if _, err := rg.program.Run(); err != nil {
			// Handle program error
		}
	}()

	return nil
}

// Stop stops the reactive GUI
func (rg *ReactiveGUI) Stop() {
	if rg.subscription != nil {
		rg.subscription.Unsubscribe()
	}
	if rg.program != nil {
		rg.program.Quit()
	}
	if rg.reactiveUI != nil {
		rg.reactiveUI.Stop()
	}
	rg.cancel()
}

// GetStore returns the reactive store
func (rg *ReactiveGUI) GetStore() *reactive.Store[reactive.AppState] {
	return rg.reactiveUI.GetStore()
}

// Dispatch dispatches an action to the reactive store
func (rg *ReactiveGUI) Dispatch(action reactive.Action) {
	rg.reactiveUI.Dispatch(action)
}

// UpdateOperation updates an operation via reactive dispatch
func (rg *ReactiveGUI) UpdateOperation(id string, status OperationStatus, description string, progress float64) {
	action := reactive.UIOperationProgress(id, progress)
	if progress >= 1.0 {
		action = reactive.UIOperationCompleted(id, "Operation completed")
	}

	rg.Dispatch(action)
}

// AddLog adds a log message via reactive dispatch
func (rg *ReactiveGUI) AddLog(message string, isError bool) {
	level := "info"
	if isError {
		level = "error"
	}

	action := reactive.UILogMessage(level, message, "gui", nil)
	rg.Dispatch(action)
}

// Complete marks the GUI as complete
func (rg *ReactiveGUI) Complete(success bool, err error) {
	if err != nil {
		rg.Dispatch(reactive.UISetError(err.Error()))
	}

	rg.Dispatch(reactive.UISetLoading(false))
}

// CompleteWithStats marks the GUI as complete with statistics
func (rg *ReactiveGUI) CompleteWithStats(success bool, err error, stats *ContainerStats) {
	rg.Complete(success, err)
	// Could add stats to reactive state here
}

// Wait waits for the GUI to complete
func (rg *ReactiveGUI) Wait() {
	<-rg.ctx.Done()
}

// Bubble Tea model implementation for ReactiveGUIModel

func (m ReactiveGUIModel) Init() tea.Cmd {
	return tea.Batch(spinner.Tick)
}

func (m ReactiveGUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case ReactiveStateMsg:
		m.state = msg.State
		m.updateViewportFromState()

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

func (m ReactiveGUIModel) View() string {
	if !m.ready {
		return "\n  Initializing reactive system..."
	}

	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Operations from reactive state
	sections = append(sections, m.renderOperationsFromState())

	// Progress from reactive state
	sections = append(sections, m.renderProgressFromState())

	// Logs from reactive state
	sections = append(sections, m.renderLogsFromState())

	// Footer
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// Rendering methods that use reactive state

func (m ReactiveGUIModel) renderHeader() string {
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

func (m ReactiveGUIModel) renderOperationsFromState() string {
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

		// Add timing info for completed operations
		if op.Status == "completed" && op.StartTime > 0 && op.EndTime > 0 {
			duration := time.Unix(op.EndTime, 0).Sub(time.Unix(op.StartTime, 0))
			line += fmt.Sprintf(" (%s)", duration.Round(time.Millisecond))
		}

		items = append(items, style.Render(line))
	}

	return listStyle.Render(lipgloss.JoinVertical(lipgloss.Left, items...))
}

func (m ReactiveGUIModel) renderProgressFromState() string {
	if len(m.state.Operations) == 0 {
		return ""
	}

	// Calculate overall progress from reactive state
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

func (m ReactiveGUIModel) renderLogsFromState() string {
	logStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238"))

	title := lipgloss.NewStyle().Bold(true).Render("Live Output:")

	if len(m.state.Logs) == 0 {
		content := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render("Waiting for output...")
		return logStyle.Render(title + "\n" + content)
	}

	return logStyle.Render(title + "\n" + m.viewport.View())
}

func (m *ReactiveGUIModel) updateViewportFromState() {
	if len(m.state.Logs) == 0 {
		return
	}

	var logLines []string
	for _, log := range m.state.Logs {
		timestamp := time.Unix(log.Timestamp, 0).Format("15:04:05")
		line := fmt.Sprintf("[%s] [%s] %s", timestamp, log.Level, log.Message)

		if log.Source != "" {
			line = fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, log.Level, log.Source, log.Message)
		}

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

	content := strings.Join(logLines, "\n")
	m.viewport.SetContent(content)

	if m.viewport.Height > 0 && len(logLines) > 0 {
		m.viewport.GotoBottom()
	}
}

func (m ReactiveGUIModel) renderFooter() string {
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

func (m ReactiveGUIModel) getOperationIcon(status string) string {
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

// Interface compatibility - implement methods expected by the system

// ReactiveGUIController implements the GUIControllerInterface using reactive patterns
type ReactiveGUIController struct {
	gui *ReactiveGUI
}

// NewReactiveGUIController creates a new reactive GUI controller
func NewReactiveGUIController(ctx context.Context, operations []string) *ReactiveGUIController {
	gui := NewReactiveGUI(ctx, operations, nil)
	return &ReactiveGUIController{gui: gui}
}

func (rgc *ReactiveGUIController) Start() error {
	return rgc.gui.Start()
}

func (rgc *ReactiveGUIController) Wait() {
	rgc.gui.Wait()
}

func (rgc *ReactiveGUIController) UpdateOperation(index int, status OperationStatus, description string, progress float64) {
	// Convert index to ID (would need better mapping in real implementation)
	id := fmt.Sprintf("operation-%d", index)
	rgc.gui.UpdateOperation(id, status, description, progress)
}

func (rgc *ReactiveGUIController) UpdateOperationError(index int, err error) {
	id := fmt.Sprintf("operation-%d", index)
	rgc.gui.Dispatch(reactive.UIOperationFailed(id, err))
}

func (rgc *ReactiveGUIController) AddLog(message string, isError bool) {
	rgc.gui.AddLog(message, isError)
}

func (rgc *ReactiveGUIController) Complete(success bool, err error) {
	rgc.gui.Complete(success, err)
}

func (rgc *ReactiveGUIController) CompleteWithStats(success bool, err error, stats *ContainerStats) {
	rgc.gui.CompleteWithStats(success, err, stats)
}

func (rgc *ReactiveGUIController) Stop() {
	rgc.gui.Stop()
}
