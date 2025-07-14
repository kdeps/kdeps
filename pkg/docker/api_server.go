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
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/config"
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

	// Use the new configuration processor for PKL-first config
	processor := config.NewConfigurationProcessor(dr.Logger)
	processedConfig, err := processor.ProcessWorkflowConfiguration(ctx, dr.Workflow)
	if err != nil {
		return err
	}

	// Validate configuration
	if err := processor.ValidateConfiguration(processedConfig); err != nil {
		return err
	}

	// Use processedConfig for all config values
	hostIP := processedConfig.APIServerHostIP.Value

	// For Docker containers, override 127.0.0.1 to 0.0.0.0 to accept external connections
	if hostIP == "127.0.0.1" {
		hostIP = "0.0.0.0"
		dr.Logger.Debug("overriding API server host IP for Docker", "original", processedConfig.APIServerHostIP.Value, "new", hostIP)
	}

	portNum := processedConfig.APIServerPort.Value
	hostPort := hostIP + ":" + strconv.FormatUint(uint64(portNum), 10)

	// Create a semaphore channel to limit to 1 active connection
	semaphore := make(chan struct{}, 1)
	router := gin.Default()

	wfAPIServerCORS := wfAPIServer.CORS

	var routes []*apiserver.APIServerRoutes
	if wfAPIServer.Routes != nil {
		routes = *wfAPIServer.Routes
	}

	setupRoutes(router, ctx, wfAPIServerCORS, wfTrustedProxies, routes, dr, semaphore)

	dr.Logger.Printf("Starting API server on port %s", hostPort)
	go func() {
		dr.Logger.Printf("GOROUTINE STARTED: About to start router on %s", hostPort)
		if err := router.Run(hostPort); err != nil {
			dr.Logger.Error("failed to start API server", "error", err)
		} else {
			dr.Logger.Printf("GOROUTINE ENDED: Router stopped normally")
		}
	}()

	return nil
}

