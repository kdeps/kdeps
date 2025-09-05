package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
  - workflow: Workflow automation and orchestration`,
		Args: cobra.MinimumNArgs(1), // Require at least one argument (agentName)
		Run: func(_ *cobra.Command, args []string) {
			// Define styles using lipgloss
			primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true)
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)

			agentName := args[0]
			fileNames := args[1:]

			// If no file names provided, show available resources
			if len(fileNames) == 0 {
				fmt.Println("Available resources:")                                //nolint:forbidigo // CLI user feedback
				fmt.Println("  - client: HTTP client for making API calls")        //nolint:forbidigo // CLI user feedback
				fmt.Println("  - exec: Execute shell commands and scripts")        //nolint:forbidigo // CLI user feedback
				fmt.Println("  - llm: Large Language Model interaction")           //nolint:forbidigo // CLI user feedback
				fmt.Println("  - python: Run Python scripts")                      //nolint:forbidigo // CLI user feedback
				fmt.Println("  - response: API response handling")                 //nolint:forbidigo // CLI user feedback
				fmt.Println("  - workflow: Workflow automation and orchestration") //nolint:forbidigo // CLI user feedback
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
					fmt.Println(errorStyle.Render("Error:"), err) //nolint:forbidigo // CLI user feedback
				} else {
					var filePath string
					if resourceName == "workflow" {
						filePath = filepath.Join(agentName, "workflow.pkl")
					} else {
						filePath = filepath.Join(agentName, "resources", resourceName+".pkl")
					}
					fmt.Println(successStyle.Render("Successfully scaffolded file:"), primaryStyle.Render(filePath)) //nolint:forbidigo // CLI user feedback
				}
			}

			// If there were invalid resources, show them and the available options
			if len(invalidResources) > 0 {
				fmt.Println("\nInvalid resource(s):", strings.Join(invalidResources, ", ")) //nolint:forbidigo // CLI user feedback
				fmt.Println("\nAvailable resources:")                                       //nolint:forbidigo // CLI user feedback
				fmt.Println("  - client: HTTP client for making API calls")                 //nolint:forbidigo // CLI user feedback
				fmt.Println("  - exec: Execute shell commands and scripts")                 //nolint:forbidigo // CLI user feedback
				fmt.Println("  - llm: Large Language Model interaction")                    //nolint:forbidigo // CLI user feedback
				fmt.Println("  - python: Run Python scripts")                               //nolint:forbidigo // CLI user feedback
				fmt.Println("  - response: API response handling")                          //nolint:forbidigo // CLI user feedback
				fmt.Println("  - workflow: Workflow automation and orchestration")          //nolint:forbidigo // CLI user feedback
			}
		},
	}
}
