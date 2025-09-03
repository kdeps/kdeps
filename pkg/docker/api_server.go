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
	"net/url"
	"os"
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

const (
	// UnknownActionID represents the default action ID when no specific action context is available
	UnknownActionID = "unknown"
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
	// Settings is a struct, not a pointer, so we can always access it

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

func setupRoutes(router *gin.Engine, ctx context.Context, wfAPIServerCORS apiserver.CORSConfig, wfTrustedProxies []string, routes []apiserver.APIServerRoutes, dr *resolver.DependencyResolver, semaphore chan struct{}) {
	for _, route := range routes {
		// APIServerRoutes is a struct, not a pointer, so we can always access it
		if route.Path == "" {
			dr.Logger.Error("route configuration is invalid", "route", route)
			continue
		}

		if wfAPIServerCORS.EnableCORS {
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

			// Always allow local development hosts in addition to configured origins
			localHosts := map[string]struct{}{
				"127.0.0.1": {},
				"0.0.0.0":   {},
				"::1":       {},
				"localhost": {},
			}

			router.Use(cors.New(cors.Config{
				AllowOrigins:     allowOrigins,
				AllowMethods:     allowMethods,
				AllowHeaders:     allowHeaders,
				ExposeHeaders:    exposeHeaders,
				AllowCredentials: wfAPIServerCORS.AllowCredentials,
				AllowOriginFunc: func(origin string) bool {
					parsed, err := url.Parse(origin)
					if err != nil {
						return false
					}
					host := parsed.Hostname()
					_, ok := localHosts[host]
					return ok
				},
				MaxAge: func() time.Duration {
					// MaxAge is a struct, not a pointer, so we can always access it
					return wfAPIServerCORS.MaxAge.GoDuration()
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

		handler := APIServerHandler(ctx, &route, dr, semaphore)
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

type requestHandler struct {
	ctx            context.Context
	route          *apiserver.APIServerRoutes
	baseDr         *resolver.DependencyResolver
	semaphore      chan struct{}
	allowedMethods []string
	c              *gin.Context

	graphID        string
	logger         *logging.Logger
	errorResponses []ErrorResponse
	dr             *resolver.DependencyResolver

	bodyData string
	fileMap  map[string]struct{ Filename, Filetype string }
}

func APIServerHandler(ctx context.Context, route *apiserver.APIServerRoutes, baseDr *resolver.DependencyResolver, semaphore chan struct{}) gin.HandlerFunc {
	if err := validateRouteConfig(route, baseDr.Logger); err != nil {
		return createInvalidRouteHandler()
	}

	allowedMethods := route.Methods

	return func(c *gin.Context) {
		handler := &requestHandler{
			ctx:            ctx,
			route:          route,
			baseDr:         baseDr,
			semaphore:      semaphore,
			allowedMethods: allowedMethods,
			c:              c,
		}
		handler.processRequest()
	}
}

func (h *requestHandler) processRequest() {
	h.initializeRequest()
	defer h.cleanup()

	if !h.acquireSemaphore() {
		return
	}
	defer h.releaseSemaphore()

	if !h.createResolver() {
		return
	}

	if !h.validateAndProcessRequest() {
		return
	}

	h.processAndSendResponse()
}

func (h *requestHandler) initializeRequest() {
	h.graphID = uuid.New().String()
	baseLogger := logging.GetLogger()
	h.logger = baseLogger.With("requestID", h.graphID)
	h.errorResponses = []ErrorResponse{}
	h.fileMap = make(map[string]struct{ Filename, Filetype string })
}

func (h *requestHandler) cleanup() {
	utils.ClearRequestErrors(h.graphID)
}

func (h *requestHandler) acquireSemaphore() bool {
	select {
	case h.semaphore <- struct{}{}:
		return true
	default:
		h.addUniqueError(http.StatusTooManyRequests, "Only one active connection is allowed", UnknownActionID)
		h.sendErrorResponse(http.StatusTooManyRequests)
		return false
	}
}

func (h *requestHandler) releaseSemaphore() {
	<-h.semaphore
}

func (h *requestHandler) createResolver() bool {
	newCtx := ktx.UpdateContext(h.ctx, ktx.CtxKeyGraphID, h.graphID)
	dr, err := resolver.NewGraphResolver(h.baseDr.Fs, newCtx, h.baseDr.Environment, h.c, h.logger)
	if err != nil {
		h.errorResponses = append(h.errorResponses, ErrorResponse{
			Code:     http.StatusInternalServerError,
			Message:  "Failed to initialize resolver",
			ActionID: UnknownActionID,
		})
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
	h.dr = dr
	return true
}

func (h *requestHandler) validateAndProcessRequest() bool {
	if !h.cleanOldFiles() {
		return false
	}

	if !h.validateMethod() {
		return false
	}

	if !h.handleSpecialMethods() {
		return false
	}

	if !h.processRequestBody() {
		return false
	}

	if !h.createRequestFile() {
		return false
	}

	if !h.processWorkflow() {
		return false
	}

	return true
}

func (h *requestHandler) cleanOldFiles() bool {
	if err := cleanOldFiles(h.dr); err != nil {
		h.addUniqueError(http.StatusInternalServerError, "Failed to clean old files", h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
	return true
}

func (h *requestHandler) validateMethod() bool {
	_, err := validateMethod(h.c.Request, h.allowedMethods)
	if err != nil {
		h.addUniqueError(http.StatusBadRequest, err.Error(), h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
	return true
}

func (h *requestHandler) handleSpecialMethods() bool {
	switch h.c.Request.Method {
	case http.MethodOptions:
		h.c.Header("Allow", "OPTIONS, GET, HEAD, POST, PUT, PATCH, DELETE")
		h.c.AbortWithStatus(http.StatusNoContent)
		return false
	case http.MethodHead:
		h.c.Header("Content-Type", "application/json")
		h.c.Status(http.StatusOK)
		h.c.Abort()
		return false
	}
	return true
}

func (h *requestHandler) processRequestBody() bool {
	switch h.c.Request.Method {
	case http.MethodGet:
		return h.processGetRequest()
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return h.processPostLikeRequest()
	case http.MethodDelete:
		h.bodyData = "Delete request received"
		return true
	default:
		h.addUniqueError(http.StatusMethodNotAllowed, "Unsupported method", h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
}

func (h *requestHandler) processGetRequest() bool {
	body, err := io.ReadAll(h.c.Request.Body)
	if err != nil {
		h.addUniqueError(http.StatusBadRequest, "Failed to read request body", h.getActionID())
		h.sendErrorResponse(http.StatusBadRequest)
		return false
	}
	defer h.c.Request.Body.Close()
	h.bodyData = string(body)
	return true
}

func (h *requestHandler) processPostLikeRequest() bool {
	contentType := h.c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return h.processMultipartForm()
	}

	// Read non-multipart body
	body, err := io.ReadAll(h.c.Request.Body)
	if err != nil {
		h.addUniqueError(http.StatusBadRequest, "Failed to read request body", h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
	defer h.c.Request.Body.Close()
	h.bodyData = string(body)
	return true
}

func (h *requestHandler) processMultipartForm() bool {
	if err := handleMultipartForm(h.c, h.dr, h.fileMap); err != nil {
		var he *handlerError
		if errors.As(err, &he) {
			h.addUniqueError(he.statusCode, he.message, h.getActionID())
			h.sendErrorResponse(he.statusCode)
		} else {
			h.addUniqueError(http.StatusInternalServerError, err.Error(), h.getActionID())
			h.sendErrorResponse(http.StatusInternalServerError)
		}
		return false
	}
	return true
}

func (h *requestHandler) processRequestData(bodyData string, fileMap map[string]struct{ Filename, Filetype string }) {
	// Store the processed data for use in createRequestFile
	// This is just a placeholder - the actual data is stored in the struct if needed
}

func (h *requestHandler) createRequestFile() bool {
	method, _ := validateMethod(h.c.Request, h.allowedMethods)

	urlSection := fmt.Sprintf(`Path = "%s"`, h.c.Request.URL.Path)
	clientIPSection := fmt.Sprintf(`IP = "%s"`, h.c.ClientIP())
	requestIDSection := fmt.Sprintf(`ID = "%s"`, h.graphID)
	dataSection := fmt.Sprintf(`Data = "%s"`, utils.EncodeBase64String(h.bodyData))

	var sb strings.Builder
	sb.WriteString("Files {\n")
	for _, fileInfo := range h.fileMap {
		fileBlock := fmt.Sprintf(`
	Filepath = "%s"
	Filetype = "%s"
`, fileInfo.Filename, fileInfo.Filetype)
		sb.WriteString(fmt.Sprintf("    [\"%s\"] {\n%s\n}\n", filepath.Base(fileInfo.Filename), fileBlock))
	}
	sb.WriteString("}\n")
	fileSection := sb.String()

	paramSection := utils.FormatRequestParams(h.c.Request.URL.Query())
	requestHeaderSection := utils.FormatRequestHeaders(h.c.Request.Header)

	sections := []string{urlSection, clientIPSection, requestIDSection, method, requestHeaderSection, dataSection, paramSection, fileSection}

	if err := evaluator.CreateAndProcessPklFile(
		h.dr.Fs,
		h.ctx,
		sections,
		h.dr.RequestPklFile,
		"APIServerRequest.pkl",
		nil,
		h.dr.Logger,
		evaluator.EvalPkl,
		true,
	); err != nil {
		h.addUniqueError(http.StatusInternalServerError, messages.ErrProcessRequestFile, h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
	return true
}

func (h *requestHandler) processWorkflow() bool {
	if err := processWorkflow(h.ctx, h.dr); err != nil {
		actionID := h.getActionID()
		h.addUniqueError(http.StatusInternalServerError, err.Error(), actionID)
		h.addUniqueError(http.StatusInternalServerError, messages.ErrEmptyResponse, actionID)
		h.sendErrorResponse(http.StatusInternalServerError)
		return false
	}
	return true
}

func (h *requestHandler) processAndSendResponse() {
	content, err := afero.ReadFile(h.dr.Fs, h.dr.ResponseTargetFile)
	if err != nil {
		h.addUniqueError(http.StatusInternalServerError, messages.ErrReadResponseFile, h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return
	}

	decodedResp, err := decodeResponseContent(content, h.dr.Logger)
	if err != nil {
		h.addUniqueError(http.StatusInternalServerError, messages.ErrDecodeResponseContent, h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return
	}

	h.mergeAccumulatedErrors()
	h.mergeAPIResponseErrors(decodedResp)

	if len(h.errorResponses) > 0 {
		h.addUniqueError(http.StatusInternalServerError, messages.ErrEmptyResponse, h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return
	}

	h.sendSuccessResponse(decodedResp)
}

func (h *requestHandler) mergeAccumulatedErrors() {
	allAccumulatedErrors := utils.GetRequestErrorsWithActionID(h.graphID)
	for _, accError := range allAccumulatedErrors {
		if accError != nil {
			actionID := accError.ActionID
			if actionID == "" {
				actionID = UnknownActionID
			}
			h.addUniqueError(accError.Code, accError.Message, actionID)
		}
	}
}

func (h *requestHandler) mergeAPIResponseErrors(decodedResp *APIResponse) {
	for _, apiError := range decodedResp.Errors {
		actionID := apiError.ActionID
		if actionID == "" {
			actionID = h.getActionID()
		}
		h.addUniqueError(apiError.Code, apiError.Message, actionID)
	}
}

func (h *requestHandler) sendSuccessResponse(decodedResp *APIResponse) {
	if decodedResp.Meta.Headers != nil {
		for key, value := range decodedResp.Meta.Headers {
			h.c.Header(key, value)
		}
	}

	decodedResp.Meta.RequestID = h.graphID
	decodedContent, err := json.Marshal(decodedResp)
	if err != nil {
		h.addUniqueError(http.StatusInternalServerError, messages.ErrMarshalResponseContent, h.getActionID())
		h.sendErrorResponse(http.StatusInternalServerError)
		return
	}

	decodedContent = formatResponseJSON(decodedContent)
	h.c.Data(http.StatusOK, "application/json; charset=utf-8", decodedContent)
}

func (h *requestHandler) getActionID() string {
	if h.dr != nil {
		if h.dr.CurrentResourceActionID != "" {
			return h.dr.CurrentResourceActionID
		}
		if h.dr.Workflow != nil {
			actionID := h.dr.Workflow.GetTargetActionID()
			if actionID != "" {
				return actionID
			}
		}
	}
	return UnknownActionID
}

func (h *requestHandler) addUniqueError(code int, message, actionID string) {
	if message == "" {
		return
	}

	if actionID == "" {
		actionID = UnknownActionID
	}

	for _, existingError := range h.errorResponses {
		if existingError.Message == message && existingError.Code == code && existingError.ActionID == actionID {
			return
		}
	}

	h.errorResponses = append(h.errorResponses, ErrorResponse{
		Code:     code,
		Message:  message,
		ActionID: actionID,
	})
}

func (h *requestHandler) sendErrorResponse(statusCode int) {
	response := APIResponse{
		Success: false,
		Response: ResponseData{
			Data: nil,
		},
		Meta: ResponseMeta{
			RequestID: h.graphID,
		},
		Errors: h.errorResponses,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		h.c.AbortWithStatusJSON(statusCode, response)
		return
	}

	h.c.Header("Content-Type", "application/json; charset=utf-8")
	h.c.AbortWithStatus(statusCode)
	if _, err := h.c.Writer.Write(jsonBytes); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write error response: %v\n", err)
	}
}

func validateRouteConfig(route *apiserver.APIServerRoutes, logger *logging.Logger) error {
	if route == nil || route.Path == "" || len(route.Methods) == 0 {
		logger.Error("invalid route configuration provided to APIServerHandler", "route", route)
		return errors.New("invalid route configuration")
	}
	return nil
}

func createInvalidRouteHandler() gin.HandlerFunc {
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
					ActionID: UnknownActionID,
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
		if _, err := c.Writer.Write(jsonBytes); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write error response: %v\n", err)
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
			return fmt.Sprintf(`Method = "%s"`, allowedMethod), nil
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
