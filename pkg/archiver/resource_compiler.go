package archiver

import (
	"bufio"
	"bytes"
	"fmt"
	"kdeps/pkg/enforcer"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// CompileResources processes .pkl files from the project directory and copies them to the resources directory.
func CompileResources(fs afero.Fs, wf *pklWf.Workflow, resourcesDir string, projectDir string, logger *log.Logger) error {
	projectResourcesDir := filepath.Join(projectDir, "resources")

	if err := ValidatePklFiles(fs, projectResourcesDir, logger); err != nil {
		return err
	}

	err := afero.Walk(fs, projectResourcesDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return logAndReturnError(logger, "Error walking project resources directory", projectResourcesDir, err)
		}

		// Process only .pkl files
		if filepath.Ext(filePath) == ".pkl" {
			logger.Debug("Processing .pkl file", "file", filePath)
			if err := processPklFile(fs, filePath, wf, resourcesDir, logger); err != nil {
				return logAndReturnError(logger, "Failed to process .pkl file", filePath, err)
			}
		}
		return nil
	})

	if err != nil {
		return logAndReturnError(logger, "Error compiling resources", fmt.Sprintf("%s (project: %s)", resourcesDir, projectDir), err)
	}

	logger.Debug("Resources compiled successfully", "resourcesDir", resourcesDir, "projectDir", projectDir)
	return nil
}

// processPklFile processes a single .pkl file and writes modifications to the resources directory.
func processPklFile(fs afero.Fs, filePath string, wf *pklWf.Workflow, resourcesDir string, logger *log.Logger) error {
	name, version := wf.Name, wf.Version

	readFile, err := fs.Open(filePath)
	if err != nil {
		return logAndReturnError(logger, "Failed to open file", filePath, err)
	}
	defer readFile.Close()

	fileBuffer := &bytes.Buffer{}
	scanner := bufio.NewScanner(readFile)

	// Regex patterns for replacing fields
	actionIDPatterns := getActionIDPatterns()
	idPattern := regexp.MustCompile(`(?i)^\s*id\s*=\s*"(.+)"`)

	var (
		inRequiresBlock     bool
		requiresBlockBuffer bytes.Buffer
		action              string
	)

	for scanner.Scan() {
		line := scanner.Text()

		if inRequiresBlock {
			if handleRequiresBlockEnd(line, &inRequiresBlock, &requiresBlockBuffer, fileBuffer, wf) {
				continue
			}
		} else if idMatch := idPattern.FindStringSubmatch(line); idMatch != nil {
			action = handleIDPatternMatch(idMatch, line, fileBuffer, wf)
		} else if matchFound := handleActionIDPatterns(line, actionIDPatterns, fileBuffer, wf); matchFound {
			continue
		} else if strings.HasPrefix(strings.TrimSpace(line), "requires {") {
			inRequiresBlock = true
			requiresBlockBuffer.Reset()
			requiresBlockBuffer.WriteString(line + "\n")
		} else {
			fileBuffer.WriteString(line + "\n")
		}
	}

	if scanner.Err() != nil {
		return logAndReturnError(logger, "Error reading file", filePath, scanner.Err())
	}

	if action == "" {
		return logAndReturnError(logger, "No valid action found in file", filePath, nil)
	}

	return writeProcessedFile(fs, fileBuffer, resourcesDir, name, action, version, logger)
}

// ValidatePklFiles checks and validates the .pkl files in the resources directory.
func ValidatePklFiles(fs afero.Fs, projectResourcesDir string, logger *log.Logger) error {
	if _, err := fs.Stat(projectResourcesDir); err != nil {
		return logAndReturnError(logger, "No resource directory found", projectResourcesDir, err)
	}

	files, err := afero.ReadDir(fs, projectResourcesDir)
	if err != nil {
		return logAndReturnError(logger, "Error reading resource directory", projectResourcesDir, err)
	}

	pklFiles := filterPklFiles(files, projectResourcesDir)
	if len(pklFiles) == 0 {
		return fmt.Errorf("no .pkl files found in the '%s' folder", projectResourcesDir)
	}

	for _, pklFile := range pklFiles {
		logger.Debug("Validating .pkl file", "file", pklFile)
		if err := enforcer.EnforcePklTemplateAmendsRules(fs, pklFile, logger); err != nil {
			return logAndReturnError(logger, "Validation failed for .pkl file", pklFile, err)
		}
	}

	logger.Debug("All .pkl files validated successfully!")
	return nil
}

// Utility and helper functions:

// logAndReturnError logs the error and returns a formatted error.
func logAndReturnError(logger *log.Logger, message, context string, err error) error {
	if err != nil {
		logger.Error(message, "context", context, "error", err)
	} else {
		logger.Error(message, "context", context)
	}
	return fmt.Errorf("%s: %v", message, err)
}

// filterPklFiles filters the list of files to only include .pkl files.
func filterPklFiles(files []os.FileInfo, dir string) []string {
	var pklFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".pkl" {
			pklFiles = append(pklFiles, filepath.Join(dir, file.Name()))
		}
	}
	return pklFiles
}

