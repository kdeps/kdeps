package resolver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/alexellis/go-execute/v2"
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
	dr.Logger.Debug("starting CreateResponsePklFile", "response", apiResponseBlock)

	if err := dr.ensureResponsePklFileNotExists(); err != nil {
		dr.Logger.Error("failed to ensure response PKL file does not exist", "error", err)
		return err
	}

	sections := dr.buildResponseSections(apiResponseBlock)
	dr.Logger.Debug("built response sections", "sections", sections)

	if err := evaluator.CreateAndProcessPklFile(dr.Fs, dr.Context, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evaluator.EvalPkl, false); err != nil {
		dr.Logger.Error("failed to create/process PKL file", "error", err)
		return fmt.Errorf("failed to create/process PKL file: %w", err)
	}

	dr.Logger.Debug("successfully created and processed PKL file", "file", dr.ResponsePklFile)
	return nil
}

// ensureResponsePklFileNotExists removes the existing PKL file if it exists.
func (dr *DependencyResolver) ensureResponsePklFileNotExists() error {
	dr.Logger.Debug("checking if response PKL file exists", "file", dr.ResponsePklFile)

	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		dr.Logger.Error("error checking file existence", "error", err)
		return fmt.Errorf("failed to check file existence: %w", err)
	}

	if exists {
		dr.Logger.Warn("response PKL file already exists. Removing it.", "file", dr.ResponsePklFile)
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			dr.Logger.Error("failed to delete old response PKL file", "file", dr.ResponsePklFile, "error", err)
			return fmt.Errorf("failed to delete old response file: %w", err)
		}
		dr.Logger.Debug("old response PKL file deleted", "file", dr.ResponsePklFile)
	}

	return nil
}

// buildResponseSections creates sections for the PKL file from the API response.
func (dr *DependencyResolver) buildResponseSections(apiResponseBlock apiserverresponse.APIServerResponse) []string {
	dr.Logger.Debug("building response sections from API response", "response", apiResponseBlock)

	sections := []string{
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Document.pkl" as document`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf("success = %v", apiResponseBlock.GetSuccess()),
		formatResponseData(apiResponseBlock.GetResponse()),
		formatErrors(apiResponseBlock.GetErrors(), dr.Logger),
	}

	dr.Logger.Debug("response sections built", "sections", sections)
	return sections
}

// formatResponseData formats the response data for the PKL file.
func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	// Preallocate the slice to avoid reallocations.
	responseData := make([]string, 0, len(response.Data))
	for _, v := range response.Data {
		responseData = append(responseData, formatDataValue(v))
	}

	// If there is data, format it; otherwise, return an empty string.
	if len(responseData) == 0 {
		return ""
	}

	return fmt.Sprintf(`
response {
  data {
%s
  }
}`, strings.Join(responseData, "\n    "))
}

// formatMap recursively formats a map[interface{}]interface{} for rendering.
func formatMap(m map[interface{}]interface{}) string {
	mappingParts := []string{"new Mapping {"}
	for k, v := range m {
		var keyStr, valueStr string
		keyStr = strings.ReplaceAll(fmt.Sprintf("%v", k), `"`, `\"`)

		// Handle nested maps and dynamic objects
		valueStr = formatValue(v)

		mappingParts = append(mappingParts, fmt.Sprintf(`    ["%s"] = %s`, keyStr, valueStr))
	}
	mappingParts = append(mappingParts, "}")
	return strings.Join(mappingParts, "\n")
}

// formatValue handles formatting for various data types, including nested maps and dynamic objects.
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		// Convert map[string]interface{} to map[interface{}]interface{}
		m := make(map[interface{}]interface{})
		for key, val := range v {
			m[key] = val
		}
		return formatMap(m)
	case map[interface{}]interface{}:
		return formatMap(v) // Recursively format nested maps
	case nil:
		return "null"
	default:
		// Check for dynamic struct-like objects
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			return formatValue(rv.Elem().Interface()) // Dereference pointer
		}
		if rv.Kind() == reflect.Struct {
			// Convert struct to map for formatting
			structMap := structToMap(rv.Interface())
			return formatMap(structMap)
		}

		// Handle standard types (e.g., strings, numbers)
		if fmt.Sprintf("%v", v) == "" {
			return "null"
		}
		return fmt.Sprintf(`
"""
%v
"""
`, v)
	}
}

// structToMap converts a struct to a map[interface{}]interface{} using reflection.
func structToMap(s interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem() // Dereference if pointer
	}
	for i := range val.NumField() {
		fieldName := val.Type().Field(i).Name
		fieldValue := val.Field(i).Interface()
		result[fieldName] = fieldValue
	}
	return result
}

// formatDataValue formats a single data value for inclusion in the response.
func formatDataValue(value interface{}) string {
	// Generate UUID with underscores
	uuidVal := strings.ReplaceAll(uuid.New().String(), "-", "_")

	// Initialize the value string
	var val string

	// Check the value type
	switch v := value.(type) {
	case map[interface{}]interface{}, map[string]interface{}:
		// Handle maps with interface{} or string keys
		val = formatValue(v)
	default:
		// Use the helper function to format non-map types
		val = formatValue(v)
	}

	// Return the formatted string
	return fmt.Sprintf(`
local JSONDocument_%s = %s
local JSONDocumentType_%s = JSONDocument_%s is Mapping | Dynamic

if (JSONDocumentType_%s)
  document.JSONRenderDocument(JSONDocument_%s)
else
  document.JSONRenderDocument((if (document.JSONParser(JSONDocument_%s) != null) document.JSONParser(JSONDocument_%s) else JSONDocument_%s))
`, uuidVal, val, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal)
}

