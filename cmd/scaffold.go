package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/template"
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
  - response: API response handling
  - workflow: Workflow automation and orchestration`,
		Args: cobra.MinimumNArgs(1), // Require at least one argument (agentName)
		Run: func(cmd *cobra.Command, args []string) {
			agentName := args[0]
			fileNames := args[1:]

			// If no file names provided, show available resources
			if len(fileNames) == 0 {
				fmt.Println("Available resources:")
				fmt.Println("  - client: HTTP client for making API calls")
				fmt.Println("  - exec: Execute shell commands and scripts")
				fmt.Println("  - llm: Large Language Model interaction")
				fmt.Println("  - python: Run Python scripts")
				fmt.Println("  - response: API response handling")
				fmt.Println("  - workflow: Workflow automation and orchestration")
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

				if err := template.GenerateSpecificAgentFile(fs, ctx, logger, agentName, resourceName); err != nil {
					logger.Error("error scaffolding file:", err)
					fmt.Println(errorStyle.Render("Error:"), err)
				} else {
					var filePath string
					if resourceName == "workflow" {
						filePath = filepath.Join(agentName, "workflow.pkl")
					} else {
						filePath = filepath.Join(agentName, "resources", resourceName+".pkl")
					}
					fmt.Println(successStyle.Render("Successfully scaffolded file:"), primaryStyle.Render(filePath))
				}
			}

			// If there were invalid resources, show them and the available options
			if len(invalidResources) > 0 {
				fmt.Println("\nInvalid resource(s):", strings.Join(invalidResources, ", "))
				fmt.Println("\nAvailable resources:")
				fmt.Println("  - client: HTTP client for making API calls")
				fmt.Println("  - exec: Execute shell commands and scripts")
				fmt.Println("  - llm: Large Language Model interaction")
				fmt.Println("  - python: Run Python scripts")
				fmt.Println("  - response: API response handling")
				fmt.Println("  - workflow: Workflow automation and orchestration")
			}
		},
	}
}
