package template

import (
	"bytes"
	"context"
	"embed"
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
	"github.com/kdeps/kdeps/pkg/texteditor"
	"github.com/spf13/afero"
)

// Embed the templates directory.
//
//go:embed templates/*.pkl
var templatesFS embed.FS

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
		return errors.New("Agent name cannot be empty or only whitespace. Please provide a valid name.")
	}
	if strings.Contains(agentName, " ") {
		return errors.New("Agent name cannot contain spaces. Please provide a valid name.")
	}
	return nil
}

func promptForAgentName(ctx context.Context) (string, error) {
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

func createDirectory(fs afero.Fs, ctx context.Context, logger *logging.Logger, path string) error {
	printWithDots(fmt.Sprintf("Creating directory: %s", lightGreen.Render(path)))
	if err := fs.MkdirAll(path, os.ModePerm); err != nil {
		logger.Error(err)
		return err
	}
	time.Sleep(80 * time.Millisecond)
	return nil
}

func createFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, path string, content string) error {
	printWithDots(fmt.Sprintf("Creating file: %s", lightGreen.Render(path)))
	if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
		logger.Error(err)
		return err
	}
	time.Sleep(80 * time.Millisecond)
	return nil
}

func generateWorkflowFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, name string) error {
	templatePath := "templates/workflow.pkl"
	outputPath := filepath.Join(mainDir, "workflow.pkl")

	// Template data for dynamic replacement
	templateData := map[string]string{
		"Header": fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`, schema.SchemaVersion()),
		"Name":   name,
	}

	// Load and process the template
	content, err := loadTemplate(templatePath, ctx, templateData)
	if err != nil {
		logger.Error("Failed to load workflow template: ", err)
		return err
	}

	return createFile(fs, logger, outputPath, content)
}

func loadTemplate(ctx context.Context, templatePath string, data map[string]string) (string, error) {
	// Load the template from the embedded FS
	content, err := templatesFS.ReadFile(templatePath)
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

func generateResourceFiles(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, name string) error {
	resourceDir := filepath.Join(mainDir, "resources")
	if err := createDirectory(fs, ctx, logger, resourceDir); err != nil {
		return err
	}

	// Common template data
	templateData := map[string]string{
		"Header": fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`, schema.SchemaVersion()),
		"Name":   name,
	}

	// List all embedded template files
	files, err := templatesFS.ReadDir("templates")
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

		templatePath := filepath.Join("templates", file.Name())
		content, err := loadTemplate(templatePath, ctx, templateData)
		if err != nil {
			logger.Error("Failed to process template: ", err)
			return err
		}

		outputPath := filepath.Join(resourceDir, file.Name())
		if err := createFile(fs, logger, outputPath, content); err != nil {
			return err
		}
	}

	return nil
}

func generateSpecificFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, fileName, agentName string) error {
	// Automatically add .pkl extension if not present
	if !strings.HasSuffix(fileName, ".pkl") {
		fileName += ".pkl"
	}

	// Determine the appropriate header based on the file name
	headerTemplate := `amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`
	if strings.ToLower(fileName) == "workflow.pkl" {
		headerTemplate = `amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`
	}

	templatePath := filepath.Join("templates", fileName)
	templateData := map[string]string{
		"Header": fmt.Sprintf(headerTemplate, schema.SchemaVersion()),
		"Name":   agentName,
	}

	// Load the template
	content, err := loadTemplate(templatePath, ctx, templateData)
	if err != nil {
		logger.Error("Failed to load specific template: ", err)
		return err
	}

	// Determine the output directory
	var outputDir string
	if strings.ToLower(fileName) == "workflow.pkl" {
		outputDir = mainDir // Place workflow.pkl in the main directory
	} else {
		outputDir = filepath.Join(mainDir, "resources") // Place other files in the resources folder
	}

	// Create the output directory if it doesn't exist
	if err := createDirectory(fs, ctx, logger, outputDir); err != nil {
		return err
	}

	// Write the generated file
	filePath := filepath.Join(outputDir, fileName)
	if err := createFile(fs, logger, filePath, content); err != nil {
		return err
	}

	// Create the data folder
	dataDir := filepath.Join(mainDir, "data")
	if err := createDirectory(fs, ctx, logger, dataDir); err != nil {
		logger.Error("Failed to create data directory: ", err)
		return err
	}

	return nil
}

func GenerateSpecificAgentFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, agentName, fileName string) error {
	var name string
	var err error

	if agentName != "" {
		if err := validateAgentName(agentName); err != nil {
			return err
		}
		name = agentName
	} else {
		name, err = promptForAgentName(ctx)
		if err != nil {
			logger.Error("Failed to prompt for agent name: ", err)
			return err
		}
	}

	mainDir := fmt.Sprintf("./%s", name)
	if err := createDirectory(fs, ctx, logger, mainDir); err != nil {
		logger.Error("Failed to create main directory: ", err)
		return err
	}

	if err := generateSpecificFile(fs, logger, mainDir, fileName, name); err != nil {
		logger.Error("Failed to generate specific file: ", err)
		return err
	}

	var openFile bool
	editorForm := huh.NewConfirm().
		Title(fmt.Sprintf("Edit %s in Editor?", fileName)).
		Affirmative("Yes").
		Negative("No").
		Value(&openFile)

	err = editorForm.Run()
	if err != nil {
		logger.Error("Failed to display editor confirmation dialog: ", err)
		return err
	}

	if openFile {
		var filePath string
		if strings.ToLower(fileName) == "workflow" {
			// Adjust path for workflows outside the resources folder
			filePath = fmt.Sprintf("%s/%s.pkl", mainDir, fileName)
		} else {
			// Default path for other files in the resources folder
			filePath = fmt.Sprintf("%s/resources/%s.pkl", mainDir, fileName)
		}

		if err := texteditor.EditPkl(fs, ctx, filePath, logger); err != nil {
			logger.Error("Failed to edit file: ", err)
			return fmt.Errorf("failed to edit file: %w", err)
		}
	}

	return nil
}

func GenerateAgent(fs afero.Fs, ctx context.Context, logger *logging.Logger, agentName string) error {
	var name string
	var err error

	if agentName != "" {
		if err := validateAgentName(agentName); err != nil {
			return err
		}
		name = agentName
	} else {
		name, err = promptForAgentName()
		if err != nil {
			logger.Error("Failed to prompt for agent name: ", err)
			return err
		}
	}

	mainDir := fmt.Sprintf("./%s", name)
	if err := createDirectory(fs, ctx, logger, mainDir); err != nil {
		logger.Error("Failed to create main directory: ", err)
		return err
	}
	if err := createDirectory(fs, ctx, logger, mainDir+"/resources"); err != nil {
		logger.Error("Failed to create resources directory: ", err)
		return err
	}
	if err := createDirectory(fs, ctx, logger, mainDir+"/data"); err != nil {
		logger.Error("Failed to create data directory: ", err)
		return err
	}
	if err := generateWorkflowFile(fs, ctx, logger, mainDir, name); err != nil {
		logger.Error("Failed to generate workflow file: ", err)
		return err
	}
	if err := generateResourceFiles(fs, ctx, logger, mainDir, name); err != nil {
		logger.Error("Failed to generate resource files: ", err)
		return err
	}

	var openWorkflow bool
	editorForm := huh.NewConfirm().
		Title("Edit the AI agent in Editor?").
		Affirmative("Yes").
		Negative("No").
		Value(&openWorkflow)

	err = editorForm.Run()
	if err != nil {
		logger.Error("Failed to display editor confirmation dialog: ", err)
		return err
	}

	if openWorkflow {
		workflowFilePath := fmt.Sprintf("%s/workflow.pkl", mainDir)
		if err := texteditor.EditPkl(fs, workflowFilePath, logger); err != nil {
			logger.Error("Failed to edit workflow file: ", err)
			return fmt.Errorf("failed to edit workflow file: %w", err)
		}
	}

	return nil
}
