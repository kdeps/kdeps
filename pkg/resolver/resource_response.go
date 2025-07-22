package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/template"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
)

// GoAPIResponse represents the JSON response structure using pure Go
type GoAPIResponse struct {
	Success  bool            `json:"success"`
	Response ResponseData    `json:"response"`
	Meta     ResponseMeta    `json:"meta"`
	Errors   []ErrorResponse `json:"errors,omitempty"`
}

// ResponseData represents the response data section
type ResponseData struct {
	Data interface{} `json:"data"`
}

// ResponseMeta represents the response metadata
type ResponseMeta struct {
	RequestID string `json:"requestID"`
}

// ErrorResponse represents an error in the response
type ErrorResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	ActionID string `json:"actionId,omitempty"`
}

// APIResponseTemplateData represents the data for the API response template
type APIResponseTemplateData struct {
	Header        string
	MemoryImport  string
	SessionImport string
	ToolImport    string
	ItemImport    string
	AgentImport   string
	Success       bool
	RequestID     string
	Headers       map[string]string
	Properties    map[string]string
	ResponseData  []string
	Errors        []ErrorTemplateData
}

// ErrorTemplateData represents error data for the template
type ErrorTemplateData struct {
	Code    int
	Message string
}

// CreateResponseGoJSON generates a JSON response using pure Go instead of PKL evaluation
//
// DEPRECATED: This method writes to files. Use BuildResponseInMemory() for memory-only approach.
// Policy: No temporary files - use memory-only approach for better performance.
func (dr *DependencyResolver) CreateResponseGoJSON(apiResponseBlock apiserverresponse.APIServerResponse) error {
	if dr == nil || len(dr.DBs) == 0 || dr.DBs[0] == nil {
		return errors.New("dependency resolver or database is nil")
	}

	// Ensure agent context is set directly in AgentReader (avoid env vars)
	if dr.Workflow != nil && dr.AgentReader != nil {
		dr.AgentReader.CurrentAgent = dr.Workflow.GetAgentID()
		dr.AgentReader.CurrentVersion = dr.Workflow.GetVersion()
	}

	if err := dr.DBs[0].PingContext(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	dr.Logger.Debug("starting CreateResponseGoJSON", "response", apiResponseBlock)

	// Get existing errors
	existingErrors := utils.GetRequestErrorsWithActionID(dr.RequestID)
	hasErrors := len(existingErrors) > 0

	// Prepare response data
	var responseData interface{}
	if hasErrors {
		responseData = nil
	} else {
		// Extract data from the response block
		if apiResponseBlock.GetResponse() != nil && apiResponseBlock.GetResponse().Data != nil {
			responseData = dr.extractResponseData(apiResponseBlock.GetResponse().Data)
		}
	}

	// Build the Go response structure
	response := GoAPIResponse{
		Success: !hasErrors,
		Response: ResponseData{
			Data: responseData,
		},
		Meta: ResponseMeta{
			RequestID: dr.RequestID,
		},
	}

	// Add errors if any
	if hasErrors {
		response.Errors = make([]ErrorResponse, len(existingErrors))
		for i, err := range existingErrors {
			response.Errors[i] = ErrorResponse{
				Code:     err.Code,
				Message:  err.Message,
				ActionID: err.ActionID, // This comes from utils.GetRequestErrors, not the PKL structure
			}
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON response: %w", err)
	}

	// Write the JSON response to the target file
	if err := afero.WriteFile(dr.Fs, dr.ResponseTargetFile, jsonData, pkg.DefaultOctalFilePerms); err != nil {
		return fmt.Errorf("failed to write JSON response file: %w", err)
	}

	dr.Logger.Debug("CreateResponseGoJSON completed", "file", dr.ResponseTargetFile)
	return nil
}

// BuildResponseInMemory generates a JSON response in memory without writing to file
// Returns the JSON string directly for immediate use
func (dr *DependencyResolver) BuildResponseInMemory(apiResponseBlock apiserverresponse.APIServerResponse) (string, error) {
	if dr == nil || len(dr.DBs) == 0 || dr.DBs[0] == nil {
		return "", errors.New("dependency resolver or database is nil")
	}

	// Ensure agent context is set directly in AgentReader (avoid env vars)
	if dr.Workflow != nil && dr.AgentReader != nil {
		dr.AgentReader.CurrentAgent = dr.Workflow.GetAgentID()
		dr.AgentReader.CurrentVersion = dr.Workflow.GetVersion()
	}

	if err := dr.DBs[0].PingContext(context.Background()); err != nil {
		return "", fmt.Errorf("failed to ping database: %w", err)
	}

	dr.Logger.Debug("building response in memory", "response", apiResponseBlock)

	// Get existing errors
	existingErrors := utils.GetRequestErrorsWithActionID(dr.RequestID)
	hasErrors := len(existingErrors) > 0

	// Prepare response data
	var responseData interface{}
	if hasErrors {
		responseData = nil
	} else {
		// Extract data from the response block
		if apiResponseBlock.GetResponse() != nil && apiResponseBlock.GetResponse().Data != nil {
			responseData = dr.extractResponseData(apiResponseBlock.GetResponse().Data)
		}
	}

	// Build the Go response structure
	response := GoAPIResponse{
		Success: !hasErrors,
		Response: ResponseData{
			Data: responseData,
		},
		Meta: ResponseMeta{
			RequestID: dr.RequestID,
		},
	}

	// Add errors if any
	if hasErrors {
		response.Errors = make([]ErrorResponse, len(existingErrors))
		for i, err := range existingErrors {
			response.Errors[i] = ErrorResponse{
				Code:     err.Code,
				Message:  err.Message,
				ActionID: err.ActionID,
			}
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON response: %w", err)
	}

	dr.Logger.Debug("BuildResponseInMemory completed", "jsonLength", len(jsonData))
	return string(jsonData), nil
}

// extractResponseData extracts data from the response block without PKL evaluation
func (dr *DependencyResolver) extractResponseData(dataList []any) interface{} {
	if len(dataList) == 0 {
		return dr.createFallbackResponseData()
	}

	// Process each data item
	var processedData []interface{}
	for _, item := range dataList {
		if item == nil {
			continue
		}

		// Convert the item to a map for processing
		if itemMap, ok := item.(map[string]interface{}); ok {
			processedItem := dr.processDataItem(itemMap)
			if processedItem != nil {
				processedData = append(processedData, processedItem)
			}
		} else {
			// If it's not a map, try to process it as is
			processedData = append(processedData, item)
		}
	}

	return processedData
}

// processDataItem processes a single data item without PKL evaluation
func (dr *DependencyResolver) processDataItem(item map[string]interface{}) interface{} {
	// Try to extract the actual data from the item
	if value, exists := item["value"]; exists {
		return dr.processValue(value)
	}

	// If no value field, process the item as-is
	return dr.processValue(item)
}

// processValue processes a value, handling base64 decoding if needed
func (dr *DependencyResolver) processValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Handle string values (check for base64 encoding)
	if str, ok := value.(string); ok {
		// Use string directly without base64 decoding
		// Try to parse as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(str), &jsonData); err == nil {
			return jsonData
		}
		return str
	}

	// Handle maps recursively
	if mapValue, ok := value.(map[string]interface{}); ok {
		processedMap := make(map[string]interface{})
		for k, v := range mapValue {
			processedMap[k] = dr.processValue(v)
		}
		return processedMap
	}

	// Handle slices recursively
	if sliceValue, ok := value.([]interface{}); ok {
		processedSlice := make([]interface{}, len(sliceValue))
		for i, v := range sliceValue {
			processedSlice[i] = dr.processValue(v)
		}
		return processedSlice
	}

	// Return other types as-is
	return value
}

// createFallbackResponseData creates meaningful fallback data when the original response is empty
func (dr *DependencyResolver) createFallbackResponseData() interface{} {
	if dr.Request == nil {
		return map[string]interface{}{
			"status":  "fallback_response",
			"message": "API is working but no query parameters provided",
		}
	}

	query := dr.Request.Query("q")
	if query == "" {
		return map[string]interface{}{
			"status":  "fallback_response",
			"message": "API is working but no query parameter 'q' provided",
		}
	}

	// Create a meaningful fallback response with the query context
	fallbackResponse := map[string]interface{}{
		"query":      query,
		"status":     "fallback_response",
		"message":    fmt.Sprintf("Information about %s: This is a fallback response. The LLM resource is not properly configured or the model is not available. Please check the workflow configuration and ensure the LLM resource files are present.", query),
		"timestamp":  time.Now().Format(time.RFC3339),
		"request_id": dr.RequestID,
	}

	if dr.Workflow != nil {
		fallbackResponse["workflow_id"] = dr.Workflow.GetAgentID()
	}

	return fallbackResponse
}

// CreateResponsePklFile generates a PKL file from the API response and processes it.
//
// DEPRECATED: This method creates temporary files. Use BuildResponseInMemory() instead.
// Policy: No temporary files - use memory-only approach for better performance.
func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock apiserverresponse.APIServerResponse) error {
	if dr == nil || len(dr.DBs) == 0 || dr.DBs[0] == nil {
		return errors.New("dependency resolver or database is nil")
	}

	// Ensure agent context is set directly in AgentReader (avoid env vars)
	if dr.Workflow != nil && dr.AgentReader != nil {
		dr.AgentReader.CurrentAgent = dr.Workflow.GetAgentID()
		dr.AgentReader.CurrentVersion = dr.Workflow.GetVersion()
	}

	if err := dr.DBs[0].PingContext(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	dr.Logger.Debug("starting CreateResponsePklFile", "response", apiResponseBlock)

	// Always allow processing - the buildResponseSections will handle error merging
	// This ensures all workflow errors are preserved regardless of the response resource content
	dr.Logger.Debug("processing response file with comprehensive error merging", "requestID", dr.RequestID)

	if err := dr.EnsureResponsePklFileNotExists(); err != nil {
		return fmt.Errorf("ensure response PKL file does not exist: %w", err)
	}

	sections := dr.buildResponseSections(dr.RequestID, apiResponseBlock)

	// Create a wrapper function that matches the expected signature
	evalFunc := func(evaluatorObj pkl.Evaluator, fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		return evaluator.EvalPkl(evaluatorObj, fs, ctx, tmpFile, headerSection, nil, logger)
	}

	// Use the evaluator directly
	pklEvaluator := dr.Evaluator

	if err := evaluator.CreateAndProcessPklFile(pklEvaluator, dr.Fs, dr.Context, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evalFunc, false); err != nil {
		return fmt.Errorf("create/process PKL file: %w", err)
	}

	dr.Logger.Debug("successfully created and processed PKL file", "file", dr.ResponsePklFile)

	// Now evaluate the PKL response file to generate the JSON response
	jsonResponse, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL response file: %w", err)
	}

	// Write the JSON response to the target file
	if err := afero.WriteFile(dr.Fs, dr.ResponseTargetFile, []byte(jsonResponse), pkg.DefaultOctalFilePerms); err != nil {
		return fmt.Errorf("failed to write JSON response to target file: %w", err)
	}

	dr.Logger.Debug("successfully generated JSON response from PKL", "file", dr.ResponseTargetFile, "contentLength", len(jsonResponse))
	return nil
}

