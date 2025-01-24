package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/gabriel-vasile/mimetype"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserver "github.com/kdeps/schema/gen/api_server"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// ErrorResponse defines the structure of each error.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// DecodedResponse defines the overall response structure.
type DecodedResponse struct {
	Success  bool `json:"success"`
	Response struct {
		Data []string `json:"data"`
	} `json:"response"`
	Errors []ErrorResponse `json:"errors"`
}

// StartAPIServerMode initializes and starts an API server based on the provided workflow configuration.
// It validates the API server configuration, sets up routes, and starts the server on the configured port.
func StartAPIServerMode(fs afero.Fs, ctx context.Context, wfCfg pklWf.Workflow, environ *environment.Environment,
	agentDir string, apiServerPath string, logger *logging.Logger,
) error {
	// Extract workflow settings and validate API server configuration
	wfSettings := wfCfg.GetSettings()
	wfAPIServer := wfSettings.APIServer

	if wfAPIServer == nil {
		return errors.New("the API server configuration is missing")
	}

	// Format the server host and port
	portNum := strconv.FormatUint(uint64(wfAPIServer.PortNum), 10)
	hostPort := ":" + portNum

	// Set up API routes as per the configuration
	setupRoutes(fs, ctx, wfAPIServer.Routes, environ, agentDir, apiServerPath, logger)

	// Start the API server asynchronously
	logger.Printf("Starting API server on port %s", hostPort)
	go func() {
		if err := http.ListenAndServe(hostPort, nil); err != nil {
			logger.Error("failed to start API server", "error", err)
		}
	}()

	return nil
}

// setupRoutes configures HTTP routes for the API server based on the provided route configuration.
// Each route is validated before being registered with the HTTP handler.
func setupRoutes(fs afero.Fs, ctx context.Context, routes []*apiserver.APIServerRoutes, environ *environment.Environment,
	agentDir string, apiServerPath string, logger *logging.Logger,
) {
	for _, route := range routes {
		if route == nil || route.Path == "" {
			logger.Error("route configuration is invalid", "route", route)
			continue
		}

		http.HandleFunc(route.Path, APIServerHandler(fs, ctx, route, environ, agentDir, apiServerPath, logger))
		logger.Printf("Route configured: %s", route.Path)
	}
}

