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

func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock *apiserverresponse.APIServerResponse) error {
	// Check if the response file already exists and remove it if so
	if exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile); err != nil {
		return fmt.Errorf("failed to check file existence: %w", err)
	} else if exists {
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			dr.Logger.Error("Unable to delete old response file", "response-pkl-file", dr.ResponsePklFile)
			return fmt.Errorf("failed to delete old response file: %w", err)
		}
	}

	// Prepare response sections
	sections := []string{
		fmt.Sprintf("success = %v", apiResponseBlock.Success),
		formatResponseData(apiResponseBlock.Response),
		formatErrors(apiResponseBlock.Errors, dr.Logger),
	}

	// Create and process the PKL file
	if err := evaluator.CreateAndProcessPklFile(dr.Fs, sections, dr.ResponsePklFile, "APIServerResponse.pkl", dr.Logger, evaluator.EvalPkl); err != nil {
		return fmt.Errorf("failed to create/process PKL file: %w", err)
	}

	return nil
}

// Helper function to format the response data
func formatResponseData(response *apiserverresponse.APIServerResponseBlock) string {
	if response == nil || response.Data == nil {
		return ""
	}

	var responseData []string
	for _, v := range response.Data {
		strVal, ok := v.(string)
		if !ok {
			strVal = fmt.Sprintf("%v", v)
		}
		responseData = append(responseData, fmt.Sprintf(`
"""
%v
"""
`, strVal))
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

// Helper function to format errors with optional base64 decoding
func formatErrors(errors *apiserverresponse.APIServerErrorsBlock, logger *log.Logger) string {
	if errors == nil {
		return ""
	}

	decodedMessage := errors.Message
	if decodedMessage != "" {
		if decoded, err := utils.DecodeBase64String(decodedMessage); err == nil {
			decodedMessage = decoded
		} else {
			logger.Warn("Failed to decode error message", "message", errors.Message, "error", err)
		}
	}

	return fmt.Sprintf(`
errors {
  code = %d
  message = %q
}`, errors.Code, decodedMessage)
}

func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", dr.ResponsePklFile)
		dr.Logger.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Check if the response file already exists, and remove it if so
	exists, err := afero.Exists(dr.Fs, dr.ResponseTargetFile)
	if err != nil {
		return "", err
	}

	if exists {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("Unable to delete old response target file", "response-target-file", dr.ResponsePklFile)
			return "", err
		}

	}

	// Ensure that the 'pkl' binary is available
	if err := evaluator.EnsurePklBinaryExists(dr.Logger); err != nil {
		return "", err
	}

	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", "--format", "json", "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(context.Background())
	if err != nil {
		errMsg := "command execution failed"
		dr.Logger.Error(errMsg, "error", err)
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	// Check for non-zero exit code
	if result.ExitCode != 0 {
		errMsg := fmt.Sprintf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		dr.Logger.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	return result.Stdout, nil
}

// Helper function to Handle API error responses
func (dr *DependencyResolver) HandleAPIErrorResponse(code int, message string) error {
	if dr.ApiServerMode {
		errorResponse := utils.NewAPIServerResponse(false, nil, code, message)
		if err := dr.CreateResponsePklFile(&errorResponse); err != nil {
			dr.Logger.Error("Failed to create error response file:", err)
			return err
		}
	}
	return nil
}
