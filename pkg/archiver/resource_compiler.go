package archiver

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/enforcer"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

var (
	idPattern       = regexp.MustCompile(`(?i)^\s*actionID\s*=\s*"(.+)"`)
	requiresPattern = regexp.MustCompile(`(?i)^\s*requires\s*=\s*{`)
)

// CompileResources processes .pkl files and copies them to resources directory.
func CompileResources(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, resourcesDir string, projectDir string, logger *logging.Logger) error {
	projectResourcesDir := filepath.Join(projectDir, "resources")

	if err := ValidatePklResources(fs, ctx, projectResourcesDir, logger); err != nil {
		return err
	}

	err := afero.Walk(fs, projectResourcesDir, pklFileProcessor(fs, wf, resourcesDir, logger))
	if err != nil {
		logger.Error("error compiling resources", "resourcesDir", resourcesDir, "projectDir", projectDir, "error", err)
		return err
	}

	// Evaluate all compiled PKL files in the resources directory to test for any problems
	logger.Debug("evaluating compiled resource PKL files")
	if err := evaluator.EvaluateAllPklFilesInDirectory(fs, ctx, resourcesDir, logger); err != nil {
		logger.Error("error evaluating resource PKL files", "resourcesDir", resourcesDir, "error", err)
		return err
	}

	logger.Debug(messages.MsgResourcesCompiled, "resourcesDir", resourcesDir, "projectDir", projectDir)
	return nil
}

func pklFileProcessor(fs afero.Fs, wf pklWf.Workflow, resourcesDir string, logger *logging.Logger) filepath.WalkFunc {
	return func(file string, info os.FileInfo, err error) error {
		if err != nil || filepath.Ext(file) != ".pkl" || info.IsDir() {
			return err
		}

		logger.Debug(messages.MsgProcessingPkl, "file", file)
		if err := processPklFile(fs, file, wf, resourcesDir, logger); err != nil {
			logger.Error("failed to process .pkl file", "file", file, "error", err)
		}
		return nil
	}
}

func processPklFile(fs afero.Fs, file string, wf pklWf.Workflow, resourcesDir string, logger *logging.Logger) error {
	fileBuffer, action, agentsToCopyAll, err := processFileContent(fs, file, wf, logger)
	if err != nil || action == "" {
		return fmt.Errorf("no valid action found in file: %s", file)
	}

	// Use agent.PklResourceReader to resolve the action ID
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		return fmt.Errorf("failed to initialize agent reader: %w", err)
	}

	// Create URI for agent ID resolution
	query := url.Values{}
	query.Set("op", "resolve")
	query.Set("agent", wf.GetAgentID())
	query.Set("version", wf.GetVersion())
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + action,
		RawQuery: query.Encode(),
	}

	resolvedIDBytes, err := agentReader.Read(uri)
	if err != nil {
		return fmt.Errorf("failed to resolve action ID: %w", err)
	}
	resolvedID := string(resolvedIDBytes)

	// Extract name and version from resolved ID for filename
	name, version := extractNameVersionFromResolvedID(resolvedID, wf.GetAgentID(), wf.GetVersion())
	fname := fmt.Sprintf("%s_%s-%s.pkl", name, action, version)
	targetPath := filepath.Join(resourcesDir, fname)

	if err := afero.WriteFile(fs, targetPath, fileBuffer.Bytes(), 0o644); err != nil {
		logger.Error("error writing file", "file", fname, "error", err)
		return fmt.Errorf("error writing file: %w", err)
	}

	// Copy all resources from agents specified in requires block
	for _, agentName := range agentsToCopyAll {
		if err := copyAllResourcesFromAgent(fs, agentName, wf, resourcesDir, logger); err != nil {
			logger.Warn("failed to copy all resources from agent", "agent", agentName, "error", err)
			// Continue processing other agents
		}
	}

	logger.Debug(messages.MsgProcessedPklFile, "file", file)
	return nil
}

func processFileContent(fs afero.Fs, file string, wf pklWf.Workflow, logger *logging.Logger) (*bytes.Buffer, string, []string, error) {
	content, err := afero.ReadFile(fs, file)
	if err != nil {
		logger.Error("failed to read file", "file", file, "error", err)
		return nil, "", nil, err
	}

	// Initialize agent reader for ID resolution
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to initialize agent reader: %w", err)
	}

	var (
		fileBuffer      bytes.Buffer
		inRequiresBlock bool
		requiresBuffer  bytes.Buffer
		currentAction   string
		requiresWritten bool
		agentsToCopyAll []string
		scanner         = bufio.NewScanner(bytes.NewReader(content))
	)

	for scanner.Scan() {
		line := scanner.Text()

		if requiresPattern.MatchString(line) && requiresWritten {
			continue // Skip redundant requires blocks
		}

		if handleRequiresSection(&line, &inRequiresBlock, wf, &requiresBuffer, &fileBuffer, agentReader, &agentsToCopyAll) {
			if !inRequiresBlock {
				requiresWritten = true
			}
			continue
		}

		line, actionModified := processLineWithAgentReader(line, wf, agentReader)
		if actionModified != "" {
			currentAction = actionModified
		}
		fileBuffer.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		logger.Error("error reading file", "file", file, "error", err)
		return nil, "", nil, err
	}

	// Add any remaining `requires` block content
	if requiresBuffer.Len() > 0 && !requiresWritten {
		processedRequires, additionalAgents := processRequiresBlockWithAgentReader(requiresBuffer.String(), wf, agentReader)
		fileBuffer.WriteString(processedRequires)
		agentsToCopyAll = append(agentsToCopyAll, additionalAgents...)
	}

	return &fileBuffer, currentAction, agentsToCopyAll, nil
}