// EnsureResponsePklFileNotExists ensures the response PKL file does not exist
func (dr *DependencyResolver) EnsureResponsePklFileNotExists() error {
	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		return fmt.Errorf("check file existence: %w", err)
	}

	if exists {
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			return fmt.Errorf("delete old response file: %w", err)
		}
		dr.Logger.Debug("old response PKL file deleted", "file", dr.ResponsePklFile)
	}
	return nil
}

func (dr *DependencyResolver) buildResponseSections(requestID string, apiResponseBlock apiserverresponse.APIServerResponse) []string {
	// Get new errors from the current response resource only
	var responseErrors []*apiserverresponse.APIServerErrorsBlock
	if apiResponseBlock.GetErrors() != nil {
		responseErrors = *apiResponseBlock.GetErrors()
	}

	// Only use response-specific errors in the PKL file
	// Workflow errors will be merged separately at the API server level

	// If there are any response-specific errors, mark as failure
	successPtr := apiResponseBlock.GetSuccess()
	isSuccess := successPtr != nil && *successPtr && len(responseErrors) == 0

	// Check if response data is empty and create fallback data
	responseData := apiResponseBlock.GetResponse()
	dr.Logger.Debug("buildResponseSections: checking response data",
		"requestID", requestID,
		"responseData_nil", responseData == nil,
		"responseData_data_nil", func() bool {
			if responseData != nil {
				return responseData.Data == nil
			}
			return true
		}(),
		"responseData_length", func() int {
			if responseData != nil && responseData.Data != nil {
				return len(responseData.Data)
			}
			return 0
		}())

	if responseData == nil || responseData.Data == nil || len(responseData.Data) == 0 {
		// Create fallback response data with meaningful information
		dr.Logger.Info("Response data is empty, creating fallback", "requestID", requestID)
		fallbackData := createFallbackResponseData(dr)
		if fallbackData != "" {
			dr.Logger.Info("Using fallback response data", "requestID", requestID, "fallbackLength", len(fallbackData))
			responseData = &apiserverresponse.APIServerResponseBlock{
				Data: []interface{}{fallbackData},
			}
		} else {
			// If fallback fails, create a basic response
			dr.Logger.Info("Fallback data is empty, creating basic response", "requestID", requestID)
			responseData = &apiserverresponse.APIServerResponseBlock{
				Data: []interface{}{`{"status": "processing", "message": "Request processed successfully but no data available", "requestID": "` + requestID + `"}`},
			}
		}
	} else {
		dr.Logger.Info("Response data is not empty, using original", "requestID", requestID, "dataLength", len(responseData.Data))
	}

	// Prepare template data
	templateData := APIResponseTemplateData{
		Header:        "",
		MemoryImport:  schema.ImportPath(dr.Context, "Memory.pkl"),
		SessionImport: schema.ImportPath(dr.Context, "Session.pkl"),
		ToolImport:    schema.ImportPath(dr.Context, "Tool.pkl"),
		ItemImport:    schema.ImportPath(dr.Context, "Item.pkl"),
		AgentImport:   schema.ImportPath(dr.Context, "Agent.pkl"),
		Success:       isSuccess,
		RequestID:     requestID,
		Headers:       make(map[string]string),
		Properties:    make(map[string]string),
		ResponseData:  []string{},
		Errors:        []ErrorTemplateData{},
	}

	// Add headers if present
	if meta := apiResponseBlock.GetMeta(); meta != nil {
		if meta.Headers != nil {
			templateData.Headers = *meta.Headers
		}
		if meta.Properties != nil {
			templateData.Properties = *meta.Properties
		}
	}

	// Add response data
	if responseData != nil && responseData.Data != nil {
		for _, data := range responseData.Data {
			templateData.ResponseData = append(templateData.ResponseData, formatDataValue(data))
		}
	}

	// Add errors
	for _, err := range responseErrors {
		if err != nil {
			templateData.Errors = append(templateData.Errors, ErrorTemplateData{
				Code:    err.Code,
				Message: err.Message,
			})
		}
	}

	// Convert template data to map[string]string for LoadTemplate
	templateMap := map[string]string{
		"Header":        templateData.Header,
		"MemoryImport":  templateData.MemoryImport,
		"SessionImport": templateData.SessionImport,
		"ToolImport":    templateData.ToolImport,
		"ItemImport":    templateData.ItemImport,
		"AgentImport":   templateData.AgentImport,
		"Success":       fmt.Sprintf("%v", templateData.Success),
		"RequestID":     templateData.RequestID,
	}

	// Add headers as string
	if len(templateData.Headers) > 0 {
		headersList := make([]string, 0, len(templateData.Headers))
		for k, v := range templateData.Headers {
			headersList = append(headersList, fmt.Sprintf(`["%s"] = "%s"`, k, v))
		}
		templateMap["Headers"] = strings.Join(headersList, "\n        ")
	}

	// Add properties as string
	if len(templateData.Properties) > 0 {
		propsList := make([]string, 0, len(templateData.Properties))
		for k, v := range templateData.Properties {
			propsList = append(propsList, fmt.Sprintf(`["%s"] = "%s"`, k, v))
		}
		templateMap["Properties"] = strings.Join(propsList, "\n        ")
	}

	// Add response data
	templateMap["ResponseData"] = strings.Join(templateData.ResponseData, "\n        ")

	// Add errors as string
	if len(templateData.Errors) > 0 {
		errorsList := make([]string, 0, len(templateData.Errors))
		for _, err := range templateData.Errors {
			errorsList = append(errorsList, fmt.Sprintf(`new {
        Code = %d
        Message = #"""
%s
"""#
    }`, err.Code, err.Message))
		}
		templateMap["Errors"] = strings.Join(errorsList, "\n    ")
	}

	// Generate PKL content using template
	content, err := template.LoadTemplate("api_response.pkl", templateMap)
	if err != nil {
		dr.Logger.Error("failed to render API response template", "error", err)
		// Fallback to single section with error
		return []string{fmt.Sprintf("Success = false\nErrors { new { Code = 500; Message = \"Template rendering failed: %s\" } }", err.Error())}
	}

	// Return the generated content as a single section
	return []string{content}
}

