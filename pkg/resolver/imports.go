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
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// PrependDynamicImports has been removed as it's no longer needed.
// We now use real-time pklres access instead of prepending import statements.
func (dr *DependencyResolver) PrependDynamicImports(pklFile string) error {
	dr.Logger.Info("Skipping PrependDynamicImports - using real-time pklres access", "pklFile", pklFile)
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
		_, err := dr.PklresHelper.RetrievePklContent(resourceType, "")
		if err != nil {
			// If it doesn't exist, create a proper PKL structure with header
			info := dr.PklresHelper.getResourceTypeInfo(resourceType)
			header := dr.PklresHelper.generatePklHeader(resourceType)
			emptyContent := fmt.Sprintf("%s%s {\n}\n", header, info.BlockName)

			// Store the empty structure
			if err := dr.PklresHelper.StorePklContent(resourceType, "__empty__", emptyContent); err != nil {
				return fmt.Errorf("failed to initialize empty %s structure in pklres: %w", key, err)
			}
		} else {
			// If it exists, we still want to ensure it has the proper structure
			// This handles the case where the record exists but is empty
			info := dr.PklresHelper.getResourceTypeInfo(resourceType)
			header := dr.PklresHelper.generatePklHeader(resourceType)
			emptyContent := fmt.Sprintf("%s%s {\n}\n", header, info.BlockName)

			// Store the empty structure (this will overwrite if it exists)
			if err := dr.PklresHelper.StorePklContent(resourceType, "__empty__", emptyContent); err != nil {
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
		return errors.New("workflow is not initialized")
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

	// All AppendXXXEntry functions have been removed as they're no longer needed.
	// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.
	// Resource output files are written directly during processing and accessed via pklres.
	dr.Logger.Info("Skipping all AppendXXXEntry calls - using real-time pklres access", "resolvedActionID", resolvedActionID)

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
		return "", errors.New("workflow is nil, cannot resolve action ID")
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

	resovledIDBytes, err := agentReader.Read(uri)
	if err != nil {
		// Fallback to default resolution if agent reader fails
		return "", fmt.Errorf("failed to resolve actionID: %w", err)
	}

	return string(resovledIDBytes), nil
}