func handleRequiresSection(line *string, inBlock *bool, wf pklWf.Workflow, requiresBuf, fileBuf *bytes.Buffer, agentReader *agent.PklResourceReader, agentsToCopyAll *[]string) bool {
	switch {
	case *inBlock:
		if strings.TrimSpace(*line) == "}" {
			*inBlock = false
			processedRequires, additionalAgents := processRequiresBlockWithAgentReader(requiresBuf.String(), wf, agentReader)
			fileBuf.WriteString(processedRequires)
			*agentsToCopyAll = append(*agentsToCopyAll, additionalAgents...)
			requiresBuf.Reset()
			fileBuf.WriteString(*line + "\n")
		} else {
			requiresBuf.WriteString(*line + "\n")
		}
		return true
	case requiresPattern.MatchString(*line):
		if requiresBuf.Len() > 0 {
			return true
		}
		*inBlock = true
		requiresBuf.WriteString(*line + "\n")
		return true
	}
	return false
}

func processLineWithAgentReader(line string, wf pklWf.Workflow, agentReader *agent.PklResourceReader) (string, string) {
	if idMatch := idPattern.FindStringSubmatch(line); idMatch != nil {
		resolvedID := resolveActionIDWithAgentReader(idMatch[1], wf, agentReader)
		return strings.ReplaceAll(line, idMatch[1], resolvedID), idMatch[1]
	}
	return line, ""
}

// processRequiresBlockWithAgentReader processes the requires block and returns the processed block string and a list of agent names for 'all resources' copying
func processRequiresBlockWithAgentReader(blockContent string, wf pklWf.Workflow, agentReader *agent.PklResourceReader) (string, []string) {
	lines := strings.Split(blockContent, "\n")
	modifiedLines := make([]string, 0, len(lines))
	agentsToCopyAll := []string{}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			modifiedLines = append(modifiedLines, trimmedLine)
			continue
		}

		// Detect quoted action names
		if strings.HasPrefix(trimmedLine, `"`) && strings.HasSuffix(trimmedLine, `"`) {
			value := strings.Trim(trimmedLine, `"`)
			if value == "" {
				modifiedLines = append(modifiedLines, `""`)
				continue
			}

			if isActionID(value) {
				// Use agent reader to resolve the value
				resolvedValue := resolveActionIDWithAgentReader(value, wf, agentReader)
				modifiedLines = append(modifiedLines, fmt.Sprintf(`"%s"`, resolvedValue))
			} else {
				// Keep non-action quoted strings as-is
				modifiedLines = append(modifiedLines, trimmedLine)
			}
			continue
		}

		// Detect unquoted agent names (for all resources)
		if isAgentName(trimmedLine) {
			agentsToCopyAll = append(agentsToCopyAll, trimmedLine)
			modifiedLines = append(modifiedLines, trimmedLine)
			continue
		}

		// Retain other unquoted lines
		modifiedLines = append(modifiedLines, trimmedLine)
	}

	return strings.Join(modifiedLines, "\n"), agentsToCopyAll
}