func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	responseData := make([]string, 0, len(response.Data))
	for _, v := range response.Data {
		// Skip empty or null values to avoid empty data entries
		if v == nil || v == "" {
			continue
		}
		responseData = append(responseData, formatDataValue(v))
	}

	if len(responseData) == 0 {
		return ""
	}

	return fmt.Sprintf(`
Response {
  Data {
%s
  }
}`, strings.Join(responseData, "\n    "))
}

func formatResponseMeta(requestID string, meta *apiserverresponse.APIServerResponseMetaBlock) string {
	if meta == nil || *meta.Headers == nil && *meta.Properties == nil {
		return fmt.Sprintf(`
Meta {
  RequestID = "%s"
}
`, requestID)
	}

	responseMetaHeaders := utils.FormatResponseHeaders(*meta.Headers)
	responseMetaProperties := utils.FormatResponseProperties(*meta.Properties)

	if len(responseMetaHeaders) == 0 && len(responseMetaProperties) == 0 {
		return fmt.Sprintf(`
Meta {
  RequestID = "%s"
}
`, requestID)
	}

	return fmt.Sprintf(`
Meta {
  RequestID = "%s"
  %s
  %s
}`, requestID, responseMetaHeaders, responseMetaProperties)
}

func formatMap(m map[interface{}]interface{}) string {
	mappingParts := []string{"new Mapping {"}
	for k, v := range m {
		keyStr := strings.ReplaceAll(fmt.Sprintf("%v", k), `"`, `\"`)
		valueStr := formatValue(v)
		mappingParts = append(mappingParts, fmt.Sprintf(`    ["%s"] = %s`, keyStr, valueStr))
	}
	mappingParts = append(mappingParts, "}")
	return strings.Join(mappingParts, "\n")
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		m := make(map[interface{}]interface{})
		for key, val := range v {
			m[key] = val
		}
		return formatMap(m)
	case map[interface{}]interface{}:
		return formatMap(v)
	case nil:
		return "null"
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			return formatValue(rv.Elem().Interface())
		}
		if rv.Kind() == reflect.Struct {
			return formatMap(structToMap(rv.Interface()))
		}
		return fmt.Sprintf(`
"""
%v
"""
`, v)
	}
}

