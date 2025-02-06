package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserver "github.com/kdeps/schema/gen/api_server"
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

type handlerError struct {
	statusCode int
	message    string
}

func (e *handlerError) Error() string {
	return e.message
}

func handleMultipartForm(c *gin.Context, dr *resolver.DependencyResolver, fileMap map[string]struct{ Filename, Filetype string }) error {
	form, err := c.MultipartForm()
	if err != nil {
		return &handlerError{http.StatusInternalServerError, "Unable to parse multipart form"}
	}

	// Handle multiple files from "file[]"
	if files := form.File["file[]"]; len(files) > 0 {
		for _, fileHeader := range files {
			if err := processFile(fileHeader, dr, fileMap); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle single file from "file"
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return &handlerError{http.StatusBadRequest, "No file uploaded"}
	}
	return processFile(fileHeader, dr, fileMap)
}

func processFile(fileHeader *multipart.FileHeader, dr *resolver.DependencyResolver, fileMap map[string]struct{ Filename, Filetype string }) error {
	file, err := fileHeader.Open()
	if err != nil {
		return &handlerError{http.StatusInternalServerError, fmt.Sprintf("Unable to open file: %v", err)}
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return &handlerError{http.StatusInternalServerError, "Failed to read file content"}
	}

	filetype := mimetype.Detect(fileBytes).String()
	filesPath := filepath.Join(dr.ActionDir, "files")
	filename := filepath.Join(filesPath, fileHeader.Filename)

	if err := dr.Fs.MkdirAll(filesPath, 0o777); err != nil {
		return &handlerError{http.StatusInternalServerError, "Unable to create files directory"}
	}

	if err := afero.WriteFile(dr.Fs, filename, fileBytes, 0o644); err != nil {
		return &handlerError{http.StatusInternalServerError, "Failed to save file"}
	}

	fileMap[filename] = struct{ Filename, Filetype string }{filename, filetype}
	return nil
}

// StartAPIServerMode initializes and starts an API server based on the provided workflow configuration.
// It validates the API server configuration, sets up routes, and starts the server on the configured port.
// StartAPIServerMode initializes and starts a Gin-based API server.
func StartAPIServerMode(ctx context.Context, dr *resolver.DependencyResolver) error {
	wfSettings := dr.Workflow.GetSettings()
	wfAPIServer := wfSettings.APIServer

	if wfAPIServer == nil {
		return errors.New("the API server configuration is missing")
	}

	portNum := strconv.FormatUint(uint64(wfAPIServer.PortNum), 10)
	hostPort := ":" + portNum

	router := gin.Default()
	setupRoutes(router, ctx, wfAPIServer.Routes, dr)

	dr.Logger.Printf("Starting API server on port %s", hostPort)
	go func() {
		if err := router.Run(hostPort); err != nil {
			dr.Logger.Error("failed to start API server", "error", err)
		}
	}()

	return nil
}

func setupRoutes(router *gin.Engine, ctx context.Context, routes []*apiserver.APIServerRoutes, dr *resolver.DependencyResolver) {
	for _, route := range routes {
		if route == nil || route.Path == "" {
			dr.Logger.Error("route configuration is invalid", "route", route)
			continue
		}

		handler := APIServerHandler(ctx, route, dr)
		for _, method := range route.Methods {
			switch method {
			case http.MethodGet:
				router.GET(route.Path, handler)
			case http.MethodPost:
				router.POST(route.Path, handler)
			case http.MethodPut:
				router.PUT(route.Path, handler)
			case http.MethodPatch:
				router.PATCH(route.Path, handler)
			case http.MethodDelete:
				router.DELETE(route.Path, handler)
			case http.MethodOptions:
				router.OPTIONS(route.Path, handler)
			case http.MethodHead:
				router.HEAD(route.Path, handler)
			default:
				dr.Logger.Warn("Unsupported HTTP method in route configuration", "method", method)
			}
		}

		dr.Logger.Printf("Route configured: %s", route.Path)
	}
}

func APIServerHandler(ctx context.Context, route *apiserver.APIServerRoutes, dr *resolver.DependencyResolver) gin.HandlerFunc {
	allowedMethods := route.Methods

	return func(c *gin.Context) {
		if err := cleanOldFiles(dr); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean old files"})
			return
		}

		method, err := validateMethod(c.Request, allowedMethods)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if c.Request.Method == http.MethodOptions {
			c.Header("Allow", "OPTIONS, GET, HEAD, POST, PUT, PATCH, DELETE")
			c.Status(http.StatusNoContent)
			return
		}

		if c.Request.Method == http.MethodHead {
			c.Header("Content-Type", "application/json")
			c.Status(http.StatusOK)
			return
		}

		var bodyData string
		fileMap := make(map[string]struct{ Filename, Filetype string })

		switch c.Request.Method {
		case http.MethodGet:
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
				return
			}
			defer c.Request.Body.Close()
			bodyData = string(body)

		case http.MethodPost, http.MethodPut, http.MethodPatch:
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "multipart/form-data") {
				if err := handleMultipartForm(c, dr, fileMap); err != nil {
					var he *handlerError
					if errors.As(err, &he) {
						c.AbortWithStatusJSON(he.statusCode, gin.H{"error": he.message})
					} else {
						c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					}
					return
				}
			} else {
				// Read non-multipart body
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
					return
				}
				defer c.Request.Body.Close()
				bodyData = string(body)
			}

		case http.MethodDelete:
			bodyData = "Delete request received"
		default:
			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{"error": "Unsupported method"})
			return
		}

		urlSection := fmt.Sprintf(`path = "%s"`, c.Request.URL.Path)
		dataSection := fmt.Sprintf(`data = "%s"`, utils.EncodeBase64String(bodyData))

		var sb strings.Builder
		sb.WriteString("files {\n")
		for _, fileInfo := range fileMap {
			fileBlock := fmt.Sprintf(`
	filepath = "%s"
	filetype = "%s"
`, fileInfo.Filename, fileInfo.Filetype)
			sb.WriteString(fmt.Sprintf("    [\"%s\"] {\n%s\n}\n", filepath.Base(fileInfo.Filename), fileBlock))
		}
		sb.WriteString("}\n")
		fileSection := sb.String()

		paramSection := formatParams(c.Request.URL.Query())
		headerSection := formatHeaders(c.Request.Header)

		sections := []string{urlSection, method, headerSection, dataSection, paramSection, fileSection}

		if err := evaluator.CreateAndProcessPklFile(dr.Fs, ctx, sections, dr.RequestPklFile,
			"APIServerRequest.pkl", dr.Logger, evaluator.EvalPkl, true); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request file"})
			return
		}

		fatal, err := processWorkflow(ctx, dr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Workflow processing failed"})
			return
		}

		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response file"})
			return
		}

		decodedContent, err := decodeResponseContent(content, dr.Logger)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response content"})
			return
		}

		decodedContent = formatResponseJSON(decodedContent)
		c.Data(http.StatusOK, "application/json", decodedContent)

		if fatal {
			if removeErr := dr.Fs.RemoveAll(dr.ActionDir); removeErr != nil {
				dr.Logger.Warn("failed to clean up temporary directory", "path", dr.ActionDir, "error", removeErr)
			}
			dr.Logger.Error("a fatal server error occurred. Restarting the service.")
			utils.SendSigterm(dr.Logger)
		}
	}
}

// cleanOldFiles removes any old response files or flags from previous API requests.
// It ensures the environment is clean before processing new requests.
func cleanOldFiles(dr *resolver.DependencyResolver) error {
	if _, err := dr.Fs.Stat(dr.ResponseTargetFile); err == nil {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("unable to delete old response file", "response-target-file", dr.ResponseTargetFile)
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
func processWorkflow(ctx context.Context, dr *resolver.DependencyResolver) (bool, error) {
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

	stdout, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		dr.Logger.Fatal(fmt.Errorf(stdout, err))
		return true, err
	}

	dr.Logger.Debug("awaiting response...")

	// Wait for the response file to be ready
	if err := utils.WaitForFileReady(dr.Fs, dr.ResponseTargetFile, dr.Logger); err != nil {
		return false, err
	}

	return fatal, nil
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
			// Pass in the invalidJSON to the FixJSON
			fixedJSON := utils.FixJSON(decodedData)

			// If the decoded data is JSON, pretty print it
			if utils.IsJSON(fixedJSON) {
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
