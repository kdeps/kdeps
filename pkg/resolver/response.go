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
func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock apiserverresponse.APIServerResponse) error {
	dr.Logger.Debug("Starting CreateResponsePklFile", "response", apiResponseBlock)

	if err := dr.ensureResponsePklFileNotExists(); err != nil {
		dr.Logger.Error("Failed to ensure response PKL file does not exist", "error", err)
		return err
	}

	sections := dr.buildResponseSections(apiResponseBlock)
	dr.Logger.Debug("Built response sections", "sections", sections)

	if err := evaluator.CreateAndProcessPklFile(dr.Fs, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evaluator.EvalPkl, false); err != nil {
		dr.Logger.Error("Failed to create/process PKL file", "error", err)
		return fmt.Errorf("failed to create/process PKL file: %w", err)
	}

	dr.Logger.Debug("Successfully created and processed PKL file", "file", dr.ResponsePklFile)
	return nil
}

// ensureResponsePklFileNotExists removes the existing PKL file if it exists.
func (dr *DependencyResolver) ensureResponsePklFileNotExists() error {
	dr.Logger.Debug("Checking if response PKL file exists", "file", dr.ResponsePklFile)

	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		dr.Logger.Error("Error checking file existence", "error", err)
		return fmt.Errorf("failed to check file existence: %w", err)
	}

	if exists {
		dr.Logger.Warn("Response PKL file already exists. Removing it.", "file", dr.ResponsePklFile)
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			dr.Logger.Error("Failed to delete old response PKL file", "file", dr.ResponsePklFile, "error", err)
			return fmt.Errorf("failed to delete old response file: %w", err)
		}
		dr.Logger.Debug("Old response PKL file deleted", "file", dr.ResponsePklFile)
	}

	return nil
}

// buildResponseSections creates sections for the PKL file from the API response.
func (dr *DependencyResolver) buildResponseSections(apiResponseBlock apiserverresponse.APIServerResponse) []string {
	dr.Logger.Debug("Building response sections from API response", "response", apiResponseBlock)

	sections := []string{
		fmt.Sprintf("success = %v", apiResponseBlock.GetSuccess()),
		formatResponseData(apiResponseBlock.GetResponse()),
		formatErrors(apiResponseBlock.GetErrors(), dr.Logger),
	}

	dr.Logger.Debug("Response sections built", "sections", sections)
	return sections
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
func formatErrors(errors *[]*apiserverresponse.APIServerErrorsBlock, logger *log.Logger) string {
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
func decodeErrorMessage(message string, logger *log.Logger) string {
	if message == "" {
		return ""
	}
	logger.Debug("Decoding error message", "message", message)
	decoded, err := utils.DecodeBase64String(message)
	if err != nil {
		logger.Warn("Failed to decode error message", "message", message, "error", err)
		return message
	}
	logger.Debug("Decoded error message", "decoded", decoded)
	return decoded
}

// EvalPklFormattedResponseFile evaluates a PKL file and formats the result as JSON.
func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	dr.Logger.Debug("Evaluating PKL file", "file", dr.ResponsePklFile)

	// Check if the response PKL file exists
	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		dr.Logger.Error("Error checking existence of PKL file", "file", dr.ResponsePklFile, "error", err)
		return "", fmt.Errorf("failed to check if PKL file exists: %w", err)
	}
	if !exists {
		dr.Logger.Error("PKL file does not exist", "file", dr.ResponsePklFile)
		return "", fmt.Errorf("PKL file does not exist: %s", dr.ResponsePklFile)
	}

	dr.Logger.Debug("PKL file exists, proceeding with validation", "file", dr.ResponsePklFile)

	// Validate the file extension
	if err := dr.validatePklFileExtension(); err != nil {
		dr.Logger.Error("Validation failed for PKL file extension", "error", err)
		return "", err
	}

	// Ensure the response target file does not exist
	if err := dr.ensureResponseTargetFileNotExists(); err != nil {
		dr.Logger.Error("Failed to ensure target file does not exist", "error", err)
		return "", err
	}

	// Check if the PKL binary exists
	if err := evaluator.EnsurePklBinaryExists(dr.Logger); err != nil {
		dr.Logger.Error("PKL binary not found", "error", err)
		return "", err
	}

	// Execute the PKL evaluation command
	result, err := dr.executePklEvalCommand()
	if err != nil {
		dr.Logger.Error("Failed to execute PKL evaluation command", "error", err)
		return "", err
	}

	dr.Logger.Debug("PKL evaluation successful", "result", result.Stdout)
	return result.Stdout, nil
}

// validatePklFileExtension checks if the response file has a .pkl extension.
func (dr *DependencyResolver) validatePklFileExtension() error {
	dr.Logger.Debug("Validating PKL file extension", "file", dr.ResponsePklFile)
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", dr.ResponsePklFile)
		dr.Logger.Error(errMsg)
		return fmt.Errorf(errMsg)
	}
	return nil
}

// ensureResponseTargetFileNotExists removes the existing target file if it exists.
func (dr *DependencyResolver) ensureResponseTargetFileNotExists() error {
	dr.Logger.Debug("Checking if response target file exists", "file", dr.ResponseTargetFile)

	exists, err := afero.Exists(dr.Fs, dr.ResponseTargetFile)
	if err != nil {
		dr.Logger.Error("Error checking target file existence", "error", err)
		return err
	}

	if exists {
		dr.Logger.Warn("Target file already exists. Removing it.", "file", dr.ResponseTargetFile)
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("Failed to delete old target file", "file", dr.ResponseTargetFile, "error", err)
			return err
		}
		dr.Logger.Debug("Old target file deleted", "file", dr.ResponseTargetFile)
	}

	return nil
}

// executePklEvalCommand runs the 'pkl eval' command and checks the result.
func (dr *DependencyResolver) executePklEvalCommand() (execute.ExecResult, error) {
	dr.Logger.Debug("Executing PKL evaluation command", "file", dr.ResponsePklFile, "output", dr.ResponseTargetFile)

	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", "--format", "json", "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
		StreamStdio: false,
	}

	result, err := cmd.Execute(context.Background())
	if err != nil {
		dr.Logger.Error("Command execution failed", "error", err)
		return execute.ExecResult{}, fmt.Errorf("command execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		errMsg := fmt.Sprintf("Command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		dr.Logger.Error(errMsg)
		return execute.ExecResult{}, fmt.Errorf(errMsg)
	}

	dr.Logger.Debug("Command executed successfully", "stdout", result.Stdout)
	return result, nil
}

// HandleAPIErrorResponse handles API error responses by creating a PKL file.
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string, fatal bool) (bool, error) {
	dr.Logger.Debug("Handling API error response", "code", code, "message", message, "fatal", fatal)

	if dr.ApiServerMode {
		errorResponse := utils.NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(errorResponse); err != nil {
			dr.Logger.Error("Failed to create error response PKL file", "error", err)
			return fatal, err
		}
		dr.Logger.Debug("Error response PKL file created successfully")
	}
	return fatal, nil
}
