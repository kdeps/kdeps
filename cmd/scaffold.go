package cmd

import (
	"context"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewScaffoldCommand creates the 'scaffold' subcommand for generating specific agent files.
func NewScaffoldCommand(fs afero.Fs, ctx context.Context, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold [agentName] [fileNames...]",
		Short: "Scaffold specific files for an agent",
		Long: `Scaffold specific files for an agent. Available resources:
  - client: HTTP client for making API calls
  - exec: Execute shell commands and scripts
  - llm: Large Language Model interaction
  - python: Run Python scripts
  - response: API response handling`,
		Args: cobra.MinimumNArgs(1), // Require at least one argument (agentName)
		Run: func(cmd *cobra.Command, args []string) {
			agentName := args[0]
			fileNames := args[1:]

			// If no file names provided, show available resources
			if len(fileNames) == 0 {
				PrintlnFn("Available resources:")
				PrintlnFn("  - client: HTTP client for making API calls")
				PrintlnFn("  - exec: Execute shell commands and scripts")
				PrintlnFn("  - llm: Large Language Model interaction")
				PrintlnFn("  - python: Run Python scripts")
				PrintlnFn("  - response: API response handling")
				return
			}

			// Validate and process each file name
			validResources := map[string]bool{
				"client":   true,
				"exec":     true,
				"llm":      true,
				"python":   true,
				"response": true,
			}

			var invalidResources []string
			for _, fileName := range fileNames {
				// Remove .pkl extension if present
				resourceName := strings.TrimSuffix(fileName, ".pkl")
				if !validResources[resourceName] {
					invalidResources = append(invalidResources, fileName)
					continue
				}

				if err := GenerateSpecificAgentFileFn(fs, ctx, logger, agentName, resourceName); err != nil {
					logger.Error("error scaffolding file:", err)
					PrintlnFn(errorStyle.Render("Error:"), err)
				} else {
					PrintlnFn(successStyle.Render("Successfully scaffolded file:"), primaryStyle.Render(JoinPathFn(agentName, "resources", resourceName+".pkl")))
				}
			}

			// If there were invalid resources, show them and the available options
			if len(invalidResources) > 0 {
				PrintlnFn("\nInvalid resource(s):", JoinFn(invalidResources, ", "))
				PrintlnFn("\nAvailable resources:")
				PrintlnFn("  - client: HTTP client for making API calls")
				PrintlnFn("  - exec: Execute shell commands and scripts")
				PrintlnFn("  - llm: Large Language Model interaction")
				PrintlnFn("  - python: Run Python scripts")
				PrintlnFn("  - response: API response handling")
			}
		},
	}
}
