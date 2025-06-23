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
	versionpkg "github.com/kdeps/kdeps/pkg/version"
	"github.com/kdeps/kdeps/templates"
	"github.com/spf13/afero"
)

var (
	lightBlue  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6495ED")).Bold(true)
	lightGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#90EE90")).Bold(true)
)

func PrintWithDots(message string) {
	fmt.Print(lightBlue.Render(message))
	fmt.Print("...")
	fmt.Println()
}

func ValidateAgentName(agentName string) error {
	if strings.TrimSpace(agentName) == "" {
		return errors.New("agent name cannot be empty or only whitespace")
	}
	if strings.Contains(agentName, " ") {
		return errors.New("agent name cannot contain spaces")
	}
	return nil
}

func PromptForAgentName() (string, error) {
	// Skip prompt if NON_INTERACTIVE=1
	if os.Getenv("NON_INTERACTIVE") == "1" {
		return "test-agent", nil
	}

	var name string
	form := huh.NewInput().
		Title("Configure Your AI Agent").
		Prompt("Enter a name for your AI Agent (no spaces): ").
		Validate(ValidateAgentName).
		Value(&name)

	if err := form.Run(); err != nil {
		return "", err
	}
	return name, nil
}

func CreateDirectory(fs afero.Fs, logger *logging.Logger, path string) error {
	if path == "" {
		err := errors.New("directory path cannot be empty")
		logger.Error(err)
		return err
	}
	PrintWithDots("Creating directory: " + lightGreen.Render(path))
	if err := fs.MkdirAll(path, os.ModePerm); err != nil {
		logger.Error(err)
		return err
	}
	if os.Getenv("NON_INTERACTIVE") != "1" {
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

// safeLogger returns a usable logger, falling back to the base logger when the provided one is nil.
func safeLogger(l *logging.Logger) *logging.Logger {
	if l == nil {
		return logging.GetLogger()
	}
	return l
}

func CreateFile(fs afero.Fs, logger *logging.Logger, path string, content string) error {
	logger = safeLogger(logger)
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	PrintWithDots("Creating file: " + lightGreen.Render(path))
	if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
		logger.Error(err)
		return err
	}
	if os.Getenv("NON_INTERACTIVE") != "1" {
		time.Sleep(80 * time.Millisecond)
	}
	return nil
}

func LoadTemplate(templatePath string, data map[string]string) (string, error) {
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

// LoadDockerfileTemplate loads and executes a template with any data type (generalized version of LoadTemplate)
func LoadDockerfileTemplate(templatePath string, data interface{}) (string, error) {
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
	logger = safeLogger(logger)
	// Validate agent name first
	if err := ValidateAgentName(name); err != nil {
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
		"OllamaImageTag": versionpkg.DefaultOllamaImageTag,
	}

	// Load and process the template
	content, err := LoadTemplate(templatePath, templateData)
	if err != nil {
		logger.Error("failed to load workflow template: ", err)
		return err
	}

	return CreateFile(fs, logger, outputPath, content)
}

// GenerateResourceFiles generates resource files for the agent.
func GenerateResourceFiles(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, name string) error {
	logger = safeLogger(logger)
	// Validate agent name first
	if err := ValidateAgentName(name); err != nil {
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

		templatePath := file.Name()
		content, err := LoadTemplate(templatePath, templateData)
		if err != nil {
			logger.Error("failed to process template: ", err)
			return err
		}

		outputPath := filepath.Join(resourceDir, file.Name())
		if err := CreateFile(fs, logger, outputPath, content); err != nil {
			return err
		}
	}

	return nil
}

func GenerateSpecificAgentFile(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
	logger = safeLogger(logger)
	// Validate inputs
	if strings.TrimSpace(mainDir) == "" {
		return fmt.Errorf("base directory cannot be empty")
	}
	// Validate agent name
	if err := ValidateAgentName(agentName); err != nil {
		return err
	}

	headerTemplate := `amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`
	if strings.ToLower(agentName) == "workflow.pkl" {
		headerTemplate = `amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`
	}

	templatePath := agentName + ".pkl"
	templateData := map[string]string{
		"Header": fmt.Sprintf(headerTemplate, schema.SchemaVersion(ctx)),
		"Name":   agentName,
	}

	// Load the template
	content, err := LoadTemplate(templatePath, templateData)
	if err != nil {
		// If the specific template does not exist, fall back to a minimal default template
		// consisting of the header and name. This ensures users can still generate
		// arbitrary agent/resource files without having to embed a dedicated template.
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "file does not exist") {
			logger.Warn("template not found, falling back to default", "template", templatePath)
			content = fmt.Sprintf("%s\nname = \"%s\"\n", templateData["Header"], agentName)
		} else {
			logger.Error("failed to load specific template: ", err)
			return err
		}
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
	return CreateFile(fs, logger, outputPath, content)
}

func GenerateAgent(fs afero.Fs, ctx context.Context, logger *logging.Logger, baseDir, agentName string) error {
	logger = safeLogger(logger)
	// Validate inputs
	if strings.TrimSpace(baseDir) == "" {
		return fmt.Errorf("base directory cannot be empty")
	}
	// Validate agent name
	if err := ValidateAgentName(agentName); err != nil {
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