// APIServerHandler handles incoming HTTP requests for the configured routes.
// It validates the HTTP method, processes the request data, and triggers workflow actions to generate responses.
func APIServerHandler(fs afero.Fs, ctx context.Context, route *apiserver.APIServerRoutes, env *environment.Environment,
	agentDir string, apiServerPath string, logger *logging.Logger,
) http.HandlerFunc {
	allowedMethods := route.Methods

	dr, err := resolver.NewGraphResolver(fs, ctx, env, agentDir, logger)
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

		// Handle OPTIONS request
		if r.Method == http.MethodOptions {
			// List all the methods supported by this endpoint
			w.Header().Set("Allow", "OPTIONS, GET, HEAD, POST, PUT, PATCH, DELETE")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Handle HEAD request
		if r.Method == http.MethodHead {
			// Simulate a GET request, but only respond with headers
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		}

		var filename, filetype, bodyData string

		fileMap := make(map[string]struct {
			Filename string
			Filetype string
		})

		// Handle logic based on HTTP methods
		switch r.Method {
		case http.MethodGet:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
			}
			defer r.Body.Close()

			bodyData = string(body)

		case http.MethodPost, http.MethodPut, http.MethodPatch:
			contentType := r.Header.Get("Content-Type")
			if strings.Contains(contentType, "multipart/form-data") {
				// Try to handle multiple files
				err := r.ParseMultipartForm(10 << 20) // limit the size to 10MB, adjust as needed
				if err != nil {
					http.Error(w, "Unable to parse multipart form", http.StatusInternalServerError)
					return
				}

				files, multiFileExists := r.MultipartForm.File["file[]"]
				singleFile, singleFileExists, err := r.FormFile("file")
				if err != nil {
					http.Error(w, "Unable to parse file", http.StatusInternalServerError)
					return
				}

				if multiFileExists {
					// Handle multiple file uploads
					for _, fileHeader := range files {
						file, err := fileHeader.Open()
						if err != nil {
							http.Error(w, fmt.Sprintf("Unable to open file: %v", err), http.StatusInternalServerError)
							return
						}
						defer file.Close()

						// Read the file contents (Base64 encode if necessary)
						fileBytes, err := io.ReadAll(file)
						if err != nil {
							http.Error(w, "Failed to read file content", http.StatusInternalServerError)
							return
						}

						filetype = mimetype.Detect(fileBytes).String()
						filesPath := filepath.Join(agentDir, "/actions/files/")
						filename = filepath.Join(filesPath, fileHeader.Filename)

						fileMap[filename] = struct {
							Filename string
							Filetype string
						}{
							Filename: filename,
							Filetype: filetype,
						}

						// Create the directory if it does not exist
						if err := fs.MkdirAll(filesPath, 0o777); err != nil {
							http.Error(w, "Unable to create files directory", http.StatusInternalServerError)
							return
						}

						// Write the file to the filesystem
						err = afero.WriteFile(fs, filename, fileBytes, 0o644)
						if err != nil {
							http.Error(w, "Failed to save file", http.StatusInternalServerError)
							return
						}
					}
				} else if singleFileExists != nil {
					// Handle single file upload
					defer singleFile.Close()

					fileBytes, err := io.ReadAll(singleFile)
					if err != nil {
						http.Error(w, "Failed to read file content", http.StatusInternalServerError)
						return
					}

					filetype = mimetype.Detect(fileBytes).String()
					filesPath := filepath.Join(agentDir, "/actions/files/")
					filename = filepath.Join(filesPath, singleFileExists.Filename)

					fileMap[filename] = struct {
						Filename string
						Filetype string
					}{
						Filename: filename,
						Filetype: filetype,
					}

					// Create the directory if it does not exist
					if err := fs.MkdirAll(filesPath, 0o777); err != nil {
						http.Error(w, "Unable to create files directory", http.StatusInternalServerError)
						return
					}

					// Write the file to the filesystem
					err = afero.WriteFile(fs, filename, fileBytes, 0o644)
					if err != nil {
						http.Error(w, "Failed to save file", http.StatusInternalServerError)
						return
					}
				} else {
					http.Error(w, "No file uploaded", http.StatusBadRequest)
					return
				}
			} else {
				// Handle regular form or raw data
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Failed to read request body", http.StatusBadRequest)
					return
				}
				defer r.Body.Close()

				bodyData = string(body)
			}

		case http.MethodDelete:
			bodyData = "Delete request received"
		default:
			http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
			return
		}

		// Prepare sections for the .pkl request file
		urlSection := fmt.Sprintf(`path = "%s"`, r.URL.Path)
		dataSection := fmt.Sprintf(`data = "%s"`, utils.EncodeBase64String(bodyData))

		var sb strings.Builder
		sb.WriteString("files {\n")

		if len(fileMap) == 0 {
			// If the map is empty, just add the closing brace
			sb.WriteString("}\n")
		} else {
			for _, fileInfo := range fileMap {
				fileBlock := fmt.Sprintf(`
filepath = "%s"
filetype = "%s"
`, fileInfo.Filename, fileInfo.Filetype)
				sb.WriteString(fmt.Sprintf("    [\"%s\"] {\n%s\n}\n", filepath.Base(fileInfo.Filename), fileBlock))
			}
			sb.WriteString("}\n")
		}

		fileSection := sb.String()
		paramSection := formatParams(r.URL.Query())
		headerSection := formatHeaders(r.Header)

		// Include filename and filetype in sections
		sections := []string{urlSection, method, headerSection, dataSection, paramSection, fileSection}

		// Create and process the .pkl request file
		if err := evaluator.CreateAndProcessPklFile(dr.Fs, ctx, sections, dr.RequestPklFile, "APIServerRequest.pkl",
			logger, evaluator.EvalPkl, true); err != nil {
			http.Error(w, "Failed to process request file", http.StatusInternalServerError)
			return
		}

		// Execute the workflow actions and generate the response
		fatal, err := processWorkflow(ctx, dr, logger)
		if err != nil {
			http.Error(w, "Workflow processing failed", http.StatusInternalServerError)
			return
		}

		// In certain error cases, Ollama needs to be restarted
		if fatal {
			logger.Fatal("a fatal server error occurred. Restarting the service.")

			// Send SIGTERM to gracefully shut down the server
			utils.SendSigterm(logger)
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

		decodedContent = formatResponseJSON(decodedContent)

		// Write the HTTP response with the appropriate content type
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(decodedContent); err != nil {
			http.Error(w, "unexpected error writing content", http.StatusInternalServerError)
			return
		}
	}
}

// Helper function to detect if a string is valid JSON.
func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func fixJSON(input string) string {
	// Fix unescaped quotes in strings
	reUnescapedQuotes := regexp.MustCompile(`([^\\])"([^,}\]\s])`)
	input = reUnescapedQuotes.ReplaceAllString(input, `$1\"$2`)

	// Remove trailing commas
	reTrailingCommas := regexp.MustCompile(`,(\s*[}\]])`)
	input = reTrailingCommas.ReplaceAllString(input, `$1`)

	// Remove extra double quotes
	reExtraQuotes := regexp.MustCompile(`\"{2,}`)
	input = reExtraQuotes.ReplaceAllString(input, `\"`)

	// Wrap unquoted keys
	reUnquotedKeys := regexp.MustCompile(`\{\s*([a-zA-Z0-9_]+)\s*:`)
	input = reUnquotedKeys.ReplaceAllString(input, `{"$1":`)

	reUnquotedKeys2 := regexp.MustCompile(`\{|\s*,\s*([a-zA-Z0-9_]+)\s*:`)
	input = reUnquotedKeys2.ReplaceAllStringFunc(input, func(match string) string {
		if match[0] == ',' {
			return `, "` + match[2:] + `"`
		}
		return match
	})

	// Remove trailing backslashes
	reTrailingBackslashes := regexp.MustCompile(`\\+$`)
	input = reTrailingBackslashes.ReplaceAllString(input, "")

	// Remove backslashes escaping quotes
	reEscapedQuotes2 := regexp.MustCompile(`\\+"`)
	input = reEscapedQuotes2.ReplaceAllString(input, `"`)

	// Remove trailing backslashes
	reTrailingBackslashes2 := regexp.MustCompile(`\\+$`)
	input = reTrailingBackslashes2.ReplaceAllString(input, "")

	return input
}

