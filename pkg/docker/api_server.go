package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"kdeps/pkg/environment"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/resolver"
	"kdeps/pkg/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	apiserver "github.com/kdeps/schema/gen/api_server"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// StartApiServerMode initializes and starts an API server based on the provided workflow configuration.
// It validates the API server configuration, sets up routes, and starts the server on the configured port.
func StartApiServerMode(fs afero.Fs, ctx context.Context, wfCfg *pklWf.Workflow, environ *environment.Environment,
	agentDir string, logger *log.Logger) error {

	// Extract workflow settings and validate API server configuration
	wfSettings := wfCfg.Settings
	wfApiServer := wfSettings.ApiServer

	if wfApiServer == nil {
		return fmt.Errorf("API server configuration is missing")
	}

	// Format the server host and port
	portNum := strconv.FormatUint(uint64(wfApiServer.PortNum), 10)
	hostPort := ":" + portNum

	// Set up API routes as per the configuration
	if err := setupRoutes(fs, ctx, wfApiServer.Routes, environ, agentDir, logger); err != nil {
		return fmt.Errorf("failed to set up routes: %w", err)
	}

	// Start the API server asynchronously
	logger.Printf("Starting API server on port %s", hostPort)
	go func() {
		if err := http.ListenAndServe(hostPort, nil); err != nil {
			logger.Error("Failed to start API server", "error", err)
		}
	}()

	return nil
}

// setupRoutes configures HTTP routes for the API server based on the provided route configuration.
// Each route is validated before being registered with the HTTP handler.
func setupRoutes(fs afero.Fs, ctx context.Context, routes []*apiserver.APIServerRoutes, environ *environment.Environment,
	agentDir string, logger *log.Logger) error {

	for _, route := range routes {
		if route == nil || route.Path == "" {
			logger.Error("Route configuration is invalid", "route", route)
			continue
		}

		http.HandleFunc(route.Path, ApiServerHandler(fs, ctx, route, environ, agentDir, logger))
		logger.Printf("Route configured: %s", route.Path)
	}

	return nil
}

// ApiServerHandler handles incoming HTTP requests for the configured routes.
// It validates the HTTP method, processes the request data, and triggers workflow actions to generate responses.
func ApiServerHandler(fs afero.Fs, ctx context.Context, route *apiserver.APIServerRoutes, env *environment.Environment,
	apiServerPath string, logger *log.Logger) http.HandlerFunc {

	responseFile := &resolver.ResponseFileInfo{
		RouteResponseType: route.ResponseType,
	}

	allowedMethods := route.Methods

	dr, err := resolver.NewGraphResolver(fs, logger, ctx, env, "/agent", responseFile)
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Clean up old response files before handling the request
		if err := cleanOldFiles(fs, dr, logger); err != nil {
			http.Error(w, "Failed to clean old files", http.StatusInternalServerError)
			return
		}

		// Validate HTTP method and prepare necessary sections for .pkl file creation
		method, err := validateMethod(r, allowedMethods)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Prepare sections for the .pkl request file
		urlSection := fmt.Sprintf(`url = "%s"`, r.URL.Path)
		dataSection := fmt.Sprintf(`data = "%s"`, string(body))
		paramSection := formatParams(r.URL.Query())
		headerSection := formatHeaders(r.Header)

		sections := []string{urlSection, method, headerSection, dataSection, paramSection}

		// Create and process the .pkl request file
		if err := evaluator.CreateAndProcessPklFile(dr.Fs, sections, dr.RequestPklFile, "APIServerRequest.pkl",
			nil, logger, evaluator.EvalPkl); err != nil {
			http.Error(w, "Failed to process request file", http.StatusInternalServerError)
			return
		}

		// Create response flag file to signal the completion of the response process
		if err = CreateFlagFile(dr.Fs, dr.ResponseFlag); err != nil {
			http.Error(w, "Failed to create response flag", http.StatusInternalServerError)
			return
		}

		// Execute the workflow actions and generate the response
		if err := processWorkflow(dr, logger); err != nil {
			http.Error(w, "Workflow processing failed", http.StatusInternalServerError)
			return
		}

		// Read the response file and write it back to the HTTP response
		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			http.Error(w, "Failed to read response file", http.StatusInternalServerError)
			return
		}

		// Format JSON response if required
		if responseFile.ContentType == "application/json" {
			content = formatResponseJson(content)
		}

		// Write the HTTP response with the appropriate content type
		w.Header().Set("Content-Type", responseFile.ContentType)
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

