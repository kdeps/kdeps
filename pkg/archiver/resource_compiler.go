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

// CompileResources processes .pkl files from the project directory and copies them to the resources directory
func CompileResources(fs afero.Fs, wf *pklWf.Workflow, resourcesDir string, projectDir string, logger *log.Logger) error {
	projectResourcesDir := filepath.Join(projectDir, "resources")

	if err := CheckAndValidatePklFiles(fs, projectResourcesDir, logger); err != nil {
		return err
	}

	// Walk through all files in the project directory
	err := afero.Walk(fs, projectResourcesDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("Error walking project resources directory", "path", projectResourcesDir, "error", err)
			return err
		}

		// Only process .pkl files
		if filepath.Ext(file) == ".pkl" {
			logger.Debug("Processing .pkl", "file", file)
			if err := processResourcePklFiles(fs, file, wf, resourcesDir, logger); err != nil {
				logger.Error("Failed to process .pkl file", "file", file, "error", err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		logger.Error("Error compiling resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
		return err
	}

	logger.Debug("Resources compiled successfully", "resourcesDir", resourcesDir, "projectDir", projectDir)
	return nil
}

// processResourcePklFiles processes a .pkl file and writes modifications to the resources directory
func processResourcePklFiles(fs afero.Fs, file string, wf *pklWf.Workflow, resourcesDir string, logger *log.Logger) error {
	name, version := wf.Name, wf.Version

	readFile, err := fs.Open(file)
	if err != nil {
		logger.Error("Failed to open file", "file", file, "error", err)
		return err
	}
	defer readFile.Close()

	var fileBuffer bytes.Buffer
	scanner := bufio.NewScanner(readFile)

	// Define regex patterns for exec, chat, client with actionID, and id replacement
	idPattern := regexp.MustCompile(`(?i)^\s*id\s*=\s*"(.+)"`)
	// Pattern to capture lines like {resources, resource, responseBody, etc.} with actionID
	actionIDPatterns := map[string]*regexp.Regexp{
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

	inRequiresBlock := false
	var requiresBlockBuffer bytes.Buffer
	var action string

	// Read file line by line
	for scanner.Scan() {
		line := scanner.Text()

		if inRequiresBlock {
			// Check if we've reached the end of the `requires { ... }` block
			if strings.TrimSpace(line) == "}" {
				inRequiresBlock = false
				// Process the accumulated `requires` block
				modifiedBlock := handleRequiresBlock(requiresBlockBuffer.String(), wf)

				// Write the modified block and the closing `}` line
				fileBuffer.WriteString(modifiedBlock)
				fileBuffer.WriteString(line + "\n")
			} else {
				// Continue accumulating the `requires` block lines
				requiresBlockBuffer.WriteString(line + "\n")
			}
			continue
		}

		// Check if the line matches the `id = "value"` pattern
		if idMatch := idPattern.FindStringSubmatch(line); idMatch != nil {
			// Extract the action from the id
			action = idMatch[1]

			// If action doesn't already start with "@", prefix and append name and version
			if !strings.HasPrefix(action, "@") {
				newLine := strings.Replace(line, action, fmt.Sprintf("@%s/%s:%s", name, action, version), 1)
				fileBuffer.WriteString(newLine + "\n")
			} else {
				fileBuffer.WriteString(line + "\n")
			}
		} else {
			// Loop through the actionIDPatterns to find any matching line
			matched := false
			for patternName, pattern := range actionIDPatterns {
				if actionIDMatch := pattern.FindStringSubmatch(line); actionIDMatch != nil {
					matched = true
					// Extract the block type (e.g., resource, responseBody) and the actionID
					blockType := actionIDMatch[1]
					field := actionIDMatch[2]
					var modifiedField string

					// Modify the field for patterns with one or two additional arguments
					if patternName == "responseHeader" || patternName == "env" {
						arg2 := actionIDMatch[3]
						if !strings.HasPrefix(field, "@") {
							modifiedField = fmt.Sprintf("%s(\"@%s/%s:%s\", \"%s\")", blockType, name, field, version, arg2)
						} else {
							modifiedField = line // leave unchanged if already starts with "@"
						}
					} else {
						// Only modify if actionID does not already start with "@"
						if !strings.HasPrefix(field, "@") {
							modifiedField = fmt.Sprintf("%s(\"@%s/%s:%s\")", blockType, name, field, version)
						} else {
							modifiedField = line // leave unchanged if already starts with "@"
						}
					}

					// Replace the original field with the modified one
					newLine := strings.Replace(line, actionIDMatch[0], modifiedField, 1)
					fileBuffer.WriteString(newLine + "\n")
					break
				}
			}

			if !matched {
				// If no patterns matched, check if this is the start of a `requires {` block
				if strings.HasPrefix(strings.TrimSpace(line), "requires {") {
					// Start of a `requires { ... }` block, set flag to accumulate lines
					inRequiresBlock = true
					requiresBlockBuffer.Reset()                  // Clear previous block data if any
					requiresBlockBuffer.WriteString(line + "\n") // Add the opening `requires {` line
				} else {
					// Write the line unchanged if no pattern matches
					fileBuffer.WriteString(line + "\n")
				}
			}
		}
	}

	// Write back to the file if modifications were made
	if scanner.Err() == nil {
		if action == "" {
			err = fmt.Errorf("no valid action found in file: %s", file)
			logger.Error("No valid action found in file", "file", file, "error", err)
			return err
		}
		// Check if the action is prefixed by an agent name (e.g., @abcAgent/fooBar4:2.0.0)
		if strings.HasPrefix(action, "@") {
			// Split by '/' to extract the agent name and action (e.g., @abcAgent/fooBar4:2.0.0 -> abcAgent, fooBar4:2.0.0)
			parts := strings.SplitN(action, "/", 2)
			if len(parts) == 2 {
				name = parts[0][1:] // Update the name to the agent name (remove the '@')
				action = parts[1]   // Update the action (e.g., fooBar4:2.0.0)
			} else {
				return fmt.Errorf("Invalid action format: %s", action)
			}
		}

		// Extract version from the action if it is present (e.g., fooBar4:2.0.0)
		if strings.Contains(action, ":") {
			parts := strings.SplitN(action, ":", 2)
			if len(parts) == 2 {
				action = parts[0]  // Update the action to just the action name (e.g., fooBar4)
				version = parts[1] // Update the version to the version from the action (e.g., 2.0.0)
			} else {
				return fmt.Errorf("Invalid action format: %s", action)
			}
		}

		fname := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
		err = afero.WriteFile(fs, filepath.Join(resourcesDir, fname), fileBuffer.Bytes(), os.FileMode(0644))
		if err != nil {
			logger.Error("Error writing file", "file", fname, "error", err)
			return fmt.Errorf("error writing file: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading file", "file", file, "error", err)
		return err
	}

	logger.Debug("Processed .pkl file", "file", file)
	return nil
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
			return fmt.Errorf("Invalid action format: %s", action)
		}
	}

	// Extract version from the action if it is present (e.g., fooBar4:2.0.0)
	if strings.Contains(action, ":") {
		parts := strings.SplitN(action, ":", 2)
		if len(parts) == 2 {
			action = parts[0]  // Update the action to just the action name (e.g., fooBar4)
			version = parts[1] // Update the version to the version from the action (e.g., 2.0.0)
		} else {
			return fmt.Errorf("Invalid action format: %s", action)
		}
	}

	// Construct the file name using the updated name, action, and version
	fileName := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
	filePath := filepath.Join(resourcesDir, fileName)

	// Write the processed file
	if err := afero.WriteFile(fs, filePath, fileBuffer.Bytes(), os.FileMode(0644)); err != nil {
		return fmt.Errorf("Error writing processed file: %w", err)
	}

	logger.Debug("Processed file written", "file", filePath)
	return nil
}

func CheckAndValidatePklFiles(fs afero.Fs, projectResourcesDir string, logger *log.Logger) error {
	// Check if the project resources directory exists
	if _, err := fs.Stat(projectResourcesDir); err != nil {
		logger.Error("No resource directory found! Exiting!")
		return fmt.Errorf("AI agent needs to have at least 1 resource in the '%s' folder.", projectResourcesDir)
	}

	// Get the list of files in the directory
	files, err := afero.ReadDir(fs, projectResourcesDir)
	if err != nil {
		logger.Error("Error reading resource directory", "error", err)
		return fmt.Errorf("failed to read directory '%s': %v", projectResourcesDir, err)
	}

	// Filter for .pkl files
	var pklFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".pkl" {
			pklFiles = append(pklFiles, filepath.Join(projectResourcesDir, file.Name()))
		}
	}

	// Exit if no .pkl files are found
	if len(pklFiles) == 0 {
		logger.Error("No .pkl files found in the directory! Exiting!")
		return fmt.Errorf("No .pkl files found in the '%s' folder.", projectResourcesDir)
	}

	// Validate each .pkl file
	for _, pklFile := range pklFiles {
		logger.Debug("Validating .pkl file", "file", pklFile)
		if err := enforcer.EnforcePklTemplateAmendsRules(fs, pklFile, logger); err != nil {
			logger.Error("Validation failed for .pkl file", "file", pklFile, "error", err)
			return fmt.Errorf("validation failed for '%s': %v", pklFile, err)
		}
	}

	logger.Debug("All .pkl files validated successfully!")
	return nil
}
