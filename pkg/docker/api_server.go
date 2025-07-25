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
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserver "github.com/kdeps/schema/gen/api_server"
	"github.com/spf13/afero"
)

// ErrorResponse defines the structure of each error.
type ErrorResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	ActionID string `json:"actionId,omitempty"`
}

// APIResponse defines the overall response structure.
type APIResponse struct {
	Success  bool            `json:"success"`
	Response ResponseData    `json:"response"`
	Meta     ResponseMeta    `json:"meta"`
	Errors   []ErrorResponse `json:"errors,omitempty"`
}

// ResponseData encapsulates the data section of the response.
type ResponseData struct {
	Data []string `json:"data"`
}

// ResponseMeta contains metadata related to the API response.
type ResponseMeta struct {
	RequestID  string            `json:"requestID"`
	Headers    map[string]string `json:"headers,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

type handlerError struct {
	statusCode int
	message    string
}

func (e *handlerError) Error() string {
	return e.message
}

// handleMultipartForm processes multipart form data and updates fileMap.
// It returns a handlerError to be appended to the errors slice.
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

// processFile processes an individual file and updates fileMap.
// It returns a handlerError to be appended to the errors slice.
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
func StartAPIServerMode(ctx context.Context, dr *resolver.DependencyResolver) error {
	wfSettings := dr.Workflow.GetSettings()
	if wfSettings == nil {
		return errors.New("the API server configuration is missing")
	}

	wfAPIServer := wfSettings.APIServer
	if wfAPIServer == nil {
		return errors.New("the API server configuration is missing")
	}

	var wfTrustedProxies []string
	if wfAPIServer.TrustedProxies != nil {
		wfTrustedProxies = *wfAPIServer.TrustedProxies
	}

	portNum := strconv.FormatUint(uint64(wfAPIServer.PortNum), 10)
	hostPort := ":" + portNum

	// Create a semaphore channel to limit to 1 active connection
	semaphore := make(chan struct{}, 1)
	router := gin.Default()

	wfAPIServerCORS := wfAPIServer.CORS

	setupRoutes(router, ctx, wfAPIServerCORS, wfTrustedProxies, wfAPIServer.Routes, dr, semaphore)

	dr.Logger.Printf("Starting API server on port %s", hostPort)
	go func() {
		if err := router.Run(hostPort); err != nil {
			dr.Logger.Error("failed to start API server", "error", err)
		}
	}()

	return nil
}

func setupRoutes(router *gin.Engine, ctx context.Context, wfAPIServerCORS *apiserver.CORSConfig, wfTrustedProxies []string, routes []*apiserver.APIServerRoutes, dr *resolver.DependencyResolver, semaphore chan struct{}) {
	for _, route := range routes {
		if route == nil || route.Path == "" {
			dr.Logger.Error("route configuration is invalid", "route", route)
			continue
		}

		if wfAPIServerCORS != nil && wfAPIServerCORS.EnableCORS {
			var allowOrigins, allowMethods, allowHeaders, exposeHeaders []string

			if wfAPIServerCORS.AllowOrigins != nil {
				allowOrigins = *wfAPIServerCORS.AllowOrigins
			}
			if wfAPIServerCORS.AllowMethods != nil {
				allowMethods = *wfAPIServerCORS.AllowMethods
			}
			if wfAPIServerCORS.AllowHeaders != nil {
				allowHeaders = *wfAPIServerCORS.AllowHeaders
			}
			if wfAPIServerCORS.ExposeHeaders != nil {
				exposeHeaders = *wfAPIServerCORS.ExposeHeaders
			}

			router.Use(cors.New(cors.Config{
				AllowOrigins:     allowOrigins,
				AllowMethods:     allowMethods,
				AllowHeaders:     allowHeaders,
				ExposeHeaders:    exposeHeaders,
				AllowCredentials: wfAPIServerCORS.AllowCredentials,
				MaxAge: func() time.Duration {
					if wfAPIServerCORS.MaxAge != nil {
						return wfAPIServerCORS.MaxAge.GoDuration()
					}
					return 12 * time.Hour
				}(),
			}))
		}

		if len(wfTrustedProxies) > 0 {
			dr.Logger.Printf("Found trusted proxies %v", wfTrustedProxies)

			router.ForwardedByClientIP = true
			if err := router.SetTrustedProxies(wfTrustedProxies); err != nil {
				dr.Logger.Error("unable to set trusted proxies")
			}
		}

		handler := APIServerHandler(ctx, route, dr, semaphore)
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

func APIServerHandler(ctx context.Context, route *apiserver.APIServerRoutes, baseDr *resolver.DependencyResolver, semaphore chan struct{}) gin.HandlerFunc {
	// Validate route parameter
	if route == nil || route.Path == "" || len(route.Methods) == 0 {
		baseDr.Logger.Error("invalid route configuration provided to APIServerHandler", "route", route)
		return func(c *gin.Context) {
			graphID := uuid.New().String()
			response := APIResponse{
				Success: false,
				Response: ResponseData{
					Data: nil,
				},
				Meta: ResponseMeta{
					RequestID: graphID,
				},
				Errors: []ErrorResponse{
					{
						Code:     http.StatusInternalServerError,
						Message:  "Invalid route configuration",
						ActionID: "unknown", // No action context available for route configuration errors
					},
				},
			}
			jsonBytes, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, response)
				return
			}
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.AbortWithStatus(http.StatusInternalServerError)
			c.Writer.Write(jsonBytes)
		}
	}

	allowedMethods := route.Methods

	return func(c *gin.Context) {
		// Initialize errors slice to collect all errors
		var errors []ErrorResponse

		graphID := uuid.New().String()
		baseLogger := logging.GetLogger()
		logger := baseLogger.With("requestID", graphID)

		// Ensure cleanup of request-specific errors when request completes
		defer func() {
			utils.ClearRequestErrors(graphID)
		}()

		// Helper function to create APIResponse with requestID
		createErrorResponse := func(errs []ErrorResponse) APIResponse {
			return APIResponse{
				Success: false,
				Response: ResponseData{
					Data: nil,
				},
				Meta: ResponseMeta{
					RequestID: graphID,
				},
				Errors: errs,
			}
		}

		// Helper function to add unique errors (prevents duplicates)
		// Action ID will be added after resolver is created
		addUniqueError := func(errs *[]ErrorResponse, code int, message, actionID string) {
			// Skip empty messages
			if message == "" {
				return
			}

			// Use "unknown" if actionID is empty
			if actionID == "" {
				actionID = "unknown"
			}

			// Check if error already exists (same message, code, and actionID)
			for _, existingError := range *errs {
				if existingError.Message == message && existingError.Code == code && existingError.ActionID == actionID {
					return // Skip duplicate
				}
			}

			// Add new unique error
			*errs = append(*errs, ErrorResponse{
				Code:     code,
				Message:  message,
				ActionID: actionID,
			})
		}

		// Helper function to send properly formatted JSON error responses
		sendErrorResponse := func(statusCode int, errs []ErrorResponse) {
			response := createErrorResponse(errs)
			jsonBytes, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				// Fallback to non-indented JSON if marshal fails
				c.AbortWithStatusJSON(statusCode, response)
				return
			}
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.AbortWithStatus(statusCode)
			c.Writer.Write(jsonBytes)
		}

		// Try to acquire the semaphore (non-blocking)
		select {
		case semaphore <- struct{}{}:
			// Successfully acquired the semaphore
			defer func() { <-semaphore }() // Release the semaphore when done
		default:
			// Semaphore is full, append error
			addUniqueError(&errors, http.StatusTooManyRequests, "Only one active connection is allowed", "unknown")
			sendErrorResponse(http.StatusTooManyRequests, errors)
			return
		}

		newCtx := ktx.UpdateContext(ctx, ktx.CtxKeyGraphID, graphID)

		dr, err := resolver.NewGraphResolver(baseDr.Fs, newCtx, baseDr.Environment, c, logger)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  "Failed to initialize resolver",
				ActionID: "unknown", // No resolver available yet
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		// Helper function to get action ID safely
		getActionID := func() string {
			if dr != nil {
				// First try to get the current resource actionID being processed
				if dr.CurrentResourceActionID != "" {
					return dr.CurrentResourceActionID
				}
				// Fall back to workflow's target action ID if no current resource
				if dr.Workflow != nil {
					actionID := dr.Workflow.GetTargetActionID()
					if actionID != "" {
						return actionID
					}
				}
			}
			return "unknown"
		}

		if err := cleanOldFiles(dr); err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  "Failed to clean old files",
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		method, err := validateMethod(c.Request, allowedMethods)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusBadRequest,
				Message:  err.Error(),
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusBadRequest, errors)
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
				errors = append(errors, ErrorResponse{
					Code:     http.StatusBadRequest,
					Message:  "Failed to read request body",
					ActionID: getActionID(),
				})
				sendErrorResponse(http.StatusBadRequest, errors)
				return
			}
			defer c.Request.Body.Close()
			bodyData = string(body)

		case http.MethodPost, http.MethodPut, http.MethodPatch:
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "multipart/form-data") {
				if err := handleMultipartForm(c, dr, fileMap); err != nil {

					if he, ok := err.(*handlerError); ok {
						errors = append(errors, ErrorResponse{
							Code:     he.statusCode,
							Message:  he.message,
							ActionID: getActionID(),
						})
						sendErrorResponse(he.statusCode, errors)
					} else {
						errors = append(errors, ErrorResponse{
							Code:     http.StatusInternalServerError,
							Message:  err.Error(),
							ActionID: getActionID(),
						})
						sendErrorResponse(http.StatusInternalServerError, errors)
					}
					return
				}
			} else {
				// Read non-multipart body
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					errors = append(errors, ErrorResponse{
						Code:     http.StatusBadRequest,
						Message:  "Failed to read request body",
						ActionID: getActionID(),
					})
					sendErrorResponse(http.StatusBadRequest, errors)
					return
				}
				defer c.Request.Body.Close()
				bodyData = string(body)
			}

		case http.MethodDelete:
			bodyData = "Delete request received"
		default:
			errors = append(errors, ErrorResponse{
				Code:     http.StatusMethodNotAllowed,
				Message:  "Unsupported method",
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusMethodNotAllowed, errors)
			return
		}

		urlSection := fmt.Sprintf(`path = "%s"`, c.Request.URL.Path)
		clientIPSection := fmt.Sprintf(`IP = "%s"`, c.ClientIP())
		requestIDSection := fmt.Sprintf(`ID = "%s"`, graphID)
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

		paramSection := utils.FormatRequestParams(c.Request.URL.Query())
		requestHeaderSection := utils.FormatRequestHeaders(c.Request.Header)

		sections := []string{urlSection, clientIPSection, requestIDSection, method, requestHeaderSection, dataSection, paramSection, fileSection}

		if err := evaluator.CreateAndProcessPklFile(dr.Fs, ctx, sections, dr.RequestPklFile,
			"APIServerRequest.pkl", dr.Logger, evaluator.EvalPkl, true); err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrProcessRequestFile,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		if err := processWorkflow(ctx, dr); err != nil {
			// Get action ID for error context
			actionID := getActionID()

			// Add the specific error first (if not empty and unique)
			errorMessage := err.Error()
			addUniqueError(&errors, http.StatusInternalServerError, errorMessage, actionID)

			// Add the generic error message as additional context (if unique)
			addUniqueError(&errors, http.StatusInternalServerError, messages.ErrEmptyResponse, actionID)

			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrReadResponseFile,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		decodedResp, err := decodeResponseContent(content, dr.Logger)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrDecodeResponseContent,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		// Always check for all accumulated errors from workflow processing
		// This includes preflight validation errors, exec errors, python errors, etc.
		allAccumulatedErrors := utils.GetRequestErrorsWithActionID(graphID)

		// Convert accumulated errors to our ErrorResponse format using their captured actionID
		for _, accError := range allAccumulatedErrors {
			if accError != nil {
				// Use the actionID that was captured when the error was created
				actionID := accError.ActionID
				if actionID == "" {
					actionID = "unknown"
				}
				addUniqueError(&errors, accError.Code, accError.Message, actionID)
			}
		}

		// Merge APIResponse errors with workflow processing errors
		if len(decodedResp.Errors) > 0 {
			for _, apiError := range decodedResp.Errors {
				// Extract actionID from existing error if present, otherwise use current actionID
				actionID := apiError.ActionID
				if actionID == "" {
					actionID = getActionID()
				}
				addUniqueError(&errors, apiError.Code, apiError.Message, actionID)
			}
		}

		// If there are any errors (workflow or APIResponse), send error response (fail-fast behavior)
		if len(errors) > 0 {
			// Add generic context error for fail-fast scenarios
			addUniqueError(&errors, http.StatusInternalServerError, messages.ErrEmptyResponse, getActionID())
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		if decodedResp.Meta.Headers != nil {
			for key, value := range decodedResp.Meta.Headers {
				c.Header(key, value)
			}
		}

		// Ensure requestID is set in the response
		decodedResp.Meta.RequestID = graphID

		decodedContent, err := json.Marshal(decodedResp)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrMarshalResponseContent,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		decodedContent = formatResponseJSON(decodedContent)
		c.Data(http.StatusOK, "application/json; charset=utf-8", decodedContent)
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

// processWorkflow handles the execution of the workflow steps after the .pkl file is created.
// It prepares the workflow directory, imports necessary files, and processes the actions defined in the workflow.
func processWorkflow(ctx context.Context, dr *resolver.DependencyResolver) error {
	dr.Context = ctx

	if err := dr.PrepareWorkflowDir(); err != nil {
		return err
	}

	if err := dr.PrepareImportFiles(); err != nil {
		return err
	}

	// context already passed via dr.Context
	if _, err := dr.HandleRunAction(); err != nil {
		return err
	}

	stdout, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		dr.Logger.Errorf("%s: %v", stdout, err)
		return err
	}

	dr.Logger.Debug(messages.MsgAwaitingResponse)

	// Wait for the response file to be ready
	if err := utils.WaitForFileReady(dr.Fs, dr.ResponseTargetFile, dr.Logger); err != nil {
		return err
	}

	return nil
}

func decodeResponseContent(content []byte, logger *logging.Logger) (*APIResponse, error) {
	var decodedResp APIResponse

	// Unmarshal JSON content into APIResponse struct
	err := json.Unmarshal(content, &decodedResp)
	if err != nil {
		logger.Error(messages.ErrUnmarshalRespContent, "error", err)
		return nil, err
	}

	// Decode Base64 strings in the Data field
	for i, encodedData := range decodedResp.Response.Data {
		decodedData, err := utils.DecodeBase64String(encodedData)
		if err != nil {
			logger.Error(messages.ErrDecodeBase64String, "data", encodedData)
			decodedResp.Response.Data[i] = encodedData // Use original if decoding fails
		} else {
			fixedJSON := utils.FixJSON(decodedData)
			if utils.IsJSON(fixedJSON) {
				var prettyJSON bytes.Buffer
				err := json.Indent(&prettyJSON, []byte(fixedJSON), "", "  ")
				if err == nil {
					fixedJSON = prettyJSON.String()
				}
			}
			decodedResp.Response.Data[i] = fixedJSON
		}
	}

	return &decodedResp, nil
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
