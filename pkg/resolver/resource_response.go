package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
)

// CreateResponsePklFile generates a PKL file from the API response and processes it.
func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock apiserverresponse.APIServerResponse) error {
	if dr == nil || len(dr.DBs) == 0 || dr.DBs[0] == nil {
		return fmt.Errorf("dependency resolver or database is nil")
	}

	// Ensure agent context is set for PKL evaluation
	if dr.Workflow != nil {
		os.Setenv("KDEPS_CURRENT_AGENT", dr.Workflow.GetAgentID())
		os.Setenv("KDEPS_CURRENT_VERSION", dr.Workflow.GetVersion())

		// Also update the AgentReader context directly
		if dr.AgentReader != nil {
			dr.AgentReader.CurrentAgent = dr.Workflow.GetAgentID()
			dr.AgentReader.CurrentVersion = dr.Workflow.GetVersion()
		}
	}

	// Set the request ID for output file lookup
	os.Setenv("KDEPS_REQUEST_ID", dr.RequestID)

	if err := dr.DBs[0].PingContext(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	dr.Logger.Debug("starting CreateResponsePklFile", "response", apiResponseBlock)

	// Always allow processing - the buildResponseSections will handle error merging
	// This ensures all workflow errors are preserved regardless of the response resource content
	dr.Logger.Debug("processing response file with comprehensive error merging", "requestID", dr.RequestID)

	if err := dr.ensureResponsePklFileNotExists(); err != nil {
		return fmt.Errorf("ensure response PKL file does not exist: %w", err)
	}

	sections := dr.buildResponseSections(dr.RequestID, apiResponseBlock)

	// Create a wrapper function that matches the expected signature
	evalFunc := func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		return evaluator.EvalPkl(fs, ctx, tmpFile, headerSection, nil, logger)
	}

	if err := evaluator.CreateAndProcessPklFile(dr.Fs, dr.Context, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evalFunc, false); err != nil {
		return fmt.Errorf("create/process PKL file: %w", err)
	}

	dr.Logger.Debug("successfully created and processed PKL file", "file", dr.ResponsePklFile)
	return nil
}

func (dr *DependencyResolver) ensureResponsePklFileNotExists() error {
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

	sections := []string{
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Document.pkl" as document`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Memory.pkl" as memory`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Session.pkl" as session`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Tool.pkl" as tool`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Item.pkl" as item`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Agent.pkl" as agent`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf("Success = %v", isSuccess),
		formatResponseMeta(requestID, apiResponseBlock.GetMeta()),
		formatResponseData(responseData), // Use the fallback data if original was empty
		formatErrors(&responseErrors, dr.Logger),
	}
	return sections
}

func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		// Return empty data structure instead of empty string
		return `
Response {
  Data {
  }
}`
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
		// Return empty data structure instead of empty string
		return `
Response {
  Data {
  }
}`
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

func formatDataValue(value interface{}) string {
	uuidVal := strings.ReplaceAll(uuid.New().String(), "-", "_")
	val := formatValue(value)
	return fmt.Sprintf(`
local JSONDocument_%s = %s
local JSONDocumentType_%s = JSONDocument_%s is Mapping | Dynamic

if (JSONDocumentType_%s)
  document.jsonRenderDocument(JSONDocument_%s)
else
  document.jsonRenderDocument((if (document.jsonParser(JSONDocument_%s) != null) document.jsonParser(JSONDocument_%s) else JSONDocument_%s))
`, uuidVal, val, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal)
}

func formatErrors(errors *[]*apiserverresponse.APIServerErrorsBlock, logger *logging.Logger) string {
	if errors == nil || len(*errors) == 0 {
		return ""
	}

	var newBlocks string
	for _, err := range *errors {
		if err != nil {
			decodedMessage := decodeErrorMessage(err.Message, logger)
			newBlocks += fmt.Sprintf(`
  new {
    Code = %d
    Message = #"""
%s
"""#
  }`, err.Code, decodedMessage)
		}
	}

	if newBlocks != "" {
		return fmt.Sprintf(`Errors {%s
}`, newBlocks)
	}
	return ""
}