// cleanOldFiles removes any old response files or flags from previous API requests.
// It ensures the environment is clean before processing new requests.
func cleanOldFiles(fs afero.Fs, dr *resolver.DependencyResolver, logger *log.Logger) error {
	if _, err := fs.Stat(dr.ResponseTargetFile); err == nil {
		if err := fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			logger.Error("Unable to delete old response file", "response-target-file", dr.ResponseTargetFile)
			return err
		}
	}
	if _, err := fs.Stat(dr.ResponseFlag); err == nil {
		if err := fs.RemoveAll(dr.ResponseFlag); err != nil {
			logger.Error("Unable to delete old response flag file", "response-flag", dr.ResponseFlag)
			return err
		}
	}
	return nil
}

// validateMethod checks if the incoming HTTP request uses a valid method.
// It returns the formatted method string for .pkl file creation.
func validateMethod(r *http.Request, allowedMethods []string) (string, error) {
	if r.Method == "" {
		r.Method = "GET"
	}

	for _, allowedMethod := range allowedMethods {
		if allowedMethod == r.Method {
			return fmt.Sprintf(`method = "%s"`, allowedMethod), nil
		}
	}

	return "", fmt.Errorf(`HTTP method "%s" not allowed!`, r.Method)
}

// formatHeaders formats the HTTP headers into a string representation for inclusion in the .pkl file.
func formatHeaders(headers map[string][]string) string {
	var headersLines []string
	for name, values := range headers {
		for _, value := range values {
			headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
		}
	}
	return "headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// formatParams formats the query parameters into a string representation for inclusion in the .pkl file.
func formatParams(params map[string][]string) string {
	var paramsLines []string
	for param, values := range params {
		for _, value := range values {
			paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, strings.TrimSpace(value)))
		}
	}
	return "params {\n" + strings.Join(paramsLines, "\n") + "\n}"
}

// processWorkflow handles the execution of the workflow steps after the .pkl file is created.
// It prepares the workflow directory, imports necessary files, and processes the actions defined in the workflow.
func processWorkflow(dr *resolver.DependencyResolver, logger *log.Logger) error {
	if err := dr.PrepareWorkflowDir(); err != nil {
		return err
	}

	if err := dr.PrepareImportFiles(); err != nil {
		return err
	}

	if err := dr.HandleRunAction(); err != nil {
		return err
	}

	stdout, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		logger.Fatal(fmt.Errorf(stdout, err))
		return err
	}

	logger.Info("Awaiting response...")

	// Wait for the response file to be ready
	if err := utils.WaitForFileReady(dr.Fs, dr.ResponseTargetFile, logger); err != nil {
		return err
	}

	return nil
}

// formatResponseJson attempts to format the response content as JSON if required.
// It unmarshals the content into a map, modifies the "data" field if necessary, and re-encodes it into a pretty-printed JSON string.
func formatResponseJson(content []byte) []byte {
	var response map[string]interface{}

	if err := json.Unmarshal(content, &response); err != nil {
		return content
	}

	// Check and format the "data" field if it exists
	if data, ok := response["response"].(map[string]interface{})["data"].([]interface{}); ok {
		for i, item := range data {
			var obj map[string]interface{}
			itemStr, _ := item.(string)
			if err := json.Unmarshal([]byte(itemStr), &obj); err == nil {
				data[i] = obj
			}
		}
	}

	// Marshal the modified response back into JSON
	modifiedContent, _ := json.MarshalIndent(response, "", "  ")

	return modifiedContent
}
