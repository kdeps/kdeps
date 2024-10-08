package resolver

import (
	"context"
	"fmt"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/utils"
	"path/filepath"
	"strings"

	"github.com/alexellis/go-execute/v2"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) CreateResponsePklFile(apiResponseBlock *apiserverresponse.APIServerResponse) error {
	success := apiResponseBlock.Success
	var responseData []string
	var errorsStr string

	// Check if the response file already exists, and remove it if so
	exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
	if err != nil {
		return err
	}

	if exists {
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			dr.Logger.Error("Unable to delete old response file", "response-pkl-file", dr.ResponsePklFile)
			return err
		}

	}

	// Format the success as "success = true/false"
	successStr := fmt.Sprintf("success = %v", success)

	// Process the response block and decode any Base64-encoded data
	if apiResponseBlock.Response != nil && apiResponseBlock.Response.Data != nil {
		// Convert the data slice to a string representation
		responseData = make([]string, len(apiResponseBlock.Response.Data))
		for i, v := range apiResponseBlock.Response.Data {
			// Type assertion to ensure v is a string
			if strVal, ok := v.(string); ok {
				// Attempt to decode the Base64-encoded data
				decodedData, err := utils.DecodeBase64String(strVal)
				if err != nil {
					decodedData = strVal // If decoding fails, use the original string
				}
				responseData[i] = fmt.Sprintf(`
"""
%v
"""
`, decodedData) // Format the decoded data into the response
			} else {
				// Handle case where the data is not a string
				dr.Logger.Warn("Non-string data found in Response.Data", "data", v)
				responseData[i] = fmt.Sprintf(`
"""
%v
"""
`, v) // Just format the non-string value as-is
			}
		}
	}

	// Format the response block as "response { data { ... } }"
	var responseStr string
	if len(responseData) > 0 {
		responseStr = fmt.Sprintf(`
response {
  data {
%s
  }
}`, strings.Join(responseData, "\n    ")) // Properly format the data block with indentation
	}

	// Process the errors block and decode any Base64-encoded error message
	if apiResponseBlock.Errors != nil {
		decodedErrorMessage := apiResponseBlock.Errors.Message
		// Check if the error message is Base64-encoded
		if decodedErrorMessage != "" {
			decoded, err := utils.DecodeBase64String(apiResponseBlock.Errors.Message)
			if err == nil {
				decodedErrorMessage = decoded // Use the decoded message if successful
			}
		}
		errorsStr = fmt.Sprintf(`
errors {
  code = %d
  message = %q
}`, apiResponseBlock.Errors.Code, decodedErrorMessage)
	}

	// Combine everything into sections as []string
	sections := []string{successStr, responseStr, errorsStr}

	// Create and process the PKL file
	if err := evaluator.CreateAndProcessPklFile(dr.Fs, sections, dr.ResponsePklFile, "APIServerResponse.pkl",
		nil, dr.Logger, evaluator.EvalPkl); err != nil {
		return err
	}

	return nil
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
		Args:        []string{"eval", "--format", dr.ResponseType, "--output-path", dr.ResponseTargetFile, dr.ResponsePklFile},
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