func structToMap(s interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for i := range val.NumField() {
		fieldName := val.Type().Field(i).Name
		fieldValue := val.Field(i).Interface()
		result[fieldName] = fieldValue
	}
	return result
}

// This is the 'data' array in the API response
func formatDataValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// For strings, return the value directly as a quoted string
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
	case map[string]interface{}, map[interface{}]interface{}:
		// For maps, use the existing formatValue logic
		return formatValue(v)
	case nil:
		return "null"
	default:
		// For other types, convert to JSON and return directly
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			// Fallback to formatValue if JSON marshaling fails
			return formatValue(v)
		}
		// Return the JSON string directly
		return string(jsonBytes)
	}
}

func formatErrors(errors *[]*apiserverresponse.APIServerErrorsBlock, logger *logging.Logger) string {
	if errors == nil || len(*errors) == 0 {
		return ""
	}

	var newBlocks string
	for _, err := range *errors {
		if err != nil {
			newBlocks += fmt.Sprintf(`
  new {
    Code = %d
    Message = #"""
%s
"""#
  }`, err.Code, err.Message)
		}
	}

	if newBlocks != "" {
		return fmt.Sprintf(`Errors {%s
}`, newBlocks)
	}
	return ""
}

// createFallbackResponseData creates meaningful fallback data when the original response is empty
func createFallbackResponseData(dr *DependencyResolver) string {
	if dr.Request == nil {
		return "API is working but no query parameters provided"
	}

	query := dr.Request.Query("q")
	if query == "" {
		return "API is working but no query parameter 'q' provided"
	}

	// Create a meaningful fallback response with the query context
	fallbackResponse := map[string]interface{}{
		"query":      query,
		"status":     "fallback_response",
		"message":    fmt.Sprintf("Information about %s: This is a fallback response. The LLM resource is not properly configured or the model is not available. Please check the workflow configuration and ensure the LLM resource files are present.", query),
		"timestamp":  time.Now().Format(time.RFC3339),
		"request_id": dr.RequestID,
		"workflow_id": func() string {
			if dr.Workflow != nil {
				return dr.Workflow.GetTargetActionID()
			}
			return "unknown"
		}(),
	}

	// Convert to JSON string
	jsonData, err := json.Marshal(fallbackResponse)
	if err != nil {
		dr.Logger.Error("Failed to marshal fallback response", "error", err)
		return fmt.Sprintf("Fallback response for query '%s': LLM resource not configured", query)
	}

	return string(jsonData)
}

