package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/ui"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// createGUILogger creates a logger that outputs to the GUI
func createGUILogger(gui *ui.GUIController) *logging.Logger {
	// Create a logger that writes to the GUI
	guiWriter := &guiWriter{gui: gui}
	baseLogger := log.New(guiWriter)
	baseLogger.SetLevel(log.InfoLevel)
	return &logging.Logger{Logger: baseLogger}
}

// guiWriter implements io.Writer to send log messages to GUI
type guiWriter struct {
	gui *ui.GUIController
}

func (gw *guiWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	// Remove newlines and add to GUI logs
	message = strings.TrimSpace(message)
	if message != "" {
		gw.gui.AddLog(message, false)
	}
	return len(p), nil
}

// NewBuildCommand creates the 'build' command and passes the necessary dependencies.
func NewBuildCommand(ctx context.Context, fs afero.Fs, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "build [package]",
		Aliases: []string{"b"},
		Example: "$ kdeps build ./myAgent.kdeps",
		Short:   "Build a dockerized AI agent",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			pkgFile := args[0]

			// Create and start modern GUI immediately to prevent terminal conflicts
			operations := []string{"Package Extraction", "Dockerfile Generation", "Docker Image Build", "Image Cleanup"}
			gui := ui.NewGUIController(ctx, operations)
			
			// Start the GUI immediately
			if err := gui.Start(); err != nil {
				return fmt.Errorf("failed to start GUI: %w", err)
			}

			// Give GUI time to initialize and take control of terminal
			time.Sleep(200 * time.Millisecond)
			
			// Add initial log message
			gui.AddLog("🚀 Starting kdeps build operation...", false)

			// Create a GUI logger to show info messages in live output
			guiLogger := createGUILogger(gui)

			// Step 1: Extract package
			gui.UpdateOperation(0, ui.StatusRunning, fmt.Sprintf("Extracting %s...", pkgFile), 0.0)
			pkgProject, err := archiver.ExtractPackage(fs, ctx, kdepsDir, pkgFile, guiLogger)
			if err != nil {
				gui.UpdateOperationError(0, fmt.Errorf("package extraction failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(0, ui.StatusCompleted, "Package extracted successfully", 1.0)

			// Step 2: Build Dockerfile
			gui.UpdateOperation(1, ui.StatusRunning, "Generating Dockerfile and build context...", 0.0)
			runDir, _, _, _, _, _, _, _, err := docker.BuildDockerfile(fs, ctx, systemCfg, kdepsDir, pkgProject, guiLogger)
			if err != nil {
				gui.UpdateOperationError(1, fmt.Errorf("dockerfile generation failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(1, ui.StatusCompleted, "Dockerfile generated successfully", 1.0)

			// Step 3: Build Docker image
			gui.UpdateOperation(2, ui.StatusRunning, "Building Docker image...", 0.0)
			dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				gui.UpdateOperationError(2, fmt.Errorf("docker client creation failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}

			agentContainerName, agentContainerNameAndVersion, err := docker.BuildDockerImageWithGUI(fs, ctx, systemCfg, dockerClient, runDir, kdepsDir, pkgProject, gui, guiLogger)
			if err != nil {
				gui.UpdateOperationError(2, fmt.Errorf("docker image build failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(2, ui.StatusCompleted, fmt.Sprintf("Docker image %s built successfully", agentContainerNameAndVersion), 1.0)

			// Step 4: Cleanup build images
			gui.UpdateOperation(3, ui.StatusRunning, "Cleaning up intermediate build images...", 0.0)
			if err := docker.CleanupDockerBuildImages(fs, ctx, agentContainerName, dockerClient); err != nil {
				gui.UpdateOperationError(3, fmt.Errorf("image cleanup failed: %w", err))
				gui.Complete(false, err)
				gui.Wait()
				return err
			}
			gui.UpdateOperation(3, ui.StatusCompleted, "Build images cleaned up successfully", 1.0)

			// Success!
			containerStats := &ui.ContainerStats{
				ImageName:    agentContainerName,
				ImageVersion: agentContainerNameAndVersion,
				Command:      "build",
			}
			gui.CompleteWithStats(true, nil, containerStats)
			
			// Wait for user input to exit
			gui.Wait()
			
			return nil
		},
	}
}