// isActionID checks if a string looks like an action ID
func isActionID(value string) bool {
	// Action IDs should be simple identifiers without special characters
	// They should not contain slashes, equals, or other special syntax
	if value == "" {
		return false
	}

	// Skip if it looks like a comment or configuration
	if strings.HasPrefix(value, "#") || strings.Contains(value, "=") {
		return false
	}

	// Skip if it contains slashes (likely a path or URL)
	if strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return false
	}

	// Skip if it contains spaces or other special characters
	if strings.ContainsAny(value, " \t\n\r") {
		return false
	}

	// Skip if it ends with common config suffixes
	if strings.HasSuffix(value, "_value") || strings.HasSuffix(value, "_config") ||
		strings.HasSuffix(value, "_setting") || strings.HasSuffix(value, "_option") {
		return false
	}

	// Skip if it contains multiple underscores (likely a config key)
	if strings.Count(value, "_") > 1 {
		return false
	}

	// Action IDs should be simple alphanumeric with possible single underscore/hyphen
	// and should be camelCase or PascalCase style
	actionIDPattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*([_-][a-zA-Z0-9]+)*$`)
	return actionIDPattern.MatchString(value)
}

func resolveActionIDWithAgentReader(actionID string, wf pklWf.Workflow, agentReader *agent.PklResourceReader) string {
	// If the actionID is already in canonical form (@agent/action:version), return it
	if strings.HasPrefix(actionID, "@") {
		return actionID
	}

	// Strip any version from the action ID (e.g., "myAction:0.3.0" -> "myAction")
	// The workflow version should take precedence
	actionName := actionID
	if strings.Contains(actionID, ":") {
		actionName = strings.Split(actionID, ":")[0]
	}

	// Create URI for agent ID resolution
	query := url.Values{}
	query.Set("op", "resolve")
	query.Set("agent", wf.GetAgentID())
	query.Set("version", wf.GetVersion())
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + actionName,
		RawQuery: query.Encode(),
	}

	resolvedIDBytes, err := agentReader.Read(uri)
	if err != nil {
		// Fallback to default resolution if agent reader fails
		return fmt.Sprintf("@%s/%s:%s", wf.GetAgentID(), actionName, wf.GetVersion())
	}

	return string(resolvedIDBytes)
}

func extractNameVersionFromResolvedID(resolvedID, defaultName, defaultVersion string) (string, string) {
	if !strings.HasPrefix(resolvedID, "@") {
		return defaultName, defaultVersion
	}

	parts := strings.SplitN(resolvedID[1:], "/", 2)
	if len(parts) != 2 {
		return defaultName, defaultVersion
	}

	name := parts[0]
	actionVersion := parts[1]

	versionParts := strings.SplitN(actionVersion, ":", 2)
	if len(versionParts) == 2 {
		return name, versionParts[1]
	}

	return name, defaultVersion
}

func ValidatePklResources(fs afero.Fs, ctx context.Context, dir string, logger *logging.Logger) error {
	if _, err := fs.Stat(dir); err != nil {
		logger.Error("resource directory not found", "path", dir)
		return fmt.Errorf("missing resource directory: %s", dir)
	}

	pklFiles, err := collectPklFiles(fs, dir)
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

func collectPklFiles(fs afero.Fs, dir string) ([]string, error) {
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

// isAgentName checks if a string looks like an agent name (unquoted, simple identifier)
func isAgentName(value string) bool {
	if value == "" {
		return false
	}
	// Agent names should be simple alphanumeric, no quotes, no special chars
	agentNamePattern := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	return agentNamePattern.MatchString(value)
}

// copyAllResourcesFromAgent copies all resources from a specified agent
func copyAllResourcesFromAgent(fs afero.Fs, agentName string, wf pklWf.Workflow, resourcesDir string, logger *logging.Logger) error {
	// Initialize agent reader
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		return fmt.Errorf("failed to initialize agent reader: %w", err)
	}

	// Create URI for agent resource listing
	query := url.Values{}
	query.Set("op", "list")
	query.Set("agent", wf.GetAgentID())
	query.Set("version", wf.GetVersion())
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + agentName,
		RawQuery: query.Encode(),
	}

	_, err = agentReader.Read(uri)
	if err != nil {
		logger.Warn("failed to list resources from agent", "agent", agentName, "error", err)
		// Continue anyway - this might be a development scenario
	}

	logger.Debug("marked all resources for copying from agent", "agent", agentName)

	// --- Copy data/ folder as well ---
	// Find kdepsDir and compiledProjectDir from resourcesDir
	compiledProjectDir := filepath.Dir(resourcesDir)
	kdepsDir := ""
	// Try to find kdepsDir by walking up from resourcesDir
	curr := resourcesDir
	for i := 0; i < 5; i++ {
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		if exists, _ := afero.DirExists(fs, filepath.Join(parent, "agents")); exists {
			kdepsDir = parent
			break
		}
		curr = parent
	}
	if kdepsDir == "" {
		logger.Warn("could not determine kdepsDir for data copy", "resourcesDir", resourcesDir)
		return nil
	}

	// Find latest version for agentName
	agentVersion := ""
	agentPath := filepath.Join(kdepsDir, "agents", agentName)
	dirs, err := afero.ReadDir(fs, agentPath)
	if err == nil && len(dirs) > 0 {
		latest := ""
		for _, d := range dirs {
			if d.IsDir() {
				if latest == "" || compareSemver(d.Name(), latest) > 0 {
					latest = d.Name()
				}
			}
		}
		agentVersion = latest
	}
	if agentVersion == "" {
		logger.Warn("could not determine agent version for data copy", "agent", agentName)
		return nil
	}

	srcData := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "data", agentName, agentVersion)
	dstData := filepath.Join(compiledProjectDir, "data", agentName, agentVersion)

	if exists, _ := afero.DirExists(fs, srcData); exists {
		ctx := context.TODO()
		if err := CopyDir(fs, ctx, srcData, dstData, logger); err != nil {
			logger.Warn("failed to copy agent data directory", "src", srcData, "dst", dstData, "error", err)
		} else {
			logger.Debug("copied agent data directory", "src", srcData, "dst", dstData)
		}
	}

	return nil
}

// compareSemver returns 1 if a > b, -1 if a < b, 0 if equal (simple semver, not prerelease aware)
func compareSemver(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var ai, bi int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &ai)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bi)
		}
		if ai > bi {
			return 1
		} else if ai < bi {
			return -1
		}
	}
	return 0
}
