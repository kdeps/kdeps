package docker

import (
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

	"github.com/apple/pkl-go/pkl"
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
	Code    int    `json:"code"`
	Message string `json:"message"`
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

// HandleMultipartForm handles multipart form processing
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

// ProcessFile processes an individual file
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

	if err := dr.Fs.MkdirAll(filesPath, 0o777); err != nil {
		return &handlerError{http.StatusInternalServerError, "Unable to create files directory"}
	}

	if err := afero.WriteFile(dr.Fs, filename, fileBytes, 0o644); err != nil {
		return &handlerError{http.StatusInternalServerError, "Failed to save file"}
	}

	fileMap[filename] = struct{ Filename, Filetype string }{filename, filetype}
	return nil
}

// StartAPIServerMode starts the API server
func StartAPIServerMode(ctx context.Context, dr *resolver.DependencyResolver) error {
	if dr.Workflow == nil {
		return errors.New("workflow is nil")
	}

	wfSettings := dr.Workflow.GetSettings()
	if wfSettings == nil {
		return errors.New("the API server configuration is missing")
	}

	wfAPIServer := wfSettings.APIServer
	var wfTrustedProxies []string
	if wfAPIServer.TrustedProxies != nil {
		wfTrustedProxies = *wfAPIServer.TrustedProxies
	}

	if wfAPIServer == nil {
		return errors.New("the API server configuration is missing")
	}

	portNum := strconv.FormatUint(uint64(wfAPIServer.PortNum), 10)
	hostPort := ":" + portNum

	// Create a semaphore channel to limit to 1 active connection
	semaphore := make(chan struct{}, 1)
	router := gin.Default()

	wfAPIServerCORS := wfAPIServer.Cors

	SetupRoutes(router, ctx, wfAPIServerCORS, wfTrustedProxies, wfAPIServer.Routes, dr, semaphore)

	dr.Logger.Printf(messages.MsgStartAPIServerOnPort, hostPort)
	go func() {
		if err := router.Run(hostPort); err != nil {
			dr.Logger.Error("failed to start API server", "error", err)
		}
	}()

	return nil
}

// SetupRoutes sets up the routes for the API server
func SetupRoutes(router *gin.Engine, ctx context.Context, wfAPIServerCORS *apiserver.CORS, wfTrustedProxies []string, routes []*apiserver.APIServerRoutes, dr *resolver.DependencyResolver, semaphore chan struct{}) {
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

// APIServerHandler creates a handler for API server routes
func APIServerHandler(ctx context.Context, route *apiserver.APIServerRoutes, baseDr *resolver.DependencyResolver, semaphore chan struct{}) gin.HandlerFunc {
	allowedMethods := route.Methods

	return func(c *gin.Context) {
		// Initialize errors slice to collect all errors
		var errors []ErrorResponse

		graphID := uuid.New().String()
		baseLogger := logging.GetLogger()
		logger := baseLogger.With("requestID", graphID)

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

		// Try to acquire the semaphore (non-blocking)
		select {
		case semaphore <- struct{}{}:
			// Successfully acquired the semaphore
			defer func() { <-semaphore }() // Release the semaphore when done
		default:
			// Semaphore is full, append error
			errors = append(errors, ErrorResponse{
				Code:    http.StatusTooManyRequests,
				Message: "Only one active connection is allowed",
			})
			c.AbortWithStatusJSON(http.StatusTooManyRequests, createErrorResponse(errors))
			return
		}

		newCtx := ktx.UpdateContext(ctx, ktx.CtxKeyGraphID, graphID)

		dr, err := resolver.NewGraphResolver(baseDr.Fs, newCtx, baseDr.Environment, c, logger)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to initialize resolver",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}

		if err := CleanOldFiles(dr); err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to clean old files",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}

		method, err := ValidateMethod(c.Request, allowedMethods)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			c.AbortWithStatusJSON(http.StatusBadRequest, createErrorResponse(errors))
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
					Code:    http.StatusBadRequest,
					Message: "Failed to read request body",
				})
				c.AbortWithStatusJSON(http.StatusBadRequest, createErrorResponse(errors))
				return
			}
			defer c.Request.Body.Close()
			bodyData = string(body)

		case http.MethodPost, http.MethodPut, http.MethodPatch:
			contentType := c.GetHeader("Content-Type")
			if strings.Contains(contentType, "multipart/form-data") {
				if err := HandleMultipartForm(c, dr, fileMap); err != nil {
					//nolint:errorlint
					if he, ok := err.(*handlerError); ok {
						errors = append(errors, ErrorResponse{
							Code:    he.statusCode,
							Message: he.message,
						})
						c.AbortWithStatusJSON(he.statusCode, createErrorResponse(errors))
					} else {
						errors = append(errors, ErrorResponse{
							Code:    http.StatusInternalServerError,
							Message: err.Error(),
						})
						c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
					}
					return
				}
			} else {
				// Read non-multipart body
				body, err := io.ReadAll(c.Request.Body)
				if err != nil {
					errors = append(errors, ErrorResponse{
						Code:    http.StatusBadRequest,
						Message: "Failed to read request body",
					})
					c.AbortWithStatusJSON(http.StatusBadRequest, createErrorResponse(errors))
					return
				}
				defer c.Request.Body.Close()
				bodyData = string(body)
			}

		case http.MethodDelete:
			bodyData = "Delete request received"
		default:
			errors = append(errors, ErrorResponse{
				Code:    http.StatusMethodNotAllowed,
				Message: "Unsupported method",
			})
			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, createErrorResponse(errors))
			return
		}

		urlSection := fmt.Sprintf(`path = "%s"`, c.Request.URL.Path)
		clientIPSection := fmt.Sprintf(`IP = "%s"`, c.ClientIP())
		requestIDSection := fmt.Sprintf(`ID = "%s"`, graphID)
		dataSection := fmt.Sprintf(`data = "%s"`, bodyData)

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

		paramSection := formatRequestParams(c.Request.URL.Query())
		requestHeaderSection := formatRequestHeaders(c.Request.Header)

		sections := []string{urlSection, clientIPSection, requestIDSection, method, requestHeaderSection, dataSection, paramSection, fileSection}

		if err := evaluator.CreateAndProcessPklFile(dr.Fs, ctx, sections, dr.RequestPklFile,
			"APIServerRequest.pkl", dr.EvaluatorOptions, dr.Logger, evaluator.EvalPkl, true, graphID); err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to process request file",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}

		if err := ProcessWorkflow(ctx, dr); err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Empty response received, possibly due to configuration issues. Please verify: 1. Allowed route paths and HTTP methods match the incoming request. 2. Skip validations that are skipping the required resource to produce the requests. 3. Timeout settings are sufficient for long-running processes (e.g., LLM operations).",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}

		content, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to read response file",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}

		pklEvaluator, err := pkl.NewEvaluator(ctx, dr.EvaluatorOptions)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to create PKL evaluator",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}
		defer pklEvaluator.Close()

		decodedResp, err := DecodeResponseContent(content, dr.Logger, pklEvaluator, ctx)
		if err != nil {
			errors = append(errors, ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Failed to decode response content",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
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
				Code:    http.StatusInternalServerError,
				Message: "Failed to marshal response content",
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, createErrorResponse(errors))
			return
		}

		decodedContent = FormatResponseJSON(decodedContent)
		c.Data(http.StatusOK, "application/json; charset=utf-8", decodedContent)
	}
}

