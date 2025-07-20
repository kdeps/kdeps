package docker

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
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

// HandleMultipartForm processes multipart form data and updates fileMap.
// It returns a handlerError to be appended to the errors slice.
func HandleMultipartForm(c *gin.Context, dr *resolver.DependencyResolver, fileMap map[string]struct{ Filename, Filetype string }) error {
	form, err := c.MultipartForm()
	if err != nil {
		return &handlerError{http.StatusInternalServerError, "Unable to parse multipart form"}
	}

	// Handle multiple files from "file[]"
	if files := form.File["file[]"]; len(files) > 0 {
		for _, fileHeader := range files {
			if err := ProcessFile(fileHeader, dr, fileMap); err != nil {
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
	return ProcessFile(fileHeader, dr, fileMap)
}

// ProcessFile processes an individual file and updates fileMap.
// It returns a handlerError to be appended to the errors slice.
func ProcessFile(fileHeader *multipart.FileHeader, dr *resolver.DependencyResolver, fileMap map[string]struct{ Filename, Filetype string }) error {
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

	if err := dr.Fs.MkdirAll(filesPath, pkg.DefaultOctalDirPerms); err != nil {
		return &handlerError{http.StatusInternalServerError, "Unable to create files directory"}
	}

	if err := afero.WriteFile(dr.Fs, filename, fileBytes, pkg.DefaultOctalFilePerms); err != nil {
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
		return stdErrors.New("the API server configuration is missing")
	}

	wfAPIServer := wfSettings.APIServer
	if wfAPIServer == nil {
		return stdErrors.New("the API server configuration is missing")
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

	// For Docker containers, override default host IP to Docker host IP to accept external connections
	if hostIP == pkg.DefaultHostIP {
		hostIP = pkg.DefaultDockerHostIP
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

	SetupRoutes(router, ctx, wfAPIServerCORS, wfTrustedProxies, routes, dr, semaphore)

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

func SetupRoutes(router *gin.Engine, ctx context.Context, wfAPIServerCORS *apiserver.CORS, wfTrustedProxies []string, routes []*apiserver.APIServerRoutes, dr *resolver.DependencyResolver, semaphore chan struct{}) {
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
			c.Header("Content-Type", pkg.DefaultContentType)
			c.AbortWithStatus(http.StatusInternalServerError)
			_, _ = c.Writer.Write(jsonBytes)
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
			c.Header("Content-Type", pkg.DefaultContentType)
			c.Data(statusCode, pkg.DefaultContentType, jsonBytes)
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

		dr, err := resolver.NewGraphResolver(baseDr.Fs, newCtx, baseDr.Environment, c, logger, baseDr.Evaluator)
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

		// Note: PklresReader is now global and should not be closed per request

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
		if err := CleanOldFiles(dr); err != nil {
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
		method, err := ValidateMethod(c.Request, allowedMethods)
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
			c.Header("Allow", pkg.DefaultHTTPMethods)
			c.Status(http.StatusNoContent)
			return
		}

		if c.Request.Method == http.MethodHead {
			c.Header("Content-Type", pkg.DefaultContentType)
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
				if err := HandleMultipartForm(c, dr, fileMap); err != nil {

					var handlerErr *handlerError
					if stdErrors.As(err, &handlerErr) {
						errors = append(errors, ErrorResponse{
							Code:     handlerErr.statusCode,
							Message:  handlerErr.message,
							ActionID: getActionID(),
						})
						sendErrorResponse(handlerErr.statusCode, errors)
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
		dataSection := fmt.Sprintf(`Data = "%s"`, bodyData)

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

		logger.Debug("creating and processing PKL file", "evaluator_nil", dr.Evaluator == nil)

		// Create a wrapper function that matches the expected signature
		evalFunc := func(eval pkl.Evaluator, fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
			return evaluator.EvalPkl(eval, fs, ctx, tmpFile, headerSection, nil, logger)
		}

		if err := evaluator.CreateAndProcessPklFile(dr.Evaluator, dr.Fs, ctx, sections, dr.RequestPklFile,
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
		if err := ProcessWorkflow(ctx, dr); err != nil {
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

		decodedResp, err := DecodeResponseContent(content, dr.Logger)
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

		decodedContent = FormatResponseJSON(decodedContent)
		logger.Debug("sending successful response", "contentLength", len(decodedContent))
		c.Data(http.StatusOK, pkg.DefaultContentType, decodedContent)
	}
}

// CleanOldFiles removes old response files
func CleanOldFiles(dr *resolver.DependencyResolver) error {
	if dr.ResponseTargetFile == "" {
		return nil
	}

	if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
		return fmt.Errorf("failed to clean old files: %w", err)
	}

	return nil
}

// ValidateMethod validates the HTTP method against allowed methods
func ValidateMethod(r *http.Request, allowedMethods []string) (string, error) {
	method := r.Method
	if method == "" {
		method = "GET"
	}

	for _, allowed := range allowedMethods {
		if strings.EqualFold(method, allowed) {
			return fmt.Sprintf(`Method = "%s"`, strings.ToUpper(method)), nil
		}
	}

	return "", fmt.Errorf("HTTP method \"%s\" not allowed", method)
}

// ProcessWorkflow processes the workflow
func ProcessWorkflow(_ context.Context, dr *resolver.DependencyResolver) error {
	// In API server mode, populate request data in pklres before any resource evaluation
	if dr.APIServerMode && dr.RequestPklFile != "" {
		dr.Logger.Debug("populating request data in pklres before workflow processing")
		if err := dr.PopulateRequestDataInPklres(); err != nil {
			dr.Logger.Warn("failed to populate request data in pklres", "error", err)
			// Don't fail the workflow for this error, but log it
		} else {
			dr.Logger.Debug("successfully populated request data in pklres")
		}
	}

	// Process the workflow
	_, err := dr.HandleRunAction()
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	return nil
}

// DecodeResponseContent decodes response content
func DecodeResponseContent(content []byte, logger *logging.Logger) (*APIResponse, error) {
	var response APIResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Remove all base64 decoding logic. Use plain text for all data handling. Remove IsBase64Encoded, DecodeBase64String, and related logic. Remove any code that checks for or decodes base64 prefixes.

	return &response, nil
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

		// Parse Data array - look for quoted strings directly
		if strings.Contains(line, "Data {") {
			// Look for data content in the following lines
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				dataLine := strings.TrimSpace(lines[j])
				if dataLine == "}" {
					break // End of Data block
				}
				if strings.Contains(dataLine, `"`) {
					// Extract the quoted string content
					start := strings.Index(dataLine, `"`)
					end := strings.LastIndex(dataLine, `"`)
					if start != -1 && end != -1 && end > start {
						jsonContent := dataLine[start+1 : end]
						// Use content directly without base64 decoding
						resp.Response.Data = append(resp.Response.Data, jsonContent)
					}
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

// FormatResponseJSON formats response JSON
func FormatResponseJSON(content []byte) []byte {
	var response APIResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return content
	}

	// Attempt to parse JSON string elements in data array and store as objects/arrays
	var newData []interface{}
	for _, data := range response.Response.Data {
		var obj interface{}
		if err := json.Unmarshal([]byte(data), &obj); err == nil {
			newData = append(newData, obj)
		} else {
			newData = append(newData, data)
		}
	}
	response.Response.Data = nil // clear original

	// Marshal with newData replacing Data
	// Use a map to marshal the struct with the new data array
	m := map[string]interface{}{}
	b, _ := json.Marshal(response)
	json.Unmarshal(b, &m)
	if resp, ok := m["response"].(map[string]interface{}); ok {
		resp["data"] = newData
	}
	formatted, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return content
	}
	return formatted
}