// EvalPklFormattedResponseFile evaluates a PKL file and returns the JSON output.
func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		return "", fmt.Errorf("check PKL file existence: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("PKL file does not exist: %s", dr.ResponsePklFile)
	}

	if err := dr.validatePklFileExtension(); err != nil {
		return "", err
	}

	if err := dr.ensureResponseTargetFileNotExists(); err != nil {
		return "", fmt.Errorf("ensure target file not exists: %w", err)
	}

	// --- BEGIN PATCH: Enhanced LLM resource handling with fallback ---
	// Note: The database issue has been fixed, so the PKL function llm.response() should work correctly
	// No need to prepend LLM content or use fallback logic
	// --- END PATCH ---

	dr.Logger.Debug("using configured pklres reader for LLM resource access")

	// Get the singleton evaluator
	pklEvaluator, err := evaluator.GetEvaluator()
	if err != nil {
		return "", fmt.Errorf("get PKL evaluator: %w", err)
	}

	// Create module source
	moduleSource := pkl.FileSource(dr.ResponsePklFile)

	// Evaluate the PKL file to get text format first
	pklText, err := pklEvaluator.EvaluateOutputText(dr.Context, moduleSource)
	if err != nil {
		return "", fmt.Errorf("evaluate PKL file: %w", err)
	}

	// Parse the PKL text and convert to proper JSON format
	result, err := dr.convertPklResponseToJSON(pklText)
	if err != nil {
		return "", fmt.Errorf("convert PKL to JSON: %w", err)
	}

	// Write result to target file
	if err := afero.WriteFile(dr.Fs, dr.ResponseTargetFile, []byte(result), pkg.DefaultOctalFilePerms); err != nil {
		return "", fmt.Errorf("write result to target file: %w", err)
	}

	return result, nil
}

