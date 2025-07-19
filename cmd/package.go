package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/ui"
	"github.com/kdeps/kdeps/pkg/workflow"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// Define styles using lipgloss.
var (
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// NewPackageCommand creates the 'package' command and passes the necessary dependencies.
func NewPackageCommand(ctx context.Context, fs afero.Fs, kdepsDir string, env *environment.Environment, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "package [agent-dir]",
		Aliases: []string{"p"},
		Example: "$ kdeps package ./myAgent/",
		Short:   "Package an AI agent to .kdeps file",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			agentDir := args[0]

			// Create and start modern GUI immediately to prevent terminal conflicts
			operations := []string{"Workflow Discovery", "Workflow Loading", "Project Compilation", "Package Creation"}
			gui := ui.NewGUIController(ctx, operations)

			// Start the GUI immediately
			if err := gui.Start(); err != nil {
				return fmt.Errorf("failed to start GUI: %w", err)
			}

			// Give GUI time to initialize and take control of terminal
			time.Sleep(200 * time.Millisecond)

			// Add initial log message
			gui.AddLog("🚀 Starting kdeps package operation...", false)

			// Create a GUI logger to show info messages in live output
			guiLogger := createGUILogger(gui)

			// Step 1: Find workflow file
			gui.UpdateOperation(0, ui.StatusRunning, fmt.Sprintf("Searching for workflow in %s...", agentDir), 0.0)
			wfFile, err := archiver.FindWorkflowFile(fs, agentDir, guiLogger)
			if err != nil {
				gui.UpdateOperationError(0, fmt.Errorf("workflow discovery failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(0, ui.StatusCompleted, "Workflow file found successfully", 1.0)

			// Step 2: Load workflow
			gui.UpdateOperation(1, ui.StatusRunning, "Loading and validating workflow...", 0.0)
			wf, err := workflow.LoadWorkflow(ctx, wfFile, guiLogger)
			if err != nil {
				gui.UpdateOperationError(1, fmt.Errorf("workflow loading failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(1, ui.StatusCompleted, "Workflow loaded and validated successfully", 1.0)

			// Step 3: Compile project
			gui.UpdateOperation(2, ui.StatusRunning, "Compiling project resources and dependencies...", 0.0)
			_, packagePath, err := archiver.CompileProject(ctx, fs, wf, kdepsDir, agentDir, env, guiLogger)
			if err != nil {
				gui.UpdateOperationError(2, fmt.Errorf("project compilation failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(2, ui.StatusCompleted, "Project compiled successfully", 1.0)

			// Step 4: Package finalization
			gui.UpdateOperation(3, ui.StatusRunning, "Finalizing package...", 0.0)
			gui.UpdateOperation(3, ui.StatusCompleted, fmt.Sprintf("Package created: %s", packagePath), 1.0)

			// Success!
			containerStats := &ui.ContainerStats{
				ImageName: packagePath, // Use packagePath as the main output for package command
				Command:   "package",
			}
			gui.CompleteWithStats(true, nil, containerStats)

			// Wait for user input to exit
			gui.Wait()

			return nil
		},
	}
}
