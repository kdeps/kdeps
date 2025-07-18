package resolver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/schema"
	pklData "github.com/kdeps/schema/gen/data"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklResource "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
)

// ResourceType defines the type of resource to load.
type ResourceType string

const (
	ExecResource     ResourceType = "exec"
	PythonResource   ResourceType = "python"
	LLMResource      ResourceType = "llm"
	HTTPResource     ResourceType = "http"
	DataResource     ResourceType = "data"
	ResponseResource ResourceType = "response"
	Resource         ResourceType = "resource"
)

// LoadResourceEntries loads .pkl resource files from the resources directory.
func (dr *DependencyResolver) LoadResourceEntries() error {
	projectDir := filepath.Join(dr.ProjectDir, "resources")
	var pklFiles []string

	// Walk through the projectDir to find .pkl files
	walkFn := dr.WalkFn
	if walkFn == nil {
		walkFn = afero.Walk
	}

	err := walkFn(dr.Fs, projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			dr.Logger.Errorf("error accessing path %s: %v", path, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			// Add the file to the list of .pkl files
			pklFiles = append(pklFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk through the project directory: %w", err)
	}

	// Process all .pkl files found
	for _, file := range pklFiles {
		if err := dr.ProcessPklFile(file); err != nil {
			dr.Logger.Errorf("error processing .pkl file %s: %v", file, err)
			return err
		}
	}

	return nil
}

// detectResourceType determines the resource type based on file content
func (dr *DependencyResolver) detectResourceType(file string) ResourceType {
	content, err := afero.ReadFile(dr.Fs, file)
	if err != nil {
		dr.Logger.Warn("failed to read file for resource type detection", "file", file, "error", err)
		return Resource // fallback to generic resource
	}

	lines := strings.SplitN(string(content), "\n", 5)
	if len(lines) > 0 {
		first := strings.TrimSpace(lines[0])
		if strings.HasPrefix(first, "extends") || strings.HasPrefix(first, "amends") {
			if strings.Contains(first, "/HTTP.pkl\"") {
				return HTTPResource
			}
			if strings.Contains(first, "/LLM.pkl\"") {
				return LLMResource
			}
			if strings.Contains(first, "/Python.pkl\"") {
				return PythonResource
			}
			if strings.Contains(first, "/Exec.pkl\"") {
				return ExecResource
			}
			if strings.Contains(first, "/Data.pkl\"") {
				return DataResource
			}
			if strings.Contains(first, "/APIServerResponse.pkl\"") {
				return ResponseResource
			}
		}
	}
	return Resource // fallback to generic resource
}

// ProcessPklFile processes an individual .pkl file and updates dependencies.
func (dr *DependencyResolver) ProcessPklFile(file string) error {
	// Detect the resource type based on file content
	resourceType := dr.detectResourceType(file)
	dr.Logger.Debug("detected resource type", "file", file, "type", resourceType)

	// Load the resource file with the detected type
	res, err := dr.LoadResourceFn(dr.Context, file, resourceType)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Extract ActionID and Requires based on the resource type
	var actionID string
	var requires *[]string

	// All resource types should have ActionID and Requires fields
	// Try to cast to the base Resource type first
	if genericRes, ok := res.(*pklResource.Resource); ok {
		actionID = genericRes.ActionID
		requires = genericRes.Requires
	} else {
		// If that fails, try to extract using reflection
		dr.Logger.Warn("failed to cast to *pklResource.Resource, trying reflection", "resourceType", resourceType, "actualType", fmt.Sprintf("%T", res))
		return errors.New("failed to extract ActionID and Requires from resource")
	}

	// Process data resources immediately when loaded
	if resourceType == DataResource {
		if dataRes, ok := res.(*pklData.DataImpl); ok {
			if err := dr.HandleData(actionID, dataRes); err != nil {
				dr.Logger.Error("failed to handle data resource", "actionID", actionID, "error", err)
				return fmt.Errorf("failed to handle data resource: %w", err)
			}
		}
	}

	// Append the resource to the list of resources
	dr.Resources = append(dr.Resources, ResourceNodeEntry{
		ActionID: actionID,
		File:     file,
	})

	// Update resource dependencies
	if requires != nil {
		dr.ResourceDependencies[actionID] = *requires
	} else {
		dr.ResourceDependencies[actionID] = nil
	}

	return nil
}

// LoadResource reads a resource file and returns the parsed resource object or an error.
func (dr *DependencyResolver) LoadResource(ctx context.Context, resourceFile string, resourceType ResourceType) (interface{}, error) {
	// Log additional info before reading the resource
	dr.Logger.Debug("reading resource file", "resource-file", resourceFile, "resource-type", resourceType)

	// Check if Workflow is initialized
	if dr.Workflow != nil {
		// Set environment variables for current agent context
		os.Setenv("KDEPS_CURRENT_AGENT", dr.Workflow.GetAgentID())
		os.Setenv("KDEPS_CURRENT_VERSION", dr.Workflow.GetVersion())
	}

	// Use the existing evaluator from the dependency resolver
	if dr.Evaluator == nil {
		return nil, fmt.Errorf("evaluator is required but was nil")
	}

	// Load the resource based on the resource type
	return dr.loadResourceByType(ctx, dr.Evaluator, resourceFile, resourceType, "")
}

// LoadResourceWithRequestContext reads a resource file in a context that includes request data
// This allows resources to access request.params(), request.headers(), etc.
func (dr *DependencyResolver) LoadResourceWithRequestContext(ctx context.Context, resourceFile string, resourceType ResourceType) (interface{}, error) {
	// Log additional info before reading the resource
	dr.Logger.Debug("reading resource file with request context", "resource-file", resourceFile, "resource-type", resourceType)

	// Check if Workflow is initialized
	if dr.Workflow != nil {
		// Set environment variables for current agent context
		os.Setenv("KDEPS_CURRENT_AGENT", dr.Workflow.GetAgentID())
		os.Setenv("KDEPS_CURRENT_VERSION", dr.Workflow.GetVersion())
	}

	// Populate request data in pklres if available
	if dr.RequestPklFile != "" {
		if err := dr.PopulateRequestDataInPklres(); err != nil {
			dr.Logger.Warn("failed to populate request data in pklres", "error", err)
		}
	}

	// Use the existing evaluator from the dependency resolver
	if dr.Evaluator == nil {
		return nil, fmt.Errorf("evaluator is required but was nil")
	}

	// Use the standard evaluator with pklres reader, which should handle template expressions
	return dr.loadResourceByType(ctx, dr.Evaluator, resourceFile, resourceType, " with request context")
}

// PopulateRequestDataInPklres stores request data in pklres using the key-value store approach
func (dr *DependencyResolver) PopulateRequestDataInPklres() error {
	dr.Logger.Debug("populateRequestDataInPklres: called", "requestID", dr.RequestID)
	if dr.RequestPklFile == "" {
		return errors.New("no request PKL file specified")
	}

	// Get the canonicalized request ID
	canonicalRequestID := dr.RequestID
	if dr.AgentReader != nil {
		// Create a URI for agent resolution
		query := url.Values{}
		query.Set("op", "resolve")
		query.Set("agent", dr.Workflow.GetAgentID())
		query.Set("version", dr.Workflow.GetVersion())
		uri := url.URL{
			Scheme:   "agent",
			Path:     "/" + dr.RequestID,
			RawQuery: query.Encode(),
		}

		if resolvedIDBytes, err := dr.AgentReader.Read(uri); err == nil {
			canonicalRequestID = string(resolvedIDBytes)
			dr.Logger.Debug("canonicalized request ID for storage", "original", dr.RequestID, "canonical", canonicalRequestID)
		}
	}

	// Check if the request file exists
	if _, err := dr.Fs.Stat(dr.RequestPklFile); err != nil {
		return fmt.Errorf("request PKL file does not exist: %w", err)
	}

	// Read the request PKL file content
	requestBytes, err := afero.ReadFile(dr.Fs, dr.RequestPklFile)
	if err != nil {
		return fmt.Errorf("failed to read request PKL file: %w", err)
	}

	// Store the request data in pklres as individual key-value pairs
	if dr.PklresHelper != nil {
		// Store the request ID
		if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "requestID", dr.RequestID); err != nil {
			dr.Logger.Warn("failed to store request ID", "error", err)
		}

		// Store the request file content
		if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "file", string(requestBytes)); err != nil {
			dr.Logger.Warn("failed to store request file", "error", err)
		}

		// Store additional request metadata
		if dr.Request != nil {
			// Store path
			if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "path", dr.Request.Request.URL.Path); err != nil {
				dr.Logger.Warn("failed to store request path", "error", err)
			}

			// Store method
			if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "method", dr.Request.Request.Method); err != nil {
				dr.Logger.Warn("failed to store request method", "error", err)
			}

			// Store client IP
			if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "ip", dr.Request.ClientIP()); err != nil {
				dr.Logger.Warn("failed to store request IP", "error", err)
			}

			// Store headers as JSON
			headers := make(map[string]string)
			for key, values := range dr.Request.Request.Header {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}
			if headersJSON, err := json.Marshal(headers); err == nil {
				if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "headers", string(headersJSON)); err != nil {
					dr.Logger.Warn("failed to store request headers", "error", err)
				}
			}

			// Store query parameters as JSON
			params := make(map[string]string)
			for key, values := range dr.Request.Request.URL.Query() {
				if len(values) > 0 {
					params[key] = values[0]
				}
			}
			if paramsJSON, err := json.Marshal(params); err == nil {
				if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "params", string(paramsJSON)); err != nil {
					dr.Logger.Warn("failed to store request params", "error", err)
				}
			}

			// Store request body
			if dr.Request.Request.Body != nil {
				if bodyBytes, err := io.ReadAll(dr.Request.Request.Body); err == nil {
					// Restore the body for potential future reads
					dr.Request.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					// Store the body as base64 encoded
					bodyBase64 := base64.StdEncoding.EncodeToString(bodyBytes)
					if err := dr.PklresHelper.StoreResourceRecord("request", canonicalRequestID, "data", bodyBase64); err != nil {
						dr.Logger.Warn("failed to store request body", "error", err)
					}
				}
			}
		}

		dr.Logger.Debug("storeRequestResource: successfully stored request resource", "requestID", canonicalRequestID)
	}

	return nil
}