func (dr *DependencyResolver) convertPklResponseToJSON(pklText string) (string, error) {
	// Parse the PKL response structure and convert to JSON
	// Expected PKL format:
	// Success = true
	// Meta { RequestID = "..." Headers {} Properties {} }
	// Response { Data { ... } }
	// Errors = null

	lines := strings.Split(strings.TrimSpace(pklText), "\n")

	// Simple parser to extract the Data section
	inDataSection := false
	dataContent := strings.Builder{}
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "Data {") {
			inDataSection = true
			braceCount = 1
			continue
		}

		if inDataSection {
			// Count braces to know when we're done with the Data section
			braceCount += strings.Count(trimmed, "{")
			braceCount -= strings.Count(trimmed, "}")

			if braceCount > 0 {
				// We're still inside the Data section, collect the content
				if trimmed != "" && !strings.Contains(trimmed, "Data {") {
					if dataContent.Len() > 0 {
						dataContent.WriteString(" ")
					}
					dataContent.WriteString(trimmed)
				}
			} else {
				// We've closed the Data section
				break
			}
		}
	}

	// Process the data content
	dataStr := strings.TrimSpace(dataContent.String())

	// Strip trailing non-JSON characters (e.g., '%')
	for len(dataStr) > 0 && (dataStr[len(dataStr)-1] < ' ' || dataStr[len(dataStr)-1] > '~') {
		dataStr = dataStr[:len(dataStr)-1]
	}
	// Also strip trailing '%' character specifically
	if strings.HasSuffix(dataStr, "%") {
		dataStr = strings.TrimSuffix(dataStr, "%")
	}
	// Additional cleanup for any remaining non-JSON characters
	dataStr = strings.TrimSpace(dataStr)

	// Log the cleaned data for debugging
	dr.Logger.Debug("Processing cleaned data", "original_length", len(dataContent.String()), "cleaned_length", len(dataStr), "cleaned_data", dataStr)

	// If the data is empty after cleaning, return a default response
	if dataStr == "" || dataStr == "{}" {
		defaultData := dr.getDefaultResponseData("No data available")
		if defaultJSON, err := json.Marshal(defaultData); err == nil {
			return string(defaultJSON), nil
		}
		return "{}", nil
	}
	if dataStr != "" && dataStr != "{}" {
		// If the data contains a JSON string, parse it
		if strings.HasPrefix(dataStr, "\"") && strings.HasSuffix(dataStr, "\"") {
			// Remove quotes and parse as JSON
			jsonStr := strings.Trim(dataStr, "\"")
			// Unescape any escaped quotes
			jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

			// Try to parse as JSON
			var jsonData interface{}
			if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
				// Valid JSON, return as is
				return jsonStr, nil
			}

			// If JSON parsing fails, return raw text wrapped in data field
			dr.Logger.Debug("JSON parsing failed, returning raw text", "raw", jsonStr)
			fallbackData := map[string]interface{}{"raw_text": jsonStr}
			if structuredJSON, err := json.Marshal(fallbackData); err == nil {
				return string(structuredJSON), nil
			}
		} else {
			// Try to parse directly as JSON
			var jsonData interface{}
			if err := json.Unmarshal([]byte(dataStr), &jsonData); err == nil {
				// Check if it's an object with a single empty string key
				if obj, ok := jsonData.(map[string]interface{}); ok {
					if len(obj) == 1 {
						for key, value := range obj {
							if key == "" && value != nil {
								// Use the value as the response
								if valueObj, ok := value.(map[string]interface{}); ok {
									if valueJSON, err := json.Marshal(valueObj); err == nil {
										return string(valueJSON), nil
									}
								}
							}
						}
					}
				}
				// Valid JSON, return as is
				return dataStr, nil
			}

			// If JSON parsing fails, return raw text wrapped in data field
			dr.Logger.Debug("JSON parsing failed, returning raw text", "raw", dataStr)
			fallbackData := map[string]interface{}{"raw_text": dataStr}
			if structuredJSON, err := json.Marshal(fallbackData); err == nil {
				return string(structuredJSON), nil
			}
		}
	}

	// If all else fails, return a default structure
	defaultData := dr.getDefaultResponseData("Unable to parse LLM response")
	defaultData["raw_response"] = dataStr

	if defaultJSON, err := json.Marshal(defaultData); err == nil {
		return string(defaultJSON), nil
	}

	return "{}", nil
}

