package resolver

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"
	"github.com/google/uuid"
	"github.com/kaptinlin/jsonrepair"
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
		return fmt.Errorf("ensure response PKL file does not exist: %w", err)
	}

	readers := []pkl.ResourceReader{
		dr.MemoryReader,
		dr.SessionReader,
		dr.ToolReader,
		dr.ItemReader,
	}

	sections := dr.buildResponseSections(dr.RequestID, apiResponseBlock)
	if err := evaluator.CreateAndProcessPklFile(dr.Fs, dr.Context, sections, dr.ResponsePklFile, "APIServerResponse.pkl", readers, dr.Logger, evaluator.EvalPkl, false); err != nil {
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
	sections := []string{
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Document.pkl" as document`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Memory.pkl" as memory`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Session.pkl" as session`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Tool.pkl" as tool`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf(`import "package://schema.kdeps.com/core@%s#/Item.pkl" as item`, schema.SchemaVersion(dr.Context)),
		fmt.Sprintf("success = %v", apiResponseBlock.GetSuccess()),
		formatResponseMeta(requestID, apiResponseBlock.GetMeta()),
		formatResponseData(apiResponseBlock.GetResponse()),
		formatErrors(apiResponseBlock.GetErrors(), dr.Logger),
	}
	return sections
}

func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	responseData := make([]string, 0, len(response.Data))
	for _, v := range response.Data {
		val := v
		// Assert v is []byte
		byteVal, ok := val.([]byte)
		if ok {
			// Repair the JSON string
			repaired, err := jsonrepair.JSONRepair(string(byteVal))
			if err == nil {
				val = repaired
			}
		}

		responseData = append(responseData, formatDataValue(val))
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
  requestID = "%s"
}
`, requestID)
	}

	responseMetaHeaders := utils.FormatResponseHeaders(*meta.Headers)
	responseMetaProperties := utils.FormatResponseProperties(*meta.Properties)

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

func (dr *DependencyResolver) executePklEvalCommand() (execute.ExecResult, error) {
	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", "--format", "json", "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
		StreamStdio: false,
	}

	result, err := cmd.Execute(dr.Context)
	if err != nil {
		return execute.ExecResult{}, fmt.Errorf("execute command: %w", err)
	}

	if result.ExitCode != 0 {
		return execute.ExecResult{}, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return result, nil
}

// HandleAPIErrorResponse creates an error response PKL file.
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string, fatal bool) (bool, error) {
	if dr.APIServerMode {
		errorResponse := utils.NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(errorResponse); err != nil {
			return fatal, fmt.Errorf("create error response: %w", err)
		}
	}
	return fatal, nil
}