// storeRequestResource stores the request resource in pklres with type=request
func (dr *DependencyResolver) storeRequestResource(requestID string) error {
	if dr.PklresHelper == nil || dr.PklresHelper.resolver == nil || dr.PklresHelper.resolver.PklresReader == nil {
		return errors.New("pklres reader not available")
	}

	dr.Logger.Debug("storeRequestResource: storing request resource", "requestID", requestID)

	// Store the request resource with type=request using the correct pklres URI format
	// The system looks for resources with type=request in pklres://?op=set&collection={requestID}&key=resource&value={resourceJSON}
	resourceData := map[string]interface{}{
		"type":     "request",
		"id":       requestID,
		"file":     "virtual://request.pkl",
		"finished": true,
	}

	resourceJSON, err := json.Marshal(resourceData)
	if err != nil {
		return fmt.Errorf("failed to marshal request resource data: %w", err)
	}

	// Store the request resource using the correct pklres URI format
	// Format: pklres://?op=set&collection={requestID}&key=resource&value={resourceJSON}
	setURI := fmt.Sprintf("pklres://?op=set&collection=%s&key=resource&value=%s",
		url.QueryEscape(requestID),
		url.QueryEscape(string(resourceJSON)))

	parsedURI, err := url.Parse(setURI)
	if err != nil {
		return fmt.Errorf("failed to parse set URI: %w", err)
	}

	_, err = dr.PklresHelper.resolver.PklresReader.Read(*parsedURI)
	if err != nil {
		dr.Logger.Error("storeRequestResource: failed to store request resource", "requestID", requestID, "error", err)
		return fmt.Errorf("failed to store request resource: %w", err)
	}

	dr.Logger.Debug("storeRequestResource: successfully stored request resource", "requestID", requestID)
	return nil
}