func setupRoutes(router *gin.Engine, ctx context.Context, wfAPIServerCORS *apiserver.CORS, wfTrustedProxies []string, routes []*apiserver.APIServerRoutes, dr *resolver.DependencyResolver, semaphore chan struct{}) {
	for _, route := range routes {
		if route == nil || route.Path == "" {
			dr.Logger.Error("route configuration is invalid", "route", route)
			continue
		}

		if wfAPIServerCORS != nil && wfAPIServerCORS.EnableCORS != nil && *wfAPIServerCORS.EnableCORS {
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

			var allowCredentials bool
			if wfAPIServerCORS.AllowCredentials != nil {
				allowCredentials = *wfAPIServerCORS.AllowCredentials
			} else {
				allowCredentials = pkg.DefaultAllowCredentials
			}

			router.Use(cors.New(cors.Config{
				AllowOrigins:     allowOrigins,
				AllowMethods:     allowMethods,
				AllowHeaders:     allowHeaders,
				ExposeHeaders:    exposeHeaders,
				AllowCredentials: allowCredentials,
				MaxAge: func() time.Duration {
					if wfAPIServerCORS.MaxAge != nil {
						return wfAPIServerCORS.MaxAge.GoDuration()
					}
					return pkg.DefaultMaxAge
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

		// Register routes based on the Methods field from the schema
		if route.Methods != nil {
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
		} else {
			// If no methods specified, register all methods for backward compatibility
			router.GET(route.Path, handler)
			router.POST(route.Path, handler)
			router.PUT(route.Path, handler)
			router.PATCH(route.Path, handler)
			router.DELETE(route.Path, handler)
			router.OPTIONS(route.Path, handler)
			router.HEAD(route.Path, handler)
		}

		dr.Logger.Printf("Route configured: %s", route.Path)
	}
}

func APIServerHandler(ctx context.Context, route *apiserver.APIServerRoutes, baseDr *resolver.DependencyResolver, semaphore chan struct{}) gin.HandlerFunc {
	// Validate route parameter
	if route == nil || route.Path == "" {
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

	// Determine allowed methods from the route configuration
	var allowedMethods []string
	if route.Methods != nil {
		allowedMethods = route.Methods
	} else {
		// If no methods specified, allow all methods for backward compatibility
		allowedMethods = []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		}
	}

	return func(c *gin.Context) {
		// CRITICAL: Add logging at the very start to see if handler is called
		baseDr.Logger.Printf("HANDLER CALLED: %s %s", c.Request.Method, c.Request.URL.Path)

		// Add panic recovery to ensure we always send a response
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				baseDr.Logger.Error("panic in API handler", "error", r, "stack", string(debug.Stack()))
				// Send a JSON error response
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"errors": []map[string]interface{}{
						{
							"code":     http.StatusInternalServerError,
							"message":  fmt.Sprintf("Internal server error: %v", r),
							"actionId": "unknown",
						},
					},
					"meta": map[string]interface{}{
						"requestID": uuid.New().String(),
					},
				})
			}
		}()

		// Initialize errors slice to collect all errors
		var errors []ErrorResponse

		graphID := uuid.New().String()
		baseLogger := logging.GetLogger()
		logger := baseLogger.With("requestID", graphID)

		// Log that we've entered the handler
		logger.Info("API request received", "method", c.Request.Method, "path", c.Request.URL.Path, "clientIP", c.ClientIP())

		// Ensure cleanup of request-specific errors when request completes
		defer func() {
			logger.Debug("cleaning up request-specific errors")
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
			c.Data(statusCode, "application/json; charset=utf-8", jsonBytes)
		}

		// Try to acquire the semaphore (non-blocking)
		logger.Debug("attempting to acquire semaphore")
		select {
		case semaphore <- struct{}{}:
			// Successfully acquired the semaphore
			logger.Debug("semaphore acquired successfully")
			defer func() {
				logger.Debug("releasing semaphore")
				<-semaphore
			}() // Release the semaphore when done
		default:
			// Semaphore is full, append error
			logger.Warn("semaphore is full, rejecting request")
			addUniqueError(&errors, http.StatusTooManyRequests, "Only one active connection is allowed", "unknown")
			sendErrorResponse(http.StatusTooManyRequests, errors)
			return
		}

		logger.Debug("creating new context and resolver")
		newCtx := ktx.UpdateContext(ctx, ktx.CtxKeyGraphID, graphID)

		dr, err := resolver.NewGraphResolver(baseDr.Fs, newCtx, baseDr.Environment, c, logger)
		if err != nil {
			logger.Error("failed to create resolver", "error", err)

			// Provide more specific error message for PKL syntax errors
			errorMessage := "Failed to initialize resolver"
			if strings.Contains(err.Error(), "Pkl Error") {
				errorMessage = "PKL syntax error in workflow configuration"
			} else if strings.Contains(err.Error(), "workflow.pkl") {
				errorMessage = "Failed to load workflow configuration"
			}

			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  errorMessage,
				ActionID: "unknown", // No resolver available yet
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}
		logger.Debug("resolver created successfully")

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

		logger.Debug("cleaning old files")
		if err := cleanOldFiles(dr); err != nil {
			logger.Error("failed to clean old files", "error", err)
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  "Failed to clean old files",
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		logger.Debug("validating HTTP method", "method", c.Request.Method, "allowedMethods", allowedMethods)
		method, err := validateMethod(c.Request, allowedMethods)
		if err != nil {
			logger.Error("method validation failed", "error", err)
			errors = append(errors, ErrorResponse{
				Code:     http.StatusBadRequest,
				Message:  err.Error(),
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusBadRequest, errors)
			return
		}
		logger.Debug("method validation passed", "method", method)

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

		logger.Debug("processing request body", "method", c.Request.Method, "contentType", c.GetHeader("Content-Type"))
		var bodyData string
		fileMap := make(map[string]struct{ Filename, Filetype string })

		switch c.Request.Method {
		case http.MethodGet:
			logger.Debug("processing GET request body")
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				logger.Error("failed to read GET request body", "error", err)
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
			logger.Debug("GET request body processed", "bodyLength", len(bodyData))

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

		urlSection := fmt.Sprintf(`Path = "%s"`, c.Request.URL.Path)
		clientIPSection := fmt.Sprintf(`IP = "%s"`, c.ClientIP())
		requestIDSection := fmt.Sprintf(`ID = "%s"`, graphID)
		dataSection := fmt.Sprintf(`Data = "%s"`, utils.EncodeBase64String(bodyData))

		var sb strings.Builder
		sb.WriteString("Files {\n")
		for _, fileInfo := range fileMap {
			fileBlock := fmt.Sprintf(`
	Filepath = "%s"
	Filetype = "%s"
`, fileInfo.Filename, fileInfo.Filetype)
			sb.WriteString(fmt.Sprintf("    [\"%s\"] {\n%s\n}\n", filepath.Base(fileInfo.Filename), fileBlock))
		}
		sb.WriteString("}\n")
		fileSection := sb.String()

		paramSection := utils.FormatRequestParams(c.Request.URL.Query())
		requestHeaderSection := utils.FormatRequestHeaders(c.Request.Header)

		sections := []string{urlSection, clientIPSection, requestIDSection, method, requestHeaderSection, dataSection, paramSection, fileSection}

		logger.Debug("creating and processing PKL file")

		// Create a wrapper function that matches the expected signature
		evalFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
			return evaluator.EvalPkl(fs, ctx, tmpFile, headerSection, nil, logger)
		}

		if err := evaluator.CreateAndProcessPklFile(dr.Fs, ctx, sections, dr.RequestPklFile,
			"APIServerRequest.pkl", dr.Logger, evalFunc, true); err != nil {
			logger.Error("failed to create and process PKL file", "error", err)
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrProcessRequestFile,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}
		logger.Debug("PKL file created and processed successfully")

		logger.Debug("processing workflow")
		if err := processWorkflow(ctx, dr); err != nil {
			logger.Error("workflow processing failed", "error", err)
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
		logger.Debug("workflow processing completed successfully")

		logger.Debug("reading response file", "file", dr.ResponseTargetFile)
		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			logger.Error("failed to read response file", "error", err, "file", dr.ResponseTargetFile)
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrReadResponseFile,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}
		logger.Debug("response file read successfully", "contentLength", len(content))

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

		// Check if we have data to return
		hasData := decodedResp.Response.Data != nil && len(decodedResp.Response.Data) > 0

		// If there are critical errors AND no data, send error response (fail-fast behavior)
		if len(errors) > 0 && !hasData {
			// Add generic context error for fail-fast scenarios
			addUniqueError(&errors, http.StatusInternalServerError, messages.ErrEmptyResponse, getActionID())
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		// If we have data but also have errors, log the errors but continue with the response
		if len(errors) > 0 && hasData {
			logger.Warn("Response contains errors but also has data, returning data", "errors", errors, "dataLength", len(decodedResp.Response.Data))
			// Clear non-critical errors if we have data to return
			errors = []ErrorResponse{}
		}

		if decodedResp.Meta.Headers != nil {
			for key, value := range decodedResp.Meta.Headers {
				c.Header(key, value)
			}
		}

		// Ensure requestID is set in the response
		decodedResp.Meta.RequestID = graphID

		logger.Debug("marshaling response content")
		decodedContent, err := json.Marshal(decodedResp)
		if err != nil {
			logger.Error("failed to marshal response content", "error", err)
			errors = append(errors, ErrorResponse{
				Code:     http.StatusInternalServerError,
				Message:  messages.ErrMarshalResponseContent,
				ActionID: getActionID(),
			})
			sendErrorResponse(http.StatusInternalServerError, errors)
			return
		}

		decodedContent = formatResponseJSON(decodedContent)
		logger.Debug("sending successful response", "contentLength", len(decodedContent))
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
			return fmt.Sprintf(`Method = "%s"`, allowedMethod), nil
		}
	}

	return "", fmt.Errorf(`HTTP method "%s" not allowed`, r.Method)
}

// processWorkflow handles the execution of the workflow steps after the .pkl file is created.
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

	// Close the singleton evaluator after response file evaluation
	if evaluatorMgr, err := evaluator.GetEvaluatorManager(); err == nil {
		if err := evaluatorMgr.Close(); err != nil {
			dr.Logger.Error("failed to close PKL evaluator", "error", err)
		}
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

	// Check if content is PCF format (starts with "Success = " or similar)
	contentStr := string(content)
	if strings.Contains(contentStr, "Success = ") || strings.Contains(contentStr, "Meta {") {
		// Parse PCF format
		pcfResp, err := parsePCFResponse(contentStr, logger)
		if err != nil {
			logger.Error("failed to parse PCF response", "error", err)
			return nil, err
		}
		return pcfResp, nil
	}

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

// parsePCFResponse parses PCF format response and converts it to APIResponse
func parsePCFResponse(pcfContent string, logger *logging.Logger) (*APIResponse, error) {
	resp := &APIResponse{
		Success: false,
		Response: ResponseData{
			Data: []string{},
		},
		Meta: ResponseMeta{
			RequestID: "",
		},
		Errors: []ErrorResponse{},
	}

	lines := strings.Split(pcfContent, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Parse Success
		if strings.HasPrefix(line, "Success = ") {
			successStr := strings.TrimSpace(strings.TrimPrefix(line, "Success = "))
			resp.Success = successStr == "true"
		}

		// Parse RequestID
		if strings.Contains(line, "RequestID = ") {
			parts := strings.Split(line, "RequestID = ")
			if len(parts) > 1 {
				requestID := strings.Trim(strings.TrimSpace(parts[1]), `"`)
				resp.Meta.RequestID = requestID
			}
		}

		// Parse Data array
		if strings.Contains(line, "document.jsonRenderDocument") {
			// Extract the JSON content from the document.jsonRenderDocument call
			// Look for the JSON content in the next few lines
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				dataLine := strings.TrimSpace(lines[j])
				if strings.Contains(dataLine, `"`) {
					// Extract the quoted string content
					start := strings.Index(dataLine, `"`)
					end := strings.LastIndex(dataLine, `"`)
					if start != -1 && end != -1 && end > start {
						jsonContent := dataLine[start+1 : end]
						// Decode any Base64 content
						if decoded, err := utils.DecodeBase64String(jsonContent); err == nil {
							resp.Response.Data = append(resp.Response.Data, decoded)
						} else {
							resp.Response.Data = append(resp.Response.Data, jsonContent)
						}
					}
					break
				}
			}
		}

		// Parse Errors
		if strings.Contains(line, "Code = ") {
			// Extract error code and message
			codeStr := ""
			messageStr := ""

			// Look for Code and Message in the next few lines
			for j := i; j < len(lines) && j < i+10; j++ {
				errLine := strings.TrimSpace(lines[j])
				if strings.HasPrefix(errLine, "Code = ") {
					codeStr = strings.TrimSpace(strings.TrimPrefix(errLine, "Code = "))
				}
				if strings.HasPrefix(errLine, "Message = ") {
					// Extract message content between """# and """#
					start := strings.Index(errLine, `"""#`)
					end := strings.LastIndex(errLine, `"""#`)
					if start != -1 && end != -1 && end > start {
						messageStr = errLine[start+4 : end]
						break
					}
				}
			}

			if codeStr != "" {
				if code, err := strconv.Atoi(codeStr); err == nil {
					resp.Errors = append(resp.Errors, ErrorResponse{
						Code:    code,
						Message: messageStr,
					})
				}
			}
		}
	}

	return resp, nil
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
