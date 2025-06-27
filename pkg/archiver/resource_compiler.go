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
	"github.com/kdeps/kdeps/pkg/messages"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

var (
	idPattern       = regexp.MustCompile(`(?i)^\s*actionID\s*=\s*"(.+)"`)
	actionIDRegex   = regexp.MustCompile(`(?i)\b(resources|resource|responseBody|responseHeader|stderr|stdout|env|response|prompt|exitCode|file)\s*\(\s*"([^"]+)"\s*(?:,\s*"([^"]+)")?\s*\)`)
	requiresPattern = regexp.MustCompile(`^\s*requires\s*{`)
)

// CompileResources processes .pkl files and copies them to resources directory.
func CompileResources(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, resourcesDir string, projectDir string, logger *logging.Logger) error {
	projectResourcesDir := filepath.Join(projectDir, "resources")

	if err := ValidatePklResources(fs, ctx, projectResourcesDir, logger); err != nil {
		return err
	}

	err := afero.Walk(fs, projectResourcesDir, PklFileProcessor(fs, wf, resourcesDir, logger))
	if err != nil {
		logger.Error("error compiling resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
	}

	logger.Debug(messages.MsgResourcesCompiled, "resourcesDir", resourcesDir, "projectDir", projectDir)
	return err
}

func PklFileProcessor(fs afero.Fs, wf pklWf.Workflow, resourcesDir string, logger *logging.Logger) filepath.WalkFunc {
	return func(file string, info os.FileInfo, err error) error {
		if err != nil || filepath.Ext(file) != ".pkl" || info.IsDir() {
			return err
		}

		logger.Debug(messages.MsgProcessingPkl, "file", file)
		if err := ProcessPklFile(fs, file, wf, resourcesDir, logger); err != nil {
			logger.Error("failed to process .pkl file", "file", file, "error", err)
		}
		return nil
	}
}

func ProcessPklFile(fs afero.Fs, file string, wf pklWf.Workflow, resourcesDir string, logger *logging.Logger) error {
	fileBuffer, action, err := ProcessFileContent(fs, file, wf, logger)
	if err != nil || action == "" {
		return fmt.Errorf("no valid action found in file: %s", file)
	}

	name, version := ParseActionID(action, wf.GetName(), wf.GetVersion())
	fname := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
	targetPath := filepath.Join(resourcesDir, fname)

	if err := afero.WriteFile(fs, targetPath, fileBuffer.Bytes(), 0o644); err != nil {
		logger.Error("error writing file", "file", fname, "error", err)
		return fmt.Errorf("error writing file: %w", err)
	}

	logger.Debug(messages.MsgProcessedPklFile, "file", file)
	return nil
}

func ProcessFileContent(fs afero.Fs, file string, wf pklWf.Workflow, logger *logging.Logger) (*bytes.Buffer, string, error) {
	content, err := afero.ReadFile(fs, file)
	if err != nil {
		logger.Error("failed to read file", "file", file, "error", err)
		return nil, "", err
	}

	var (
		fileBuffer      bytes.Buffer
		inRequiresBlock bool
		requiresBuffer  bytes.Buffer
		currentAction   string
		requiresWritten bool // Tracks if a 'requires' block is already processed
		skippingBlock   bool // Tracks if we're currently skipping a redundant requires block
		scanner         = bufio.NewScanner(bytes.NewReader(content))
		name            = wf.GetName()
		version         = wf.GetVersion()
	)

	for scanner.Scan() {
		line := scanner.Text()

		// If we're skipping a redundant requires block, only look for the closing brace
		if skippingBlock {
			if strings.TrimSpace(line) == "}" {
				skippingBlock = false
			}
			continue
		}

		// Check if this is a redundant requires block
		if requiresPattern.MatchString(line) && requiresWritten {
			skippingBlock = true
			continue
		}

		if HandleRequiresSection(&line, &inRequiresBlock, wf, &requiresBuffer, &fileBuffer) {
			if !inRequiresBlock {
				requiresWritten = true // Mark requires block as written
			}
			continue
		}

		line, actionModified := ProcessLine(line, name, version)
		if actionModified != "" {
			currentAction = actionModified
		}
		fileBuffer.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		logger.Error("error reading file", "file", file, "error", err)
		return nil, "", err
	}

	// Add any remaining `requires` block content
	if requiresBuffer.Len() > 0 && !requiresWritten {
		fileBuffer.WriteString(HandleRequiresBlock(requiresBuffer.String(), wf))
	}

	return &fileBuffer, currentAction, nil
}

func HandleRequiresSection(line *string, inBlock *bool, wf pklWf.Workflow, requiresBuf, fileBuf *bytes.Buffer) bool {
	switch {
	case *inBlock:
		if strings.TrimSpace(*line) == "}" {
			*inBlock = false
			fileBuf.WriteString(HandleRequiresBlock(requiresBuf.String(), wf))
			requiresBuf.Reset() // Clear the buffer after processing
			fileBuf.WriteString(*line + "\n")
		} else {
			requiresBuf.WriteString(*line + "\n")
		}
		return true
	case requiresPattern.MatchString(*line):
		// Skip if this requires block was already processed
		if requiresBuf.Len() > 0 {
			return true
		}
		*inBlock = true
		requiresBuf.WriteString(*line + "\n")
		return true
	}
	return false
}

func ProcessLine(line, name, version string) (string, string) {
	if idMatch := idPattern.FindStringSubmatch(line); idMatch != nil {
		return ProcessActionIDLine(line, idMatch[1], name, version), idMatch[1]
	}
	return ProcessActionPatterns(line, name, version), ""
}

func ProcessActionIDLine(line, action, name, version string) string {
	if !strings.HasPrefix(action, "@") {
		return strings.ReplaceAll(line, action, fmt.Sprintf("@%s/%s:%s", name, action, version))
	}
	return line
}

func ProcessActionPatterns(line, name, version string) string {
	return actionIDRegex.ReplaceAllStringFunc(line, func(match string) string {
		parts := actionIDRegex.FindStringSubmatch(match)
		if strings.HasPrefix(parts[2], "@") {
			return match
		}

		newID := fmt.Sprintf("@%s/%s:%s", name, parts[2], version)
		switch parts[1] {
		case "responseHeader", "env":
			return fmt.Sprintf("%s(\"%s\", \"%s\")", parts[1], newID, parts[3])
		default:
			return fmt.Sprintf("%s(\"%s\")", parts[1], newID)
		}
	})
}

func ParseActionID(action, defaultName, defaultVersion string) (string, string) {
	name, version := defaultName, defaultVersion
	if strings.HasPrefix(action, "@") {
		parts := strings.SplitN(action[1:], "/", 2)
		if len(parts) > 1 {
			name, action = parts[0], parts[1]
		}
	}

	if versionParts := strings.SplitN(action, ":", 2); len(versionParts) > 1 {
		_, version = versionParts[0], versionParts[1]
	}
	return name, version
}

func ValidatePklResources(fs afero.Fs, ctx context.Context, dir string, logger *logging.Logger) error {
	if _, err := fs.Stat(dir); err != nil {
		logger.Error("resource directory not found", "path", dir)
		return fmt.Errorf("missing resource directory: %s", dir)
	}

	pklFiles, err := CollectPklFiles(fs, dir)
	if err != nil || len(pklFiles) == 0 {
		logger.Error("no .pkl files found", "directory", dir)
		return fmt.Errorf("no .pkl files in %s", dir)
	}

	for _, file := range pklFiles {
		if err := enforcer.EnforcePklTemplateAmendsRules(fs, ctx, file, logger); err != nil {
			return fmt.Errorf("validation failed for %s: %w", file, err)
		}
	}
	return nil
}

func CollectPklFiles(fs afero.Fs, dir string) ([]string, error) {
	files, err := afero.ReadDir(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	var pklFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".pkl" {
			pklFiles = append(pklFiles, filepath.Join(dir, f.Name()))
		}
	}
	return pklFiles, nil
}