// storeRequestData is a helper function to store a single key-value pair in pklres
func (dr *DependencyResolver) storeRequestData(collection, key, value string) error {
	if dr.PklresHelper == nil || dr.PklresHelper.resolver == nil || dr.PklresHelper.resolver.PklresReader == nil {
		return errors.New("pklres reader not available")
	}

	dr.Logger.Debug("storeRequestData: storing", "collection", collection, "key", key, "value", value, "value_length", len(value))

	setURI := fmt.Sprintf("pklres://?op=set&collection=%s&key=%s&value=%s",
		url.QueryEscape(collection),
		url.QueryEscape(key),
		url.QueryEscape(value))

	dr.Logger.Debug("storeRequestData: constructed URI", "uri", setURI)

	parsedURI, err := url.Parse(setURI)
	if err != nil {
		return fmt.Errorf("failed to parse set URI: %w", err)
	}

	_, err = dr.PklresHelper.resolver.PklresReader.Read(*parsedURI)
	if err != nil {
		dr.Logger.Error("storeRequestData: failed to store", "collection", collection, "key", key, "value", value, "error", err)
		return fmt.Errorf("failed to store request data: %w", err)
	}

	dr.Logger.Debug("storeRequestData: successfully stored", "collection", collection, "key", key, "value", value)
	return nil
}

