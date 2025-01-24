package archiver

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/logging"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// CompileResources processes .pkl files from the project directory and copies them to the resources directory.
func CompileResources(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, resourcesDir string, projectDir string, logger *logging.Logger) error {
	projectResourcesDir := filepath.Join(projectDir, "resources")

	if err := CheckAndValidatePklFiles(fs, ctx, projectResourcesDir, logger); err != nil {
		return err
	}

	// Walk through all files in the project directory
	err := afero.Walk(fs, projectResourcesDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("error walking project resources directory", "path", projectResourcesDir, "error", err)
			return err
		}

		// Only process .pkl files
		if filepath.Ext(file) == ".pkl" {
			logger.Debug("processing .pkl", "file", file)
			if err := processResourcePklFiles(fs, file, wf, resourcesDir, logger); err != nil {
				logger.Error("failed to process .pkl file", "file", file, "error", err)
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("error compiling resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
		return err
	}

	logger.Debug("resources compiled successfully", "resourcesDir", resourcesDir, "projectDir", projectDir)
	return nil
}

// processResourcePklFiles processes a .pkl file and writes modifications to the resources directory.
func processResourcePklFiles(fs afero.Fs, file string, wf pklWf.Workflow, resourcesDir string, logger *logging.Logger) error {
	name, version := wf.GetName(), wf.GetVersion()

	readFile, err := fs.Open(file)
	if err != nil {
		logger.Error("failed to open file", "file", file, "error", err)
		return err
	}
	defer readFile.Close()

	var fileBuffer bytes.Buffer
	scanner := bufio.NewScanner(readFile)

	// Define regex patterns for exec, chat, client with actionID, and id replacement
	idPattern := regexp.MustCompile(`(?i)^\s*actionID\s*=\s*"(.+)"`)
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
		"exitCode":       regexp.MustCompile(`(?i)(exitCode)\("(.+)"\)`),
		"file":           regexp.MustCompile(`(?i)(file)\("(.+)"\)`),
	}

	inRequiresBlock := false
	var requiresBlockBuffer bytes.Buffer
	var action string

	// Read file line by line
	for scanner.Scan() {
		line := scanner.Text()

		if inRequiresBlock {
			// Check if we've reached the end of the requires { ... } block
			if strings.TrimSpace(line) == "}" {
				inRequiresBlock = false
				// Process the accumulated requires block
				modifiedBlock := handleRequiresBlock(requiresBlockBuffer.String(), wf)

				// Write the modified block and the closing } line
				fileBuffer.WriteString(modifiedBlock)
				fileBuffer.WriteString(line + "\n")
			} else {
				// Continue accumulating the requires block lines
				requiresBlockBuffer.WriteString(line + "\n")
			}
			continue
		}

		// Check if the line matches the ID = "value" pattern
		if idMatch := idPattern.FindStringSubmatch(line); idMatch != nil {
			// Extract the action from the id
			action = idMatch[1]
			// If action doesn't already start with "@", prefix and append name and version
			if !strings.HasPrefix(action, "@") {
				newLine := strings.ReplaceAll(line, action, fmt.Sprintf("@%s/%s:%s", name, action, version))
				fileBuffer.WriteString(newLine + "\n")
			} else {
				fileBuffer.WriteString(line + "\n")
			}
		} else {
			// Loop through the actionIDPatterns to find any matching line
			matched := false
			for patternName, pattern := range actionIDPatterns {
				// Find all matches in the line for the current pattern
				actionIDMatches := pattern.FindAllStringSubmatch(line, -1)
				if len(actionIDMatches) > 0 {
					matched = true
					// Iterate over each match in the line
					for _, actionIDMatch := range actionIDMatches {
						if !strings.HasPrefix(actionIDMatch[2], "@") {
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

							// Replace all occurrences of the original field with the modified one
							line = strings.ReplaceAll(line, actionIDMatch[0], modifiedField)
						}
					}
				}
			}

			if matched {
				// Write the modified line after processing all matches
				fileBuffer.WriteString(line + "\n")
			} else {
				// If no patterns matched, check if this is the start of a requires { block
				if strings.HasPrefix(strings.TrimSpace(line), "requires {") {
					// Start of a requires { ... } block, set flag to accumulate lines
					inRequiresBlock = true
					requiresBlockBuffer.Reset()                  // Clear previous block data if any
					requiresBlockBuffer.WriteString(line + "\n") // Add the opening requires { line
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
			logger.Error("no valid action found in file", "file", file, "error", err)
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
				return fmt.Errorf("invalid action format: %s", action)
			}
		}

		// Extract version from the action if it is present (e.g., fooBar4:2.0.0)
		if strings.Contains(action, ":") {
			parts := strings.SplitN(action, ":", 2)
			if len(parts) == 2 {
				action = parts[0]  // Update the action to just the action name (e.g., fooBar4)
				version = parts[1] // Update the version to the version from the action (e.g., 2.0.0)
			} else {
				return fmt.Errorf("invalid action format: %s", action)
			}
		}

		fname := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
		err = afero.WriteFile(fs, filepath.Join(resourcesDir, fname), fileBuffer.Bytes(), os.FileMode(0o644))
		if err != nil {
			logger.Error("error writing file", "file", fname, "error", err)
			return fmt.Errorf("error writing file: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("error reading file", "file", file, "error", err)
		return err
	}

	logger.Debug("processed .pkl file", "file", file)
	return nil
}

func CheckAndValidatePklFiles(fs afero.Fs, ctx context.Context, projectResourcesDir string, logger *logging.Logger) error {
	// Check if the project resources directory exists
	if _, err := fs.Stat(projectResourcesDir); err != nil {
		logger.Error("no resource directory found! Exiting!")
		return fmt.Errorf("the AI agent needs to have at least 1 resource in the '%s' folder", projectResourcesDir)
	}

	// Get the list of files in the directory
	files, err := afero.ReadDir(fs, projectResourcesDir)
	if err != nil {
		logger.Error("error reading resource directory", "error", err)
		return fmt.Errorf("failed to read directory '%s': %w", projectResourcesDir, err)
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
		logger.Error("no .pkl files found in the directory! Exiting!")
		return fmt.Errorf("no .pkl files found in the '%s' folder", projectResourcesDir)
	}

	// Validate each .pkl file
	for _, pklFile := range pklFiles {
		logger.Debug("validating .pkl file", "file", pklFile)
		if err := enforcer.EnforcePklTemplateAmendsRules(fs, ctx, pklFile, logger); err != nil {
			logger.Error("validation failed for .pkl file", "file", pklFile, "error", err)
			return fmt.Errorf("validation failed for '%s': %w", pklFile, err)
		}
	}

	logger.Debug("all .pkl files validated successfully!")
	return nil
}