// CleanOldFiles cleans old files
func CleanOldFiles(dr *resolver.DependencyResolver) error {
	if _, err := dr.Fs.Stat(dr.ResponseTargetFile); err == nil {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("unable to delete old response file", "response-target-file", dr.ResponseTargetFile)
			return err
		}
	}
	return nil
}

// ValidateMethod validates HTTP methods
func ValidateMethod(r *http.Request, allowedMethods []string) (string, error) {
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

// ProcessWorkflow processes workflow execution
func ProcessWorkflow(ctx context.Context, dr *resolver.DependencyResolver) error {
	dr.Context = ctx

	if err := dr.PrepareWorkflowDir(); err != nil {
		return err
	}

	if err := dr.PrepareImportFiles(); err != nil {
		return err
	}

	//nolint:contextcheck // context already passed via dr.Context
	if _, err := dr.HandleRunAction(); err != nil {
		return err
	}

	stdout, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		dr.Logger.Errorf("%s: %v", stdout, err)
		return err
	}

	dr.Logger.Debug("awaiting response...")

	// Wait for the response file to be ready
	if err := utils.WaitForFileReady(dr.Fs, dr.ResponseTargetFile, dr.Logger); err != nil {
		return err
	}

	return nil
}

// DecodeResponseContent decodes response content - simplified since DecodeBase64String was removed
func DecodeResponseContent(content []byte, logger *logging.Logger, pklEvaluator pkl.Evaluator, ctx context.Context) (*APIResponse, error) {
	var decodedResp APIResponse

	// Unmarshal JSON content into APIResponse struct
	err := json.Unmarshal(content, &decodedResp)
	if err != nil {
		logger.Error("failed to unmarshal response content", "error", err)
		return nil, err
	}

	// Since base64 decoding is no longer needed, just use the data as-is
	// No need to decode Base64 strings anymore

	return &decodedResp, nil
}

// Helper functions to replace the removed utils functions
func formatRequestParams(params map[string][]string) string {
	if len(params) == 0 {
		return "params{}"
	}
	var paramsLines []string
	for param, values := range params {
		for _, value := range values {
			paramsLines = append(paramsLines, fmt.Sprintf(`["%s"]="%s"`, param, value))
		}
	}
	return "params{" + strings.Join(paramsLines, ";") + "}"
}

func formatRequestHeaders(headers map[string][]string) string {
	if len(headers) == 0 {
		return "headers{}"
	}
	var headersLines []string
	for name, values := range headers {
		for _, value := range values {
			headersLines = append(headersLines, fmt.Sprintf(`["%s"]="%s"`, name, value))
		}
	}
	return "headers{" + strings.Join(headersLines, ";") + "}"
}

// FormatResponseJSON formats response as JSON
func FormatResponseJSON(content []byte) []byte {
	var response map[string]interface{}

	// Unmarshal the main response
	if err := json.Unmarshal(content, &response); err != nil {
		return content // Return original if JSON is invalid
	}

	// Navigate to response["response"]["data"]
	if responseField, ok := response["response"].(map[string]interface{}); ok {
		if data, ok := responseField["data"].([]interface{}); ok {
			for i, item := range data {
				if strItem, ok := item.(string); ok {
					// Try unmarshaling the string
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(strItem), &obj); err == nil {
						data[i] = obj // Replace with parsed object
					}
				}
			}
		}
	}

	// Marshal the updated JSON back to []byte with compact formatting
	formatted, err := json.Marshal(response)
	if err != nil {
		return content
	}

	return formatted
}