// createResourceWithRequestContext creates a temporary PKL file that includes both the request data and the resource
func (dr *DependencyResolver) createResourceWithRequestContext(resourceFile string) (string, error) {
	// Read the request PKL file content
	requestBytes, err := afero.ReadFile(dr.Fs, dr.RequestPklFile)
	if err != nil {
		return "", fmt.Errorf("failed to read request PKL file: %w", err)
	}

	// Read the resource file content
	resourceBytes, err := afero.ReadFile(dr.Fs, resourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read resource file: %w", err)
	}

	// Create a temporary file that combines both
	tmpFile, err := afero.TempFile(dr.Fs, "", "resource-with-request-*.pkl")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	// Write the request data first (this provides the request context)
	if _, err := tmpFile.Write(requestBytes); err != nil {
		return "", fmt.Errorf("failed to write request data to temporary file: %w", err)
	}

	// Add a newline separator
	if _, err := tmpFile.WriteString("\n\n"); err != nil {
		return "", fmt.Errorf("failed to write separator to temporary file: %w", err)
	}

	// Convert the resource to amend the request context instead of extending its own template
	resourceContent := string(resourceBytes)

	// Replace the extends statement with amends to modify the request context
	// This allows the resource to access the request data while maintaining its own structure
	if strings.Contains(resourceContent, "extends") {
		// Find and replace extends with amends to the request context
		lines := strings.Split(resourceContent, "\n")
		for i, line := range lines {
			if strings.Contains(line, "extends") {
				// Skip the extends line and let the resource amend the request context
				lines[i] = "// Original extends line: " + line
				break
			}
		}
		resourceContent = strings.Join(lines, "\n")
	}

	// Write the modified resource content
	if _, err := tmpFile.WriteString(resourceContent); err != nil {
		return "", fmt.Errorf("failed to write resource content to temporary file: %w", err)
	}

	return tmpFile.Name(), nil
}

// loadResourceByType is a helper function that eliminates duplicate code between LoadResource and LoadResourceWithRequestContext
func (dr *DependencyResolver) loadResourceByType(ctx context.Context, pklEvaluator pkl.Evaluator, resourceFile string, resourceType ResourceType, contextSuffix string) (interface{}, error) {
	var source *pkl.ModuleSource

	// Set loading phase flag to prevent circular dependencies
	if dr.PklresHelper != nil {
		dr.PklresHelper.SetLoadingPhase(true)
		defer dr.PklresHelper.SetLoadingPhase(false)
	}

	// In API server mode with request context, create a temporary PKL file that includes the request data
	if dr.APIServerMode && dr.RequestPklFile != "" && contextSuffix == " with request context" {
		// Create a temporary file that amends the resource with request context
		tmpFile, err := dr.createResourceWithRequestContext(resourceFile)
		if err != nil {
			dr.Logger.Error("failed to create resource with request context"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("failed to create resource with request context: %w", err)
		}
		defer func() {
			if err := dr.Fs.Remove(tmpFile); err != nil {
				dr.Logger.Warn("failed to cleanup temporary resource file", "file", tmpFile, "error", err)
			}
		}()
		source = pkl.FileSource(tmpFile)
		dr.Logger.Debug("created temporary resource file with request context", "original", resourceFile, "temporary", tmpFile)
	} else {
		source = pkl.FileSource(resourceFile)
	}

	switch resourceType {
	case Resource:
		res, err := pklResource.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	case ExecResource:
		res, err := pklExec.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading exec resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading exec resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded exec resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	case PythonResource:
		res, err := pklPython.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading python resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading python resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded python resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	case LLMResource:
		res, err := pklLLM.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading llm resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading llm resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded llm resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	case HTTPResource:
		res, err := pklHTTP.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading http resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading http resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded http resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	case DataResource:
		res, err := pklData.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading data resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading data resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded data resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	case ResponseResource:
		// For response resources, we need to import the APIServerResponse schema
		// and load it as a generic resource since there's no specific response loader
		res, err := pklResource.Load(ctx, pklEvaluator, source)
		if err != nil {
			dr.Logger.Error("error reading response resource file"+contextSuffix, "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading response resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded response resource"+contextSuffix, "resource-file", resourceFile)
		return res, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// getSchemaVersion returns the schema version for the current context
func (dr *DependencyResolver) getSchemaVersion(ctx context.Context) string {
	// Use the schema package to get the proper version
	if ctx != nil {
		return schema.Version(ctx)
	}
	return "0.3.0" // Fallback version
}

// stripAmendsLine removes the amends/extends line from PKL content
func (dr *DependencyResolver) stripAmendsLine(content string) string {
	lines := strings.Split(content, "\n")
	var filteredLines []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && (strings.HasPrefix(trimmed, "amends") || strings.HasPrefix(trimmed, "extends")) {
			continue // skip the first amends/extends line
		}
		filteredLines = append(filteredLines, line)
	}

	return strings.Join(filteredLines, "\n")
}
