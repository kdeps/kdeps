package docker

import (
	"bytes"
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

// Response structure
type DecodedResponse struct {
	Success  bool `json:"success"`
	Response struct {
		Data []string `json:"data"`
	} `json:"response"`
	Errors struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

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

	allowedMethods := route.Methods

	dr, err := resolver.NewGraphResolver(fs, logger, ctx, env, "/agent")
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
			logger, evaluator.EvalPkl); err != nil {
			http.Error(w, "Failed to process request file", http.StatusInternalServerError)
			return
		}

		// Execute the workflow actions and generate the response
		if err := processWorkflow(dr, logger); err != nil {
			http.Error(w, "Workflow processing failed", http.StatusInternalServerError)
			return
		}

		// Read the raw response file (this is the undecoded data)
		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			http.Error(w, "Failed to read response file", http.StatusInternalServerError)
			return
		}

		// Decode the Base64-encoded data in the content (if applicable)
		decodedContent, err := decodeResponseContent(content, logger)
		if err != nil {
			http.Error(w, "Failed to decode response content", http.StatusInternalServerError)
			return
		}

		decodedContent = formatResponseJson(decodedContent)

		// Write the HTTP response with the appropriate content type
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(decodedContent)
	}
}

// Helper function to detect if a string is valid JSON
func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func decodeResponseContent(content []byte, logger *log.Logger) ([]byte, error) {
	var decodedResp DecodedResponse

	// Unmarshal JSON content into DecodedResponse struct
	err := json.Unmarshal(content, &decodedResp)
	if err != nil {
		logger.Error("Failed to unmarshal response content", "error", err)
		return nil, err
	}

	// Decode Base64 strings in the Data field
	for i, encodedData := range decodedResp.Response.Data {
		decodedData, err := utils.DecodeBase64String(encodedData)
		if err != nil {
			logger.Error("Failed to decode Base64 string", "data", encodedData)
			decodedResp.Response.Data[i] = encodedData // Use original if decoding fails
		} else {
			// If the decoded string is still wrapped in extra quotes, handle unquoting
			if strings.HasPrefix(decodedData, "\"") && strings.HasSuffix(decodedData, "\"") {
				unquotedData, err := strconv.Unquote(decodedData)
				if err == nil {
					decodedData = unquotedData
				}
			}

			// Clean up any remaining escape sequences (like \n or \") if present
			decodedData = strings.ReplaceAll(decodedData, "\\\"", "\"")
			decodedData = strings.ReplaceAll(decodedData, "\\n", "\n")

			// If the decoded data is JSON, pretty print it
			if isJSON(decodedData) {
				var prettyJSON bytes.Buffer
				err := json.Indent(&prettyJSON, []byte(decodedData), "", "  ")
				if err == nil {
					decodedData = prettyJSON.String()
				}
			}

			// Assign the cleaned-up data back to the response
			decodedResp.Response.Data[i] = decodedData
		}
	}

	// Marshal the decoded response back to JSON
	decodedContent, err := json.Marshal(decodedResp)
	if err != nil {
		logger.Error("Failed to marshal decoded response content", "error", err)
		return nil, err
	}

	return decodedContent, nil
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

	logger.Debug("Awaiting response...")

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

	// Attempt to unmarshal the content into the response map
	if err := json.Unmarshal(content, &response); err != nil {
		// Return the original content if unmarshalling fails
		return content
	}

	// Safely check if "response" is a map and if "data" exists and is an array
	if responseField, ok := response["response"].(map[string]interface{}); ok {
		if data, ok := responseField["data"].([]interface{}); ok {
			for i, item := range data {
				// Attempt to unmarshal each item in "data" if it's a string
				if itemStr, isString := item.(string); isString {
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(itemStr), &obj); err == nil {
						data[i] = obj
					}
				}
			}
		}
	}

	// Marshal the modified response back into JSON
	modifiedContent, _ := json.MarshalIndent(response, "", "  ")

	return modifiedContent
}