// getActionIDPatterns returns the regex patterns for action IDs.
func getActionIDPatterns() map[string]*regexp.Regexp {
	return map[string]*regexp.Regexp{
		"resources":      regexp.MustCompile(`(?i)(resources)\["(.+)"\]`),
		"resource":       regexp.MustCompile(`(?i)(resource)\("(.+)"\)`),
		"responseBody":   regexp.MustCompile(`(?i)(responseBody)\("(.+)"\)`),
		"responseHeader": regexp.MustCompile(`(?i)(responseHeader)\("(.+)",\s*"(.+)"\)`),
		"stderr":         regexp.MustCompile(`(?i)(stderr)\("(.+)"\)`),
		"stdout":         regexp.MustCompile(`(?i)(stdout)\("(.+)"\)`),
		"env":            regexp.MustCompile(`(?i)(env)\("(.+)",\s*"(.+)"\)`),
		"response":       regexp.MustCompile(`(?i)(response)\("(.+)"\)`),
		"prompt":         regexp.MustCompile(`(?i)(prompt)\("(.+)"\)`),
	}
}

// handleIDPatternMatch processes a line with a matched "id = " pattern.
func handleIDPatternMatch(idMatch []string, line string, buffer *bytes.Buffer, wf *pklWf.Workflow) string {
	action := idMatch[1]
	if !strings.HasPrefix(action, "@") {
		newLine := fmt.Sprintf(`id = "@%s/%s:%s"`, wf.Name, action, wf.Version)
		buffer.WriteString(newLine + "\n")
	} else {
		buffer.WriteString(line + "\n")
	}
	return action
}

// handleActionIDPatterns processes action ID pattern matches and returns whether a match was found.
func handleActionIDPatterns(line string, patterns map[string]*regexp.Regexp, buffer *bytes.Buffer, wf *pklWf.Workflow) bool {
	for patternName, pattern := range patterns {
		if actionIDMatch := pattern.FindStringSubmatch(line); actionIDMatch != nil {
			modifiedLine := modifyActionIDLine(actionIDMatch, patternName, line, wf)
			buffer.WriteString(modifiedLine + "\n")
			return true
		}
	}
	return false
}

// modifyActionIDLine modifies actionID line based on the matching pattern.
func modifyActionIDLine(actionIDMatch []string, patternName, line string, wf *pklWf.Workflow) string {
	blockType := actionIDMatch[1]
	field := actionIDMatch[2]
	if patternName == "responseHeader" || patternName == "env" {
		arg2 := actionIDMatch[3]
		if !strings.HasPrefix(field, "@") {
			return fmt.Sprintf(`%s("@%s/%s:%s", "%s")`, blockType, wf.Name, field, wf.Version, arg2)
		}
	} else if !strings.HasPrefix(field, "@") {
		return fmt.Sprintf(`%s("@%s/%s:%s")`, blockType, wf.Name, field, wf.Version)
	}
	return line // Return the original line if no modification is needed
}

// handleRequiresBlockEnd checks if a "requires { ... }" block has ended and processes it accordingly.
func handleRequiresBlockEnd(line string, inRequiresBlock *bool, requiresBlockBuffer *bytes.Buffer, fileBuffer *bytes.Buffer, wf *pklWf.Workflow) bool {
	if strings.TrimSpace(line) == "}" {
		*inRequiresBlock = false
		modifiedBlock := handleRequiresBlock(requiresBlockBuffer.String(), wf)
		fileBuffer.WriteString(modifiedBlock)
		fileBuffer.WriteString(line + "\n")
		return true
	}
	requiresBlockBuffer.WriteString(line + "\n")
	return false
}

// writeProcessedFile writes the processed .pkl content to the resources directory.
func writeProcessedFile(fs afero.Fs, fileBuffer *bytes.Buffer, resourcesDir, name, action, version string, logger *log.Logger) error {
	// Check if the action is prefixed by an agent name (e.g., @abcAgent/fooBar4:2.0.0)
	if strings.HasPrefix(action, "@") {
		// Split by '/' to extract the agent name and action (e.g., @abcAgent/fooBar4:2.0.0 -> abcAgent, fooBar4:2.0.0)
		parts := strings.SplitN(action, "/", 2)
		if len(parts) == 2 {
			name = parts[0][1:] // Update the name to the agent name (remove the '@')
			action = parts[1]   // Update the action (e.g., fooBar4:2.0.0)
		} else {
			return logAndReturnError(logger, "Invalid action format", action, nil)
		}
	}

	// Extract version from the action if it is present (e.g., fooBar4:2.0.0)
	if strings.Contains(action, ":") {
		parts := strings.SplitN(action, ":", 2)
		if len(parts) == 2 {
			action = parts[0]  // Update the action to just the action name (e.g., fooBar4)
			version = parts[1] // Update the version to the version from the action (e.g., 2.0.0)
		} else {
			return logAndReturnError(logger, "Invalid action format", action, nil)
		}
	}

	// Construct the file name using the updated name, action, and version
	fileName := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
	filePath := filepath.Join(resourcesDir, fileName)

	// Write the processed file
	if err := afero.WriteFile(fs, filePath, fileBuffer.Bytes(), os.FileMode(0644)); err != nil {
		return logAndReturnError(logger, "Error writing processed file", filePath, err)
	}

	logger.Debug("Processed file written", "file", filePath)
	return nil
}
