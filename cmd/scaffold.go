package cmd

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/template"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewScaffoldCommand creates the 'scaffold' subcommand for generating specific agent files.
func NewScaffoldCommand(ctx context.Context, fs afero.Fs, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold [agentName] [fileNames...]",
		Short: "Scaffold specific files for an agent",
		Long: `Scaffold specific files for an agent. Available resources:
  - client: HTTP client for making API calls
  - exec: Execute shell commands and scripts
  - llm: Large Language Model interaction
  - python: Run Python scripts
  - response: API response handling
  - workflow: Workflow configuration file`,
		Args: cobra.MinimumNArgs(1), // Require at least one argument (agentName)
		Run: func(_ *cobra.Command, args []string) {
			agentName := args[0]
			fileNames := args[1:]

			// If no file names provided, show available resources
			if len(fileNames) == 0 {
				logger.Info("Available resources")
				logger.Info("  - client: HTTP client for making API calls")
				logger.Info("  - exec: Execute shell commands and scripts")
				logger.Info("  - llm: Large Language Model interaction")
				logger.Info("  - python: Run Python scripts")
				logger.Info("  - response: API response handling")
				logger.Info("  - workflow: Workflow configuration file")
				return
			}

			// Validate and process each file name
			validResources := map[string]bool{
				"client":   true,
				"exec":     true,
				"llm":      true,
				"python":   true,
				"response": true,
				"workflow": true,
			}

			var invalidResources []string
			for _, fileName := range fileNames {
				// Remove .pkl extension if present
				resourceName := strings.TrimSuffix(fileName, ".pkl")
				if !validResources[resourceName] {
					invalidResources = append(invalidResources, fileName)
					continue
				}

				if err := template.GenerateSpecificAgentFile(ctx, fs, logger, agentName, resourceName); err != nil {
					logger.Error("error scaffolding file:", err)
					logger.Error("Error", "err", err)
				} else {
					// Workflow files go in the main directory, others go in resources
					var filePath string
					if resourceName == "workflow" {
						filePath = filepath.Join(agentName, resourceName+".pkl")
					} else {
						filePath = filepath.Join(agentName, "resources", resourceName+".pkl")
					}
					logger.Info("Successfully scaffolded file", "file", filePath)
				}
			}

			// If there were invalid resources, show them and the available options
			if len(invalidResources) > 0 {
				logger.Error("Invalid resource(s)", "resources", strings.Join(invalidResources, ", "))
				logger.Info("Available resources")
				logger.Info("  - client: HTTP client for making API calls")
				logger.Info("  - exec: Execute shell commands and scripts")
				logger.Info("  - llm: Large Language Model interaction")
				logger.Info("  - python: Run Python scripts")
				logger.Info("  - response: API response handling")
				logger.Info("  - workflow: Workflow configuration file")
			}
		},
	}
}
