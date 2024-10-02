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

	// Check the ResponseFlag (assuming this is a precondition)
	if err := dr.GetResponseFlag(); err != nil {
		return err
	}

	// Check if the response file already exists, and remove it if so
	if _, err := dr.Fs.Stat(dr.ResponsePklFile); err == nil {
		if err := dr.Fs.RemoveAll(dr.ResponsePklFile); err != nil {
			dr.Logger.Error("Unable to delete old response file", "response-pkl-file", dr.ResponsePklFile)
			return err
		}
	}

	// Format the success as "success = true/false"
	successStr := fmt.Sprintf("success = %v", success)

	// Process the response block
	if apiResponseBlock.Response != nil && apiResponseBlock.Response.Data != nil {
		// Convert the data slice to a string representation
		responseData = make([]string, len(apiResponseBlock.Response.Data))
		for i, v := range apiResponseBlock.Response.Data {
			responseData[i] = fmt.Sprintf(`
"""
%v
"""
`, v) // Convert each item to a string
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

	// Process the errors block
	if apiResponseBlock.Errors != nil {
		errorsStr = fmt.Sprintf(`
errors {
  code = %d
  message = %q
}`, apiResponseBlock.Errors.Code, apiResponseBlock.Errors.Message)
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

func (dr *DependencyResolver) GetResponseFlag() error {
	responseFiles := []struct {
		Flag              string
		Ext               string
		PklResponseFormat string
	}{
		{"response-jsonnet", ".json", "jsonnet"},
		{"response-txtpb", ".txtpb", "textproto"},
		{"response-yaml", ".yaml", "yaml"},
		{"response-plist", ".plist", "plist"},
		{"response-xml", ".xml", "xml"},
		{"response-pcf", ".pcf", "pcf"},
		{"response-json", ".json", "json"},
	}

	// Loop through each response flag file and check its existence
	for _, file := range responseFiles {
		dr.ResponseFlag = filepath.Join(dr.ActionDir, "/api/"+file.Flag)

		// Check if the response flag file exists
		exists, err := afero.Exists(dr.Fs, dr.ResponseFlag)
		if err != nil {
			return fmt.Errorf("error checking file existence: %w", err)
		}

		if exists {
			// If the file exists, return the file extension and content type
			fmt.Printf("Response flag file found: %s\n", dr.ResponseFlag)
			dr.ResponseType = file.PklResponseFormat
			dr.ResponseTargetFile = filepath.Join(dr.ActionDir, fmt.Sprintf("/api/response%s", file.Ext))
			return nil
		}
	}

	// If no response flag file is found, return an error
	return fmt.Errorf("no valid response flag file found in %s", dr.ActionDir)
}

func (dr *DependencyResolver) EvalPklFormattedResponseFile() (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(dr.ResponsePklFile) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", dr.ResponsePklFile)
		dr.Logger.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	if _, err := dr.Fs.Stat(dr.ResponseTargetFile); err == nil {
		if err := dr.Fs.RemoveAll(dr.ResponseTargetFile); err != nil {
			dr.Logger.Error("Unable to delete old response file", "response-file", dr.ResponseTargetFile)
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