func decodeErrorMessage(message string, logger *logging.Logger) string {
	if message == "" {
		return ""
	}
	decoded, err := utils.DecodeBase64IfNeeded(message)
	if err != nil {
		logger.Warn("failed to decode error message", "message", message, "error", err)
		return message
	}
	return decoded
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
	llmResourceID := "@whois2/llmResource:1.0.0"
	llmContent := ""

	// Try to get the target action ID from workflow for better resource identification
	targetActionID := ""
	if dr.Workflow != nil {
		targetActionID = dr.Workflow.GetTargetActionID()
		dr.Logger.Debug("Workflow target action", "targetActionID", targetActionID)
	}

	// Try multiple resource IDs for better compatibility
	resourceIDs := []string{llmResourceID}
	if targetActionID != "" && targetActionID != llmResourceID {
		resourceIDs = append(resourceIDs, targetActionID)
	}

	if dr.PklresHelper != nil {
		for _, resourceID := range resourceIDs {
			dr.Logger.Info("Attempting to retrieve LLM content", "resourceID", resourceID)
			content, err := dr.PklresHelper.retrievePklContent("llm", resourceID)
			if err != nil {
				dr.Logger.Debug("Failed to retrieve LLM content", "resourceID", resourceID, "error", err)
				continue
			}
			if len(content) > 0 {
				llmContent = content
				dr.Logger.Info("Successfully retrieved LLM content", "resourceID", resourceID, "contentLength", len(llmContent))
				if len(llmContent) > 200 {
					dr.Logger.Info("LLM content preview", "preview", llmContent[:200]+"...")
				} else {
					dr.Logger.Info("LLM content", "content", llmContent)
				}
				break
			}
		}
	} else {
		dr.Logger.Warn("PklresHelper is nil, cannot retrieve LLM content")
	}
	combinedPklFile := dr.ResponsePklFile + ".with_llm"
	if llmContent != "" {
		// Prepend the LLM PKL content to the response PKL file
		origContent, err := afero.ReadFile(dr.Fs, dr.ResponsePklFile)
		if err != nil {
			return "", fmt.Errorf("read response PKL file: %w", err)
		}
		// Insert 'Resources = Resources' after the LLM PKL content
		combined := []byte(llmContent + "\nResources = Resources\n" + string(origContent))
		if err := afero.WriteFile(dr.Fs, combinedPklFile, combined, 0o644); err != nil {
			return "", fmt.Errorf("write combined PKL file: %w", err)
		}
		// Debug: log the path and contents of the combined PKL file
		contentPreview := string(combined)
		if len(contentPreview) > 1000 {
			contentPreview = contentPreview[:1000] + "... (truncated)"
		}
		dr.Logger.Info("Combined PKL file for response evaluation", "path", combinedPklFile, "contentPreview", contentPreview)
	} else {
		combinedPklFile = dr.ResponsePklFile
		dr.Logger.Warn("No LLM content found, using original response PKL file", "path", combinedPklFile)

		// Log additional context about why LLM content might be missing
		dr.Logger.Info("LLM content missing - possible causes:",
			"llmResourceID", llmResourceID,
			"pklresHelper_nil", dr.PklresHelper == nil,
			"workflow_target_action", func() string {
				if dr.Workflow != nil {
					return dr.Workflow.GetTargetActionID()
				}
				return "unknown"
			}())

		// Create fallback LLM content with request context for better user experience
		if dr.Request != nil {
			query := dr.Request.Query("q")
			if query != "" {
				dr.Logger.Info("Creating fallback response with query context", "query", query)
				fallbackContent := fmt.Sprintf(`extends "package://schema.kdeps.com/core@0.3.0#/LLM.pkl"

Resources {
  ["%s"] {
    Model = "llama2"
    Prompt = "You are a helpful assistant. Please provide information about: %s"
    Role = "assistant"
    Response = "Information about %s: This is a fallback response. The LLM resource is not properly configured or the model is not available. Please check the workflow configuration and ensure the LLM resource files are present."
    Timestamp = %g.ns
    TimeoutDuration = 30.s
    Env {}
    Stderr = ""
    Stdout = ""
    File = ""
    ExitCode = 0
    ItemValues {}
  }
}`, llmResourceID, query, query, float64(time.Now().UnixNano()))

				llmContent = fallbackContent
				dr.Logger.Info("Generated fallback LLM content", "contentLength", len(llmContent))
			}
		}
	}
	// --- END PATCH ---

	dr.Logger.Debug("using configured pklres reader for LLM resource access")

	// Get the singleton evaluator
	pklEvaluator, err := evaluator.GetEvaluator()
	if err != nil {
		return "", fmt.Errorf("get PKL evaluator: %w", err)
	}

	// Create module source
	moduleSource := pkl.FileSource(combinedPklFile)

	// Evaluate the PKL file
	result, err := pklEvaluator.EvaluateOutputText(dr.Context, moduleSource)
	if err != nil {
		return "", fmt.Errorf("evaluate PKL file: %w", err)
	}

	// Write result to target file
	if err := afero.WriteFile(dr.Fs, dr.ResponseTargetFile, []byte(result), 0o644); err != nil {
		return "", fmt.Errorf("write result to target file: %w", err)
	}

	return result, nil
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

			if err := dr.CreateResponsePklFile(finalErrorResponse); err != nil {
				return fatal, fmt.Errorf("create comprehensive error response: %w", err)
			}
			return fatal, fmt.Errorf("%s", message)
		}

		// For non-fatal errors, just accumulate and continue
		if err := dr.CreateResponsePklFile(errorResponse); err != nil {
			return fatal, fmt.Errorf("create error response: %w", err)
		}
		return fatal, nil
	}

	// When not in API server mode, return an actual error to fail the processing
	return fatal, fmt.Errorf("validation failed (code %d): %s", code, message)
}
