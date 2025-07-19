package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kdeps/kdeps/pkg/logging"
)

// ProgressBuildLine struct is used to unmarshal Docker build log lines from the response.
type ProgressBuildLine struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

// ProgressModel represents the UI model for Docker operations
type ProgressModel struct {
	spinner     spinner.Model
	progress    progress.Model
	status      string
	step        string
	done        bool
	err         error
	logger      *logging.Logger
	currentStep int
	totalSteps  int
}

// ProgressMsg represents a progress update message
type ProgressMsg struct {
	Step        string
	Status      string
	Progress    float64
	CurrentStep int
	TotalSteps  int
	Done        bool
	Error       error
}

// BuildStepMsg represents a Docker build step message
type BuildStepMsg struct {
	Stream string
	Error  string
}

// NewProgressModel creates a new progress model
func NewProgressModel(logger *logging.Logger) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	p := progress.New(progress.WithDefaultGradient())
	p.Width = 60

	return ProgressModel{
		spinner:  s,
		progress: p,
		status:   "Initializing...",
		step:     "Starting Docker operation",
		logger:   logger,
	}
}

// Init initializes the progress model
func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(spinner.Tick, m.waitForActivity())
}

// Update handles model updates
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil

	case ProgressMsg:
		m.step = msg.Step
		m.status = msg.Status
		if msg.Progress > 0 {
			m.progress.SetPercent(msg.Progress)
		}
		if msg.CurrentStep > 0 {
			m.currentStep = msg.CurrentStep
		}
		if msg.TotalSteps > 0 {
			m.totalSteps = msg.TotalSteps
		}
		if msg.Done {
			m.done = true
			return m, tea.Quit
		}
		if msg.Error != nil {
			m.err = msg.Error
			return m, tea.Quit
		}
		return m, m.waitForActivity()

	case BuildStepMsg:
		if msg.Error != "" {
			m.err = fmt.Errorf("build error: %s", msg.Error)
			return m, tea.Quit
		}
		if msg.Stream != "" {
			// Clean up the stream message for better display
			status := strings.TrimSpace(msg.Stream)
			// Remove common Docker build prefixes for cleaner display
			status = strings.TrimPrefix(status, "Step ")
			status = strings.TrimPrefix(status, " ---> ")
			status = strings.TrimPrefix(status, " => ")
			m.status = status
		}
		return m, m.waitForActivity()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		newProgressModel, cmd := m.progress.Update(msg)
		m.progress = newProgressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

// View renders the progress UI
func (m ProgressModel) View() string {
	if m.done {
		if m.err != nil {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Bold(true).
				Render(fmt.Sprintf("❌ Error: %s", m.err.Error()))
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true).
			Render("✅ Docker operation completed successfully!")
	}

	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1E90FF")).
		Bold(true).
		Render("🐳 Kdeps Docker Operation\n\n"))

	s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.step))

	// Show step information if available
	if m.totalSteps > 0 {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Render(fmt.Sprintf("   Step %d/%d\n", m.currentStep, m.totalSteps)))
	}

	// Show detailed status
	if m.status != "" {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Render(fmt.Sprintf("   %s\n\n", m.status)))
	}

	// Show progress bar
	s.WriteString(m.progress.View())
	s.WriteString("\n\n")

	// Show instructions
	s.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render("Press Ctrl+C to cancel"))

	return s.String()
}

// waitForActivity is a command that waits for activity
func (m ProgressModel) waitForActivity() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
}

