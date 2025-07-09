package template

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/version"
	"github.com/kdeps/kdeps/templates"
	"github.com/spf13/afero"
)

var (
	lightBlue  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6495ED")).Bold(true)
	lightGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#90EE90")).Bold(true)
)

func printWithDots(message string) {
	fmt.Print(lightBlue.Render(message))
	fmt.Print("...")
	fmt.Println()
}

func validateAgentName(agentName string) error {
	if strings.TrimSpace(agentName) == "" {
		return errors.New("agent name cannot be empty or only whitespace")
	}
	if strings.Contains(agentName, " ") {
		return errors.New("agent name cannot contain spaces")
	}
	return nil
}

func promptForAgentName() (string, error) {
	// Skip prompt if NON_INTERACTIVE=1
	if os.Getenv("NON_INTERACTIVE") == "1" {
		return "test-agent", nil
	}

	var name string
	form := huh.NewInput().
		Title("Configure Your AI Agent").
		Prompt("Enter a name for your AI Agent (no spaces): ").
		Validate(validateAgentName).
		Value(&name)

	if err := form.Run(); err != nil {
		return "", err
	}
	return name, nil
}

func createDirectory(fs afero.Fs, logger *logging.Logger, path string) error {
	if path == "" {
		err := errors.New("directory path cannot be empty")
		logger.Error(err)
		return err
	}
	printWithDots("Creating directory: " + lightGreen.Render(path))
	if err := fs.MkdirAll(path, os.ModePerm); err != nil {
		logger.Error(err)
		return err
	}
	if os.Getenv("NON_INTERACTIVE") != "1" {
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

func createFile(fs afero.Fs, logger *logging.Logger, path string, content string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	printWithDots("Creating file: " + lightGreen.Render(path))
	if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
		logger.Error(err)
		return err
	}
	if os.Getenv("NON_INTERACTIVE") != "1" {
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

func loadTemplate(templatePath string, data map[string]string) (string, error) {
	// If TEMPLATE_DIR is set, load from disk instead of embedded FS
	if dir := os.Getenv("TEMPLATE_DIR"); dir != "" {
		path := filepath.Join(dir, filepath.Base(templatePath))
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read template from disk: %w", err)
		}
		tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
		if err != nil {
			return "", fmt.Errorf("failed to parse template file: %w", err)
		}
		var output bytes.Buffer
		if err := tmpl.Execute(&output, data); err != nil {
			return "", fmt.Errorf("failed to execute template: %w", err)
		}
		return output.String(), nil
	}

	// Otherwise, use embedded FS
	content, err := templates.TemplatesFS.ReadFile(filepath.Base(templatePath))
	if err != nil {
		return "", fmt.Errorf("failed to read embedded template: %w", err)
	}
	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template file: %w", err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return output.String(), nil
}

// GenerateWorkflowFile generates a workflow file for the agent.
func GenerateWorkflowFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, name string) error {
	// Validate agent name first
	if err := validateAgentName(name); err != nil {
		return err
	}

	// Create the directory if it doesn't exist
	if err := fs.MkdirAll(mainDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	templatePath := "workflow.pkl"
	outputPath := filepath.Join(mainDir, "workflow.pkl")

	// Template data for dynamic replacement
	templateData := map[string]string{
		"Header":         fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`, schema.SchemaVersion(ctx)),
		"Name":           name,
		"OllamaImageTag": version.DefaultOllamaImageTag,
	}

	// Load and process the template
	content, err := loadTemplate(templatePath, templateData)
	if err != nil {
		logger.Error("failed to load workflow template: ", err)
		return err
	}

	return createFile(fs, logger, outputPath, content)
}

// GenerateResourceFiles generates resource files for the agent.
func GenerateResourceFiles(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, name string) error {
	// Validate agent name first
	if err := validateAgentName(name); err != nil {
		return err
	}

	resourceDir := filepath.Join(mainDir, "resources")
	if err := fs.MkdirAll(resourceDir, 0o755); err != nil {
		return fmt.Errorf("failed to create resources directory: %w", err)
	}

	// Common template data
	templateData := map[string]string{
		"Header": fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`, schema.SchemaVersion(ctx)),
		"Name":   name,
	}

	// List all embedded template files
	files, err := templates.TemplatesFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded templates directory: %w", err)
	}

	for _, file := range files {
		// Skip directories and files that aren't .pkl
		if file.IsDir() || filepath.Ext(file.Name()) != ".pkl" {
			continue
		}

		// Skip the workflow.pkl file
		if file.Name() == "workflow.pkl" {
			continue
		}

		// Skip the agent.pkl file (it's not a standard resource file)
		if file.Name() == "agent.pkl" {
			continue
		}

		templatePath := file.Name()
		content, err := loadTemplate(templatePath, templateData)
		if err != nil {
			logger.Error("failed to process template: ", err)
			return err
		}

		outputPath := filepath.Join(resourceDir, file.Name())
		if err := createFile(fs, logger, outputPath, content); err != nil {
			return err
		}
	}

	return nil
}

func GenerateSpecificAgentFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
	// Validate agent name
	if err := validateAgentName(agentName); err != nil {
		return err
	}

	headerTemplate := `amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`
	if strings.ToLower(agentName) == "workflow.pkl" {
		headerTemplate = `amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`
	}

	templatePath := agentName + ".pkl"
	templateData := map[string]string{
		"Header":         fmt.Sprintf(headerTemplate, schema.SchemaVersion(ctx)),
		"Name":           agentName,
		"OllamaImageTag": version.DefaultOllamaImageTag,
	}

	// Load the template
	content, err := loadTemplate(templatePath, templateData)
	if err != nil {
		logger.Error("failed to load specific template: ", err)
		return err
	}

	// Determine the output directory
	var outputDir string
	if strings.ToLower(agentName) == "workflow.pkl" {
		outputDir = mainDir // Place workflow.pkl in the main directory
	} else {
		outputDir = filepath.Join(mainDir, "resources") // Place other files in the resources folder
	}

	// Create the output directory if it doesn't exist
	if err := fs.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, agentName+".pkl")
	return createFile(fs, logger, outputPath, content)
}

func GenerateAgent(fs afero.Fs, ctx context.Context, logger *logging.Logger, baseDir, agentName string) error {
	// Validate agent name
	if err := validateAgentName(agentName); err != nil {
		return err
	}

	// Create the main directory under baseDir
	mainDir := filepath.Join(baseDir, agentName)
	if err := fs.MkdirAll(mainDir, 0o755); err != nil {
		return fmt.Errorf("failed to create main directory: %w", err)
	}

	// Generate workflow file
	if err := GenerateWorkflowFile(fs, ctx, logger, mainDir, agentName); err != nil {
		return err
	}

	// Generate resource files
	if err := GenerateResourceFiles(fs, ctx, logger, mainDir, agentName); err != nil {
		return err
	}

	// Generate the agent file
	if err := GenerateSpecificAgentFile(fs, ctx, logger, mainDir, agentName); err != nil {
		return err
	}

	return nil
}

// GenerateDockerfileFromTemplate generates a Dockerfile using the template system.
func GenerateDockerfileFromTemplate(templateData map[string]interface{}) (string, error) {
	// If TEMPLATE_DIR is set, load from disk instead of embedded FS
	if dir := os.Getenv("TEMPLATE_DIR"); dir != "" {
		path := filepath.Join(dir, "dockerfile.template")
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read dockerfile template from disk: %w", err)
		}
		tmpl, err := template.New("dockerfile.template").Parse(string(content))
		if err != nil {
			return "", fmt.Errorf("failed to parse dockerfile template file: %w", err)
		}
		var output bytes.Buffer
		if err := tmpl.Execute(&output, templateData); err != nil {
			return "", fmt.Errorf("failed to execute dockerfile template: %w", err)
		}
		return output.String(), nil
	}

	// Otherwise, use embedded FS
	content, err := templates.TemplatesFS.ReadFile("dockerfile.template")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded dockerfile template: %w", err)
	}
	tmpl, err := template.New("dockerfile.template").Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse dockerfile template file: %w", err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, templateData); err != nil {
		return "", fmt.Errorf("failed to execute dockerfile template: %w", err)
	}
	return output.String(), nil
}