// formatErrors formats error messages with optional base64 decoding.
func formatErrors(errors *[]*apiserverresponse.APIServerErrorsBlock, logger *logging.Logger) string {
	// If no errors, return an empty string (no errors block is created)
	if errors == nil || len(*errors) == 0 {
		return ""
	}

	var newBlocks string
	for _, err := range *errors {
		if err != nil {
			decodedMessage := decodeErrorMessage(err.Message, logger)
			newBlocks += fmt.Sprintf(`
  new {
    code = %d
    message = %q
  }`, err.Code, decodedMessage)
		}
	}

	// Only create the errors block if there are valid error entries
	if newBlocks != "" {
		return fmt.Sprintf(`errors {%s
}`, newBlocks)
	}

	// No valid errors to format, return an empty string
	return ""
}

// decodeErrorMessage attempts to base64 decode the error message.
func decodeErrorMessage(message string, logger *logging.Logger) string {
	if message == "" {
		return ""
	}
	logger.Debug("decoding error message", "message", message)
	decoded, err := utils.DecodeBase64String(message)
	if err != nil {
		logger.Warn("failed to decode error message", "message", message, "error", err)
		return message
	}
	logger.Debug("decoded error message", "decoded", decoded)
	return decoded
}

// EvalPklFormattedResponseFile evaluates a PKL file and formats the result as JSON.
func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	dr.Logger.Debug("evaluating PKL file", "file", dr.ResponsePklFile)

	// Check if the response PKL file exists
	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		dr.Logger.Error("error checking existence of PKL file", "file", dr.ResponsePklFile, "error", err)
		return "", fmt.Errorf("failed to check if PKL file exists: %w", err)
	}
	if !exists {
		dr.Logger.Error("PKL file does not exist", "file", dr.ResponsePklFile)
		return "", fmt.Errorf("PKL file does not exist: %s", dr.ResponsePklFile)
	}

	dr.Logger.Debug("PKL file exists, proceeding with validation", "file", dr.ResponsePklFile)

	// Validate the file extension
	if err := dr.validatePklFileExtension(); err != nil {
		dr.Logger.Error("validation failed for PKL file extension", "error", err)
		return "", err
	}

	// Ensure the response target file does not exist
	if err := dr.ensureResponseTargetFileNotExists(); err != nil {
		dr.Logger.Error("failed to ensure target file does not exist", "error", err)
		return "", err
	}

	// Check if the PKL binary exists
	if err := evaluator.EnsurePklBinaryExists(dr.Context, dr.Logger); err != nil {
		dr.Logger.Error("PKL binary not found", "error", err)
		return "", err
	}

	// Execute the PKL evaluation command
	result, err := dr.executePklEvalCommand()
	if err != nil {
		dr.Logger.Error("failed to execute PKL evaluation command", "error", err)
		return "", err
	}

	dr.Logger.Debug("PKL evaluation successful", "result", result.Stdout)
	return result.Stdout, nil
}

// validatePklFileExtension checks if the response file has a .pkl extension.
func (dr *DependencyResolver) validatePklFileExtension() error {
	dr.Logger.Debug("validating PKL file extension", "file", dr.ResponsePklFile)
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", dr.ResponsePklFile)
		dr.Logger.Error(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// ensureResponseTargetFileNotExists removes the existing target file if it exists.
func (dr *DependencyResolver) ensureResponseTargetFileNotExists() error {
	dr.Logger.Debug("checking if response target file exists", "file", dr.ResponseTargetFile)

	exists, err := afero.Exists(dr.Fs, dr.ResponseTargetFile)
	if err != nil {
		dr.Logger.Error("error checking target file existence", "error", err)
		return err
	}

	if exists {
		dr.Logger.Warn("target file already exists. Removing it.", "file", dr.ResponseTargetFile)
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("failed to delete old target file", "file", dr.ResponseTargetFile, "error", err)
			return err
		}
		dr.Logger.Debug("old target file deleted", "file", dr.ResponseTargetFile)
	}

	return nil
}

// executePklEvalCommand runs the 'pkl eval' command and checks the result.
func (dr *DependencyResolver) executePklEvalCommand() (execute.ExecResult, error) {
	dr.Logger.Debug("executing PKL evaluation command", "file", dr.ResponsePklFile, "output", dr.ResponseTargetFile)

	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", "--format", "json", "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
		StreamStdio: false,
	}

	result, err := cmd.Execute(context.Background())
	if err != nil {
		dr.Logger.Error("command execution failed", "error", err)
		return execute.ExecResult{}, fmt.Errorf("command execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		errMsg := fmt.Sprintf("Command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		dr.Logger.Error(errMsg)
		return execute.ExecResult{}, errors.New(errMsg)
	}

	dr.Logger.Debug("command executed successfully", "stdout", result.Stdout)
	return result, nil
}

// HandleAPIErrorResponse handles API error responses by creating a PKL file.
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string, fatal bool) (bool, error) {
	dr.Logger.Debug("handling API error response", "code", code, "message", message, "fatal", fatal)

	if dr.APIServerMode {
		errorResponse := utils.NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(errorResponse); err != nil {
			dr.Logger.Error("failed to create error response PKL file", "error", err)
			return fatal, err
		}
		dr.Logger.Debug("error response PKL file created successfully")
	}
	return fatal, nil
}
