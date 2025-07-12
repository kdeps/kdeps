package resolver

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/data"
	"github.com/kdeps/kdeps/pkg/schema"
	pklData "github.com/kdeps/schema/gen/data"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) PrependDynamicImports(pklFile string) error {
	// Read the file content
	content, err := afero.ReadFile(dr.Fs, pklFile)
	if err != nil {
		return err
	}
	contentStr := string(content)

	// Define ImportConfig struct
	type ImportConfig struct {
		Alias string
		Check bool // Flag to specify if the file existence should be checked
	}

	// Import configurations
	importCheck := map[string]ImportConfig{
		"pkl:json":     {Alias: "", Check: false},
		"pkl:test":     {Alias: "", Check: false},
		"pkl:math":     {Alias: "", Check: false},
		"pkl:platform": {Alias: "", Check: false},
		"pkl:semver":   {Alias: "", Check: false},
		"pkl:shell":    {Alias: "", Check: false},
		"pkl:xml":      {Alias: "", Check: false},
		"pkl:yaml":     {Alias: "", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Document.pkl", schema.SchemaVersion(dr.Context)): {Alias: "document", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Memory.pkl", schema.SchemaVersion(dr.Context)):   {Alias: "memory", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Session.pkl", schema.SchemaVersion(dr.Context)):  {Alias: "session", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Tool.pkl", schema.SchemaVersion(dr.Context)):     {Alias: "tool", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Item.pkl", schema.SchemaVersion(dr.Context)):     {Alias: "item", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Agent.pkl", schema.SchemaVersion(dr.Context)):    {Alias: "agent", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Skip.pkl", schema.SchemaVersion(dr.Context)):     {Alias: "skip", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Utils.pkl", schema.SchemaVersion(dr.Context)):    {Alias: "utils", Check: false},
		dr.PklresHelper.getResourcePath("llm"):                                                            {Alias: "llm", Check: true},
		dr.PklresHelper.getResourcePath("client"):                                                         {Alias: "client", Check: true},
		dr.PklresHelper.getResourcePath("exec"):                                                           {Alias: "exec", Check: true},
		dr.PklresHelper.getResourcePath("python"):                                                         {Alias: "python", Check: true},
		dr.PklresHelper.getResourcePath("data"):                                                           {Alias: "data", Check: true},
		dr.RequestPklFile:                                                                                 {Alias: "request", Check: true},
	}

	// Helper to check file existence (including pklres resources)
	fileExists := func(file string) bool {
		// Check if this is a pklres path
		if strings.HasPrefix(file, "pklres://") {
			// For pklres paths, check if the resource exists
			// Extract the resource type from the path
			if parts := strings.Split(file, "?type="); len(parts) == 2 {
				resourceType := strings.Split(parts[1], "&")[0]
				_, err := dr.PklresHelper.retrievePklContent(resourceType, "")
				return err == nil
			}
			return false
		}
		// For regular file paths, use afero
		exists, _ := afero.Exists(dr.Fs, file)
		return exists
	}

	// Helper to generate import lines
	generateImportLine := func(file, alias string) string {
		if alias == "" {
			return fmt.Sprintf(`import "%s"`, file)
		}
		return fmt.Sprintf(`import "%s" as %s`, file, alias)
	}

	// Helper to check if an alias is already used
	aliasExists := func(alias string) bool {
		if alias == "" {
			return false
		}
		// Check for pattern: import "..." as alias
		aliasPattern := regexp.MustCompile(`import\s+"[^"]+"\s+as\s+` + regexp.QuoteMeta(alias) + `\b`)
		return aliasPattern.MatchString(contentStr)
	}

	// Construct the dynamic import lines
	var importBuilder strings.Builder
	for file, config := range importCheck {
		if config.Check && !fileExists(file) {
			continue
		}

		// Skip if alias is already in use
		if config.Alias != "" && aliasExists(config.Alias) {
			continue
		}

		importLine := generateImportLine(file, config.Alias)
		if !strings.Contains(contentStr, importLine) {
			importBuilder.WriteString(importLine + "\n")
		}
	}

	// If there are no new imports, return early
	importFiles := importBuilder.String()
	if importFiles == "" {
		return nil
	}

	// Add the imports after the "amends" line
	amendsIndex := strings.Index(contentStr, "amends")
	if amendsIndex != -1 {
		amendsLineEnd := strings.Index(contentStr[amendsIndex:], "\n") + amendsIndex + 1
		newContent := contentStr[:amendsLineEnd] + importFiles + contentStr[amendsLineEnd:]

		// Write the updated content back to the file
		err = afero.WriteFile(dr.Fs, pklFile, []byte(newContent), 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareImportFiles() error {
	// Map resource types to their pklres paths
	resourceTypes := map[string]string{
		"llm":    "llm",
		"client": "client",
		"exec":   "exec",
		"python": "python",
		"data":   "data",
	}

	for key, resourceType := range resourceTypes {
		// Initialize empty PKL content for this resource type if it doesn't exist
		// This ensures pklres has the basic structure for imports to work

		// Check if we already have this resource type in pklres
		_, err := dr.PklresHelper.retrievePklContent(resourceType, "")
		if err != nil {
			// If it doesn't exist, create a proper PKL structure with header
			info := dr.PklresHelper.getResourceTypeInfo(resourceType)
			header := dr.PklresHelper.generatePklHeader(resourceType)
			emptyContent := fmt.Sprintf("%s%s {\n}\n", header, info.BlockName)

			// Store the empty structure
			if err := dr.PklresHelper.storePklContent(resourceType, "", emptyContent); err != nil {
				return fmt.Errorf("failed to initialize empty %s structure in pklres: %w", key, err)
			}
		} else {
			// If it exists, we still want to ensure it has the proper structure
			// This handles the case where the record exists but is empty
			info := dr.PklresHelper.getResourceTypeInfo(resourceType)
			header := dr.PklresHelper.generatePklHeader(resourceType)
			emptyContent := fmt.Sprintf("%s%s {\n}\n", header, info.BlockName)

			// Store the empty structure (this will overwrite if it exists)
			if err := dr.PklresHelper.storePklContent(resourceType, "", emptyContent); err != nil {
				return fmt.Errorf("failed to initialize empty %s structure in pklres: %w", key, err)
			}
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareWorkflowDir() error {
	src := dr.ProjectDir
	dest := dr.WorkflowDir
	fs := dr.Fs

	// Check if the destination exists and remove it if it does
	exists, err := afero.Exists(fs, dest)
	if err != nil {
		return fmt.Errorf("failed to check if destination exists: %w", err)
	}
	if exists {
		if err := fs.RemoveAll(dest); err != nil {
			return fmt.Errorf("failed to remove existing destination: %w", err)
		}
	}

	// Walk through the source directory
	err = afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Determine the relative path and destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			// Create directories in the destination
			if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Copy file contents to the destination
			in, err := fs.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()

			out, err := fs.Create(targetPath)
			if err != nil {
				return err
			}
			defer out.Close()

			// Copy file contents
			if _, err := io.Copy(out, in); err != nil {
				return err
			}

			// Set file permissions to match the source file
			if err := fs.Chmod(targetPath, info.Mode()); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (dr *DependencyResolver) AddPlaceholderImports(filePath string) error {
	// Check if Workflow is initialized
	if dr.Workflow == nil {
		return fmt.Errorf("workflow is not initialized")
	}

	// Open the file using afero file system (dr.Fs)
	file, err := dr.Fs.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Use agent.PklResourceReader to resolve the actionID
	agentReader, err := agent.GetGlobalAgentReader(dr.Fs, "", dr.Workflow.GetAgentID(), dr.Workflow.GetVersion(), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize agent reader: %w", err)
	}

	// Extract actionID using a more robust approach
	actionID, err := extractActionIDFromContent(content)
	if err != nil {
		return fmt.Errorf("failed to extract actionID: %w", err)
	}

	// Resolve the actionID canonically using the agent reader
	resolvedActionID, err := resolveActionIDCanonically(actionID, dr.Workflow, agentReader)
	if err != nil {
		return fmt.Errorf("failed to resolve actionID canonically: %w", err)
	}

	dataFileList, err := data.PopulateDataFileRegistry(dr.Fs, dr.DataDir)
	if err != nil {
		return err
	}

	dataFiles := &pklData.DataImpl{
		Files: *dataFileList,
	}
	llmChat := &pklLLM.ResourceChat{}
	execCmd := &pklExec.ResourceExec{}
	pythonCmd := &pklPython.ResourcePython{}
	HTTPClient := &pklHTTP.ResourceHTTPClient{
		Method: "GET",
	}

	if err := dr.AppendDataEntry(resolvedActionID, dataFiles); err != nil {
		return err
	}

	if err := dr.AppendChatEntry(resolvedActionID, llmChat); err != nil {
		return err
	}

	if err := dr.AppendExecEntry(resolvedActionID, execCmd); err != nil {
		return err
	}

	if err := dr.AppendHTTPEntry(resolvedActionID, HTTPClient); err != nil {
		return err
	}

	if err := dr.AppendPythonEntry(resolvedActionID, pythonCmd); err != nil {
		return err
	}

	return nil
}

// extractActionIDFromContent extracts actionID from PKL file content using a more robust approach
func extractActionIDFromContent(content []byte) (string, error) {
	// First try to find actionID using a more precise regex pattern
	// This pattern looks for actionID = "value" with proper PKL syntax
	actionIDPattern := regexp.MustCompile(`(?m)^\s*actionID\s*=\s*"([^"]+)"\s*$`)
	matches := actionIDPattern.FindSubmatch(content)
	if len(matches) >= 2 {
		return string(matches[1]), nil
	}

	// Fallback to a more flexible pattern if the strict one doesn't match
	fallbackPattern := regexp.MustCompile(`(?i)actionID\s*=\s*"([^"]+)"`)
	matches = fallbackPattern.FindSubmatch(content)
	if len(matches) >= 2 {
		return string(matches[1]), nil
	}

	return "", errors.New("actionID not found in file content")
}

// resolveActionIDCanonically resolves an actionID to its canonical form using the agent reader
func resolveActionIDCanonically(actionID string, wf pklWf.Workflow, agentReader *agent.PklResourceReader) (string, error) {
	// If the actionID is already in canonical form (@agent/action:version), return it
	if strings.HasPrefix(actionID, "@") {
		return actionID, nil
	}

	// Add nil check for Workflow
	if wf == nil {
		return "", fmt.Errorf("workflow is nil, cannot resolve action ID")
	}

	// Create URI for agent ID resolution
	query := url.Values{}
	query.Set("op", "resolve")
	query.Set("agent", wf.GetAgentID())
	query.Set("version", wf.GetVersion())
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + actionID,
		RawQuery: query.Encode(),
	}

	resolvedIDBytes, err := agentReader.Read(uri)
	if err != nil {
		// Fallback to default resolution if agent reader fails
		return fmt.Sprintf("@%s/%s:%s", wf.GetAgentID(), actionID, wf.GetVersion()), nil
	}

	return string(resolvedIDBytes), nil
}