func decodeResponseContent(content []byte, logger *logging.Logger) ([]byte, error) {
	var decodedResp DecodedResponse

	// Unmarshal JSON content into DecodedResponse struct
	err := json.Unmarshal(content, &decodedResp)
	if err != nil {
		logger.Error("failed to unmarshal response content", "error", err)
		return nil, err
	}

	// Decode Base64 strings in the Data field
	for i, encodedData := range decodedResp.Response.Data {
		decodedData, err := utils.DecodeBase64String(encodedData)
		if err != nil {
			logger.Error("failed to decode Base64 string", "data", encodedData)
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
			// https://stackoverflow.com/questions/53776683/regex-find-newline-between-double-quotes-and-replace-with-space/53777149#53777149
			matchNewlines := regexp.MustCompile(`[\r\n]`)
			escapeNewlines := func(s string) string {
				return matchNewlines.ReplaceAllString(s, "\\n")
			}
			re := regexp.MustCompile(`"[^"\\]*(?:\\[\s\S][^"\\]*)*"`)
			invalidJSON := re.ReplaceAllStringFunc(decodedData, escapeNewlines)

			// Pass in the invalidJSON to the fixJSON
			fixedJSON := fixJSON(invalidJSON)

			// If the decoded data is JSON, pretty print it
			if isJSON(fixedJSON) {
				var prettyJSON bytes.Buffer
				err := json.Indent(&prettyJSON, []byte(fixedJSON), "", "  ")
				if err == nil {
					fixedJSON = prettyJSON.String()
				}
			}

			// Assign the cleaned-up data back to the response
			decodedResp.Response.Data[i] = fixedJSON
		}
	}

	// Marshal the decoded response back to JSON
	decodedContent, err := json.Marshal(decodedResp)
	if err != nil {
		logger.Error("failed to marshal decoded response content", "error", err)
		return nil, err
	}

	return decodedContent, nil
}

// cleanOldFiles removes any old response files or flags from previous API requests.
// It ensures the environment is clean before processing new requests.
func cleanOldFiles(fs afero.Fs, dr *resolver.DependencyResolver, logger *logging.Logger) error {
	if _, err := fs.Stat(dr.ResponseTargetFile); err == nil {
		if err := fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			logger.Error("unable to delete old response file", "response-target-file", dr.ResponseTargetFile)
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

	return "", fmt.Errorf(`HTTP method "%s" not allowed`, r.Method)
}

// formatHeaders formats the HTTP headers into a string representation for inclusion in the .pkl file.
func formatHeaders(headers map[string][]string) string {
	var headersLines []string
	for name, values := range headers {
		for _, value := range values {
			encodedValue := utils.EncodeBase64String(strings.TrimSpace(value))
			headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, encodedValue))
		}
	}

	return "headers {\n" + strings.Join(headersLines, "\n") + "\n}"
}

// formatParams formats the query parameters into a string representation for inclusion in the .pkl file.
func formatParams(params map[string][]string) string {
	var paramsLines []string
	for param, values := range params {
		for _, value := range values {
			encodedValue := utils.EncodeBase64String(strings.TrimSpace(value))
			paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, encodedValue))
		}
	}
	return "params {\n" + strings.Join(paramsLines, "\n") + "\n}"
}

// processWorkflow handles the execution of the workflow steps after the .pkl file is created.
// It prepares the workflow directory, imports necessary files, and processes the actions defined in the workflow.
func processWorkflow(ctx context.Context, dr *resolver.DependencyResolver, logger *logging.Logger) (bool, error) {
	dr.Context = ctx

	if err := dr.PrepareWorkflowDir(); err != nil {
		return false, err
	}

	if err := dr.PrepareImportFiles(); err != nil {
		return false, err
	}

	//nolint:contextcheck // context already passed via dr.Context
	fatal, err := dr.HandleRunAction()
	if err != nil {
		return fatal, err
	}

	//nolint:contextcheck // context already passed via dr.Context
	stdout, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		logger.Fatal(fmt.Errorf(stdout, err))
		return true, err
	}

	logger.Debug("awaiting response...")

	// Wait for the response file to be ready
	if err := utils.WaitForFileReady(dr.Fs, ctx, dr.ResponseTargetFile, logger); err != nil {
		return false, err
	}

	return fatal, nil
}

// formatResponseJSON attempts to format the response content as JSON if required.
// It unmarshals the content into a map, modifies the "data" field if necessary, and re-encodes it into a pretty-printed JSON string.
func formatResponseJSON(content []byte) []byte {
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
	modifiedContent, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return content
	}

	return modifiedContent
}
