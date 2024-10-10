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

func StartApiServerMode(fs afero.Fs, ctx context.Context, wfCfg *pklWf.Workflow, environ *environment.Environment,
	agentDir string, logger *log.Logger) error {
	// Extracting workflow settings and API server config
	wfSettings := *wfCfg.Settings
	wfApiServer := wfSettings.ApiServer

	if wfApiServer == nil {
		return fmt.Errorf("API server configuration is missing")
	}

	portNum := wfApiServer.PortNum
	hostPort := ":" + strconv.FormatUint(uint64(portNum), 10) // Format port for ListenAndServe

	// Set up routes from the configuration
	routes := wfApiServer.Routes
	for _, route := range routes {
		http.HandleFunc(route.Path, ApiServerHandler(fs, ctx, route, environ, agentDir, logger))
	}

	// Start the server
	log.Printf("Starting API server on port %s", hostPort)
	go func() error {
		if err := http.ListenAndServe(hostPort, nil); err != nil {
			// Return the error instead of log.Fatal to allow better error handling
			return fmt.Errorf("failed to start API server: %w", err)
		}
		return nil
	}()

	return nil
}

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
		// Clean up any old response files or flags
		if err := cleanOldFiles(fs, dr, logger); err != nil {
			http.Error(w, "Failed to clean old files", http.StatusInternalServerError)
			return
		}

		// Validate method and prepare URL, method, headers, params, and data sections
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

		// Create the response flag file
		if err = CreateFlagFile(dr.Fs, dr.ResponseFlag); err != nil {
			http.Error(w, "Failed to create response flag", http.StatusInternalServerError)
			return
		}

		// Handle workflow and process response
		if err := processWorkflow(dr, logger); err != nil {
			http.Error(w, "Workflow processing failed", http.StatusInternalServerError)
			return
		}

		// Read and respond with the contents of the response file
		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			http.Error(w, "Failed to read response file", http.StatusInternalServerError)
			return
		}

		// Format JSON response if necessary
		if responseFile.ContentType == "application/json" {
			content = formatResponseJson(content)
		}

		// Write the response
		w.Header().Set("Content-Type", responseFile.ContentType)
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

// Clean up old response files and flags
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

// Validate the HTTP method and return a formatted method string
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

// Format request headers
func formatHeaders(headers map[string][]string) string {
	var headersLines []string
	for name, values := range headers {
		for _, value := range values {
			headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, strings.TrimSpace(value)))
		}
	}
	return "headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// Format request parameters
func formatParams(params map[string][]string) string {
	var paramsLines []string
	for param, values := range params {
		for _, value := range values {
			paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, strings.TrimSpace(value)))
		}
	}
	return "params {\n" + strings.Join(paramsLines, "\n") + "\n}"
}

// Process workflow and evaluate response file
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

// Format response JSON content
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