// getDefaultResponseData retrieves the configured JSONResponseKeys from the current resource
// and creates a default response structure with those keys set to nil
func (dr *DependencyResolver) getDefaultResponseData(errorMessage string) map[string]interface{} {
	defaultData := map[string]interface{}{
		"error": errorMessage,
	}

	// Try to get JSONResponseKeys from the current resource
	if dr.CurrentResourceActionID != "" && dr.PklresHelper != nil {
		jsonResponseKeysStr, err := dr.PklresHelper.Get(dr.CurrentResourceActionID, "jsonResponseKeys")
		if err == nil && jsonResponseKeysStr != "" {
			var jsonResponseKeys []string
			if err := json.Unmarshal([]byte(jsonResponseKeysStr), &jsonResponseKeys); err == nil {
				// Add each configured key with nil value
				for _, key := range jsonResponseKeys {
					defaultData[key] = nil
				}
				return defaultData
			}
		}
	}

	// Fallback to empty structure if no keys are configured
	return defaultData
}

func (dr *DependencyResolver) validatePklFileExtension() error {
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		return errors.New("file must have .pkl extension")
	}
	return nil
}

func (dr *DependencyResolver) ensureResponseTargetFileNotExists() error {
	exists, err := afero.Exists(dr.Fs, dr.ResponseTargetFile)
	if err != nil {
		return fmt.Errorf("check target file existence: %w", err)
	}

	if exists {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			return fmt.Errorf("remove target file: %w", err)
		}
	}
	return nil
}

