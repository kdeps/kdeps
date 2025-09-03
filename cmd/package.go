package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/workflow"
	"github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// Define styles using lipgloss (moved inside functions to avoid global variables)

// NewPackageCommand creates the 'package' command and passes the necessary dependencies.
func NewPackageCommand(ctx context.Context, fs afero.Fs, kdepsDir string, env *environment.Environment, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "package [agent-dir]",
		Aliases: []string{"p"},
		Example: "$ kdeps package ./myAgent/",
		Short:   "Package an AI agent to .kdeps file",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return executePackageCommand(ctx, fs, kdepsDir, env, logger, args[0])
		},
	}
}

func executePackageCommand(ctx context.Context, fs afero.Fs, kdepsDir string, env *environment.Environment, logger *logging.Logger, agentDir string) error {
	styles := createPackageStyles()

	wfFile, err := findWorkflowFile(fs, agentDir, logger, styles.errorStyle)
	if err != nil {
		return err
	}

	wf, err := loadWorkflow(ctx, wfFile, logger, styles.errorStyle)
	if err != nil {
		return err
	}

	err = compileProject(fs, ctx, wf, kdepsDir, agentDir, env, logger, styles.errorStyle)
	if err != nil {
		return err
	}

	printSuccessMessage(agentDir, styles)
	return nil
}

type PackageStyles struct {
	primaryStyle lipgloss.Style
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
}

func createPackageStyles() *PackageStyles {
	return &PackageStyles{
		primaryStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("75")),
		successStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true),
		errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
	}
}

func findWorkflowFile(fs afero.Fs, agentDir string, logger *logging.Logger, errorStyle lipgloss.Style) (string, error) {
	wfFile, err := archiver.FindWorkflowFile(fs, agentDir, logger)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errorStyle.Render("Error finding workflow file"), err)
	}
	return wfFile, nil
}

func loadWorkflow(ctx context.Context, wfFile string, logger *logging.Logger, errorStyle lipgloss.Style) (interface{}, error) {
	wf, err := workflow.LoadWorkflow(ctx, wfFile, logger)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errorStyle.Render("Error loading workflow"), err)
	}
	return wf, nil
}

func compileProject(fs afero.Fs, ctx context.Context, wf interface{}, kdepsDir, agentDir string, env *environment.Environment, logger *logging.Logger, errorStyle lipgloss.Style) error {
	// Cast to the expected workflow type with all required methods
	workflowObj, ok := wf.(interface {
		GetAgentID() string
		GetVersion() string
		GetAgentIcon() *string
		GetWorkflows() []string
		GetAuthors() *[]string
		GetDescription() string
		GetDocumentation() *string
		GetHeroImage() *string
		GetRepository() *string
		GetWebsite() *string
		GetSettings() project.Settings
		GetTargetActionID() string
	})
	if !ok {
		return fmt.Errorf("%s: invalid workflow type", errorStyle.Render("Error compiling project"))
	}

	_, _, err := archiver.CompileProject(fs, ctx, workflowObj, kdepsDir, agentDir, env, logger)
	if err != nil {
		return fmt.Errorf("%s: %w", errorStyle.Render("Error compiling project"), err)
	}
	return nil
}

func printSuccessMessage(agentDir string, styles *PackageStyles) {
	fmt.Println(styles.successStyle.Render("AI agent packaged successfully:"), styles.primaryStyle.Render(agentDir)) //nolint:forbidigo // CLI user feedback
}
