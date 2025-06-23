package resolver

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
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

	if err := dr.ensureResponsePklFileNotExists(); err != nil {
		return fmt.Errorf("ensure response PKL file does not exist: %w", err)
	}

	sections := dr.BuildResponseSections(dr.RequestID, apiResponseBlock)
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

func FormatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	responseData := make([]string, 0, len(response.Data))
	for _, v := range response.Data {
		responseData = append(responseData, FormatDataValue(v))
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

func FormatResponseMeta(requestID string, meta *apiserverresponse.APIServerResponseMetaBlock) string {
	if meta == nil || (meta.Headers == nil && meta.Properties == nil) {
		return fmt.Sprintf(`
meta {
  requestID = "%s"
}
`, requestID)
	}

	var responseMetaHeaders string
	var responseMetaProperties string

	if meta.Headers != nil && len(*meta.Headers) > 0 {
		responseMetaHeaders = utils.FormatResponseHeaders(*meta.Headers)
	}
	if meta.Properties != nil && len(*meta.Properties) > 0 {
		responseMetaProperties = utils.FormatResponseProperties(*meta.Properties)
	}

	if len(responseMetaHeaders) == 0 && len(responseMetaProperties) == 0 {
		return fmt.Sprintf(`
meta {
  requestID = "%s"
}
`, requestID)
	}

	return fmt.Sprintf(`
meta {
  requestID = "%s"
  %s
  %s
}`,
		requestID, responseMetaHeaders, responseMetaProperties)
}

func FormatMap(m map[interface{}]interface{}) string {
	mappingParts := []string{"new Mapping {"}
	for k, v := range m {
		keyStr := strings.ReplaceAll(fmt.Sprintf("%v", k), `"`, `\"`)
		valueStr := FormatValue(v)
		mappingParts = append(mappingParts, fmt.Sprintf(`    ["%s"] = %s`, keyStr, valueStr))
	}
	mappingParts = append(mappingParts, "}")
	return strings.Join(mappingParts, "\n")
}

func FormatValue(value interface{}) string {
	switch v := value.(type) {
	case map[string]interface{}:
		m := make(map[interface{}]interface{})
		for key, val := range v {
			m[key] = val
		}
		return FormatMap(m)
	case map[interface{}]interface{}:
		return FormatMap(v)
	case nil:
		return "null"
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			return FormatValue(rv.Elem().Interface())
		}
		if rv.Kind() == reflect.Struct {
			return FormatMap(StructToMap(rv.Interface()))
		}
		return fmt.Sprintf(`
"""
%v
"""
`, v)
	}
}

func StructToMap(s interface{}) map[interface{}]interface{} {
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

func FormatDataValue(value interface{}) string {
	uuidVal := strings.ReplaceAll(uuid.New().String(), "-", "_")
	val := FormatValue(value)
	return fmt.Sprintf(`
local JSONDocument_%s = %s
local JSONDocumentType_%s = JSONDocument_%s is Mapping | Dynamic

if (JSONDocumentType_%s)
  document.JSONRenderDocument(JSONDocument_%s)
else
  document.JSONRenderDocument((if (document.JSONParser(JSONDocument_%s) != null) document.JSONParser(JSONDocument_%s) else JSONDocument_%s))
`, uuidVal, val, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal, uuidVal)
}

func FormatErrors(errors *[]*apiserverresponse.APIServerErrorsBlock, logger *logging.Logger) string {
	if errors == nil || len(*errors) == 0 {
		return ""
	}

	var newBlocks string
	for _, err := range *errors {
		if err != nil {
			decodedMessage := DecodeErrorMessage(err.Message, logger)
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

func DecodeErrorMessage(message string, logger *logging.Logger) string {
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

	if err := dr.ValidatePklFileExtension(); err != nil {
		return "", err
	}

	if err := dr.EnsureResponseTargetFileNotExists(); err != nil {
		return "", fmt.Errorf("ensure target file not exists: %w", err)
	}

	if err := evaluator.EnsurePklBinaryExists(dr.Context, dr.Logger); err != nil {
		return "", fmt.Errorf("PKL binary check: %w", err)
	}

	result, err := dr.ExecutePklEvalCommand()
	if err != nil {
		return "", fmt.Errorf("execute PKL eval: %w", err)
	}
	return result.Stdout, nil
}
