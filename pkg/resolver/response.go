package resolver

import (
	"context"
	"fmt"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/utils"
	"path/filepath"
	"strings"

	"github.com/alexellis/go-execute/v2"
	"github.com/charmbracelet/log"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
)

// CreateResponsePklFile generates a PKL file from the API response and processes it.
func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock *apiserverresponse.APIServerResponse) error {
	if err := dr.ensureResponsePklFileNotExists(); err != nil {
		return err
	}

	sections := dr.buildResponseSections(apiResponseBlock)

	if err := evaluator.CreateAndProcessPklFile(dr.Fs, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evaluator.EvalPkl, false); err != nil {
		return fmt.Errorf("failed to create/process PKL file: %w", err)
	}

	return nil
}

// ensureResponsePklFileNotExists removes the existing PKL file if it exists.
func (dr *DependencyResolver) ensureResponsePklFileNotExists() error {
	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		return fmt.Errorf("failed to check file existence: %w", err)
	}
	if exists {
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			dr.Logger.Error("Unable to delete old response file", "response-pkl-file", dr.ResponsePklFile)
			return fmt.Errorf("failed to delete old response file: %w", err)
		}
	}
	return nil
}

// buildResponseSections creates sections for the PKL file from the API response.
func (dr *DependencyResolver) buildResponseSections(apiResponseBlock *apiserverresponse.APIServerResponse) []string {
	return []string{
		fmt.Sprintf("success = %v", apiResponseBlock.Success),
		formatResponseData(apiResponseBlock.Response),
		formatErrors(apiResponseBlock.Errors, dr.Logger),
	}
}

// formatResponseData formats the response data for the PKL file.
func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	var responseData []string
	for _, v := range response.Data {
		responseData = append(responseData, formatDataValue(v))
	}

	if len(responseData) > 0 {
		return fmt.Sprintf(`
response {
  data {
%s
  }
}`, strings.Join(responseData, "\n    "))
	}
	return ""
}

// formatDataValue formats a single data value for inclusion in the response.
func formatDataValue(value interface{}) string {
	strVal, ok := value.(string)
	if !ok {
		strVal = fmt.Sprintf("%v", value)
	}
	return fmt.Sprintf(`
#"""
%v
"""#
`, strVal)
}

// formatErrors formats error messages with optional base64 decoding.
func formatErrors(errors *apiserverresponse.APIServerErrorsBlock, logger *log.Logger) string {
	if errors == nil {
		return ""
	}

	decodedMessage := decodeErrorMessage(errors.Message, logger)

	return fmt.Sprintf(`
errors {
  code = %d
  message = %q
}`, errors.Code, decodedMessage)
}

// decodeErrorMessage attempts to base64 decode the error message.
func decodeErrorMessage(message string, logger *log.Logger) string {
	if message == "" {
		return ""
	}
	decoded, err := utils.DecodeBase64String(message)
	if err != nil {
		logger.Warn("Failed to decode error message", "message", message, "error", err)
		return message
	}
	return decoded
}

// EvalPklFormattedResponseFile evaluates a PKL file and formats the result as JSON.
func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	if err := dr.validatePklFileExtension(); err != nil {
		return "", err
	}

	if err := dr.ensureResponseTargetFileNotExists(); err != nil {
		return "", err
	}

	if err := evaluator.EnsurePklBinaryExists(dr.Logger); err != nil {
		return "", err
	}

	result, err := dr.executePklEvalCommand()
	if err != nil {
		return "", err
	}

	return result.Stdout, nil
}

// validatePklFileExtension checks if the response file has a .pkl extension.
func (dr *DependencyResolver) validatePklFileExtension() error {
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", dr.ResponsePklFile)
		dr.Logger.Error(errMsg)
		return fmt.Errorf(errMsg)
	}
	return nil
}

// ensureResponseTargetFileNotExists removes the existing target file if it exists.
func (dr *DependencyResolver) ensureResponseTargetFileNotExists() error {
	exists, err := afero.Exists(dr.Fs, dr.ResponseTargetFile)
	if err != nil {
		return err
	}
	if exists {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("Unable to delete old response target file", "response-target-file", dr.ResponsePklFile)
			return err
		}
	}
	return nil
}

// executePklEvalCommand runs the 'pkl eval' command and checks the result.
func (dr *DependencyResolver) executePklEvalCommand() (execute.ExecResult, error) {
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
		errMsg := fmt.Sprintf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		dr.Logger.Error(errMsg)
		return execute.ExecResult{}, fmt.Errorf(errMsg)
	}

	return result, nil
}

// HandleAPIErrorResponse handles API error responses by creating a PKL file.
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string) error {
	if dr.ApiServerMode {
		errorResponse := utils.NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(&errorResponse); err != nil {
			dr.Logger.Error("Failed to create error response file", "error", err)
			return err
		}
	}
	return nil
}