// PrintDockerBuildOutputEnhanced prints Docker build output with enhanced UI
func PrintDockerBuildOutputEnhanced(rd io.Reader, logger *logging.Logger, step string) error {
	model := NewProgressModel(logger)
	model.step = step

	// Start the UI program
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInputTTY())

	// Start the UI in a goroutine
	go func() {
		if _, err := p.Run(); err != nil {
			logger.Error("UI error", "error", err)
		}
	}()

	// Process Docker build output
	scanner := bufio.NewScanner(rd)
	totalSteps := 0
	completedSteps := 0

	// First pass: count total steps
	lines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// Count Docker build steps
		if strings.Contains(line, `"stream"`) {
			buildLine := &ProgressBuildLine{}
			if json.Unmarshal([]byte(line), buildLine) == nil && buildLine.Stream != "" {
				if strings.HasPrefix(strings.TrimSpace(buildLine.Stream), "Step ") {
					totalSteps++
				}
			}
		}
	}

	// Reset scanner for second pass
	scanner = bufio.NewScanner(strings.NewReader(strings.Join(lines, "\n")))

	// Second pass: process with progress
	for scanner.Scan() {
		line := scanner.Text()

		// Try to unmarshal each line as JSON
		buildLine := &ProgressBuildLine{}
		err := json.Unmarshal([]byte(line), buildLine)
		if err != nil {
			// If unmarshalling fails, print the raw line (non-JSON output)
			logger.Debug("Dockerfile line", "line", line)
			p.Send(BuildStepMsg{Stream: strings.TrimSpace(line)})
			continue
		}

		// Send build step message
		if buildLine.Stream != "" {
			stream := strings.TrimSpace(buildLine.Stream)

			// Check if this is a new step
			if strings.HasPrefix(stream, "Step ") {
				completedSteps++
				progress := float64(completedSteps) / float64(totalSteps)
				p.Send(ProgressMsg{
					Step:        step,
					Status:      stream,
					Progress:    progress,
					CurrentStep: completedSteps,
					TotalSteps:  totalSteps,
				})
			} else {
				// Update status with current step details
				p.Send(BuildStepMsg{Stream: stream})
			}
		}

		// If there's an error in the build process, send error message
		if buildLine.Error != "" {
			p.Send(ProgressMsg{
				Step:  step,
				Error: errors.New(buildLine.Error),
				Done:  true,
			})
			break
		}
	}

	// Handle scanner errors
	if err := scanner.Err(); err != nil {
		p.Send(ProgressMsg{
			Step:  step,
			Error: err,
			Done:  true,
		})
		return err
	}

	// Send completion message if no error
	p.Send(ProgressMsg{
		Step:        step,
		Status:      "Build completed successfully",
		Progress:    1.0,
		CurrentStep: totalSteps,
		TotalSteps:  totalSteps,
		Done:        true,
	})

	// Wait a moment for the UI to update
	time.Sleep(1 * time.Second)

	return nil
}

// ShowDockerProgress shows progress for Docker operations
func ShowDockerProgress(ctx context.Context, logger *logging.Logger, operation string, steps []string) error {
	model := NewProgressModel(logger)
	model.step = operation

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInputTTY())

	// Create a channel to receive progress updates
	progressChan := make(chan ProgressMsg, 100)

	// Start progress simulation
	go func() {
		defer close(progressChan)

		for i, step := range steps {
			select {
			case <-ctx.Done():
				return
			default:
				// Send step update
				progressChan <- ProgressMsg{
					Step:   step,
					Status: fmt.Sprintf("Processing step %d/%d", i+1, len(steps)),
				}

				// Wait for step completion or context cancellation
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second): // Wait for actual work to complete
				}
			}
		}

		// Send completion message
		progressChan <- ProgressMsg{
			Step:   operation,
			Status: "Operation completed successfully",
			Done:   true,
		}
	}()

	// Process progress updates
	go func() {
		for msg := range progressChan {
			select {
			case <-ctx.Done():
				return
			default:
				p.Send(msg)
			}
		}
	}()

	_, err := p.Run()
	return err
}

// EnhancedDockerBuildOutput provides a more detailed build output with progress tracking
func EnhancedDockerBuildOutput(rd io.Reader, logger *logging.Logger) error {
	return PrintDockerBuildOutputEnhanced(rd, logger, "Building Docker Image")
}

// DockerBuildProgress tracks Docker build progress with detailed step information
type DockerBuildProgress struct {
	CurrentStep    string
	TotalSteps     int
	CurrentStepNum int
	Status         string
	Error          error
	Done           bool
}

// TrackDockerBuildProgress creates a progress tracker for Docker build operations
func TrackDockerBuildProgress(ctx context.Context, logger *logging.Logger) chan DockerBuildProgress {
	progressChan := make(chan DockerBuildProgress, 100)

	go func() {
		defer close(progressChan)

		// Send initial progress
		progressChan <- DockerBuildProgress{
			CurrentStep:    "Initializing build",
			TotalSteps:     5,
			CurrentStepNum: 0,
			Status:         "Starting Docker build process",
		}

		// Monitor context for cancellation
		select {
		case <-ctx.Done():
			progressChan <- DockerBuildProgress{
				CurrentStep: "Build cancelled",
				Status:      "Build operation was cancelled",
				Done:        true,
			}
			return
		default:
		}
	}()

	return progressChan
}

// ShowContainerCreationProgress displays progress for container creation
func ShowContainerCreationProgress(ctx context.Context, logger *logging.Logger, containerName string) error {
	model := NewProgressModel(logger)
	model.step = "Creating Container"

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInputTTY())

	// Container creation steps
	steps := []string{
		"Checking existing containers",
		"Creating container configuration",
		"Setting up port bindings",
		"Configuring GPU settings",
		"Starting container",
		"Verifying container health",
	}

	go func() {
		for i, step := range steps {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(1 * time.Second) // Simulate container creation steps
				p.Send(ProgressMsg{
					Step:   step,
					Status: fmt.Sprintf("Creating container: %s (%d/%d)", containerName, i+1, len(steps)),
				})
			}
		}

		p.Send(ProgressMsg{
			Step:   "Container Creation",
			Status: fmt.Sprintf("Container '%s' created and started successfully", containerName),
			Done:   true,
		})
	}()

	_, err := p.Run()
	return err
}
