package resolver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/kdepsexec"
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
	if err := evaluator.CreateAndProcessPklFile(dr.Fs, dr.Context, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evaluator.EvalPkl, false); err != nil {
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
	isSuccess := apiResponseBlock.GetSuccess() && len(responseErrors) == 0

	sections := []string{
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Document.pkl" as document`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Memory.pkl" as memory`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Session.pkl" as session`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Tool.pkl" as tool`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Item.pkl" as item`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf("Success = %v", isSuccess),
		formatResponseMeta(requestID, apiResponseBlock.GetMeta()),
		formatResponseData(apiResponseBlock.GetResponse()),
		formatErrors(&responseErrors, dr.Logger),
	}
	return sections
}

func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	responseData := make([]string, 0, len(response.Data))
	for _, v := range response.Data {
		responseData = append(responseData, formatDataValue(v))
	}

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

func formatResponseMeta(requestID string, meta *apiserverresponse.APIServerResponseMetaBlock) string {
	if meta == nil || *meta.Headers == nil && *meta.Properties == nil {
		return fmt.Sprintf(`
meta {
  RequestID = "%s"
}
`, requestID)
	}

	responseMetaHeaders := utils.FormatResponseHeaders(*meta.Headers)
	responseMetaProperties := utils.FormatResponseProperties(*meta.Properties)

	if len(responseMetaHeaders) == 0 && len(responseMetaProperties) == 0 {
		return fmt.Sprintf(`
meta {
  RequestID = "%s"
}
`, requestID)
	}

	return fmt.Sprintf(`
meta {
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
  document.JSONRenderDocument(JSONDocument_%s)
else
  document.JSONRenderDocument((if (document.JSONParser(JSONDocument_%s) != null) document.JSONParser(JSONDocument_%s) else JSONDocument_%s))
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
    code = %d
    message = #"""
%s
"""#
  }`, err.Code, decodedMessage)
		}
	}

	if newBlocks != "" {
		return fmt.Sprintf(`errors {%s
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

	if err := evaluator.EnsurePklBinaryExists(dr.Context, dr.Logger); err != nil {
		return "", fmt.Errorf("PKL binary check: %w", err)
	}

	result, err := dr.executePklEvalCommand()
	if err != nil {
		return "", fmt.Errorf("execute PKL eval: %w", err)
	}
	return result.Stdout, nil
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

func (dr *DependencyResolver) executePklEvalCommand() (kdepsexecStd struct {
	Stdout, Stderr string
	ExitCode       int
}, err error,
) {
	stdout, stderr, exitCode, err := kdepsexec.KdepsExec(
		dr.Context,
		"pkl",
		[]string{"eval", "--format", "json", "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
		"",
		false,
		false,
		dr.Logger,
	)
	if err != nil {
		return kdepsexecStd, err
	}
	if exitCode != 0 {
		return kdepsexecStd, fmt.Errorf("command failed with exit code %d: %s", exitCode, stderr)
	}
	kdepsexecStd.Stdout = stdout
	kdepsexecStd.Stderr = stderr
	kdepsexecStd.ExitCode = exitCode
	return kdepsexecStd, nil
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
			finalErrorResponse := &apiserverresponse.APIServerResponseImpl{
				Success:  false,
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