// HandleAPIErrorResponse creates an error response PKL file when in API server mode,
// or returns an actual error when not in API server mode.
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string, fatal bool) (bool, error) {
	if dr.APIServerMode {
		// Get the current actionID for error context
		actionID := "unknown"
		if dr.CurrentResourceActionID != "" {
			actionID = dr.CurrentResourceActionID
		} else if dr.Workflow != nil {
			workflowActionID := dr.Workflow.GetTargetActionID()
			if workflowActionID != "" {
				actionID = workflowActionID
			}
		}

		// Always accumulate the error in the global error collection with actionID
		errorResponse := utils.NewAPIServerResponseWithActionID(false, nil, code, message, dr.RequestID, actionID)

		// For fail-fast scenarios, we need to create a comprehensive error response
		// that includes all accumulated errors, not just the current one
		if fatal {
			// Get all accumulated errors and merge with the current error
			currentErrors := []*apiserverresponse.APIServerErrorsBlock{
				{Code: code, Message: message},
			}
			allErrors := utils.MergeAllErrors(dr.RequestID, currentErrors)

			// Create a comprehensive error response with all accumulated errors
			successFalse := false
			finalErrorResponse := &apiserverresponse.APIServerResponseImpl{
				Success:  &successFalse,
				Response: &apiserverresponse.APIServerResponseBlock{Data: nil},
				Errors:   &allErrors,
			}

			if err := dr.CreateResponseGoJSON(finalErrorResponse); err != nil {
				return fatal, fmt.Errorf("create comprehensive error response: %w", err)
			}
			return fatal, fmt.Errorf("%s", message)
		}

		// For non-fatal errors, just accumulate and continue
		if err := dr.CreateResponseGoJSON(errorResponse); err != nil {
			return fatal, fmt.Errorf("create error response: %w", err)
		}
		return fatal, nil
	}

	// When not in API server mode, return an actual error to fail the processing
	return fatal, fmt.Errorf("validation failed (code %d): %s", code, message)
}

// Exported for testing
var FormatResponseMeta = formatResponseMeta

// StructToMap converts a struct to a map using reflection
func StructToMap(s interface{}) map[interface{}]interface{} {
	return structToMap(s)
}

// Exported for testing
func (dr *DependencyResolver) BuildResponseSections() ([]string, error) {
	// For testing, we need to provide default values
	requestID := dr.RequestID
	if requestID == "" {
		requestID = "test-request-id"
	}

	// Create a default API response for testing
	success := true
	response := &apiserverresponse.APIServerResponseImpl{
		Success: &success,
		Response: &apiserverresponse.APIServerResponseBlock{
			Data: []interface{}{"test data"},
		},
	}

	return dr.buildResponseSections(requestID, response), nil
}

// BuildResponseSectionsForTesting is exported for testing with custom responses
func (dr *DependencyResolver) BuildResponseSectionsForTesting(response apiserverresponse.APIServerResponse) ([]string, error) {
	requestID := dr.RequestID
	if requestID == "" {
		requestID = "request-id"
	}

	return dr.buildResponseSections(requestID, response), nil
}

// ValidatePklFileExtension is exported for testing
func (dr *DependencyResolver) ValidatePklFileExtension() error {
	return dr.validatePklFileExtension()
}

// EnsureResponseTargetFileNotExists is exported for testing
func (dr *DependencyResolver) EnsureResponseTargetFileNotExists() error {
	return dr.ensureResponseTargetFileNotExists()
}
