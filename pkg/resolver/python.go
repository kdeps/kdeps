package resolver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandlePython(actionID string, pythonBlock *pklPython.ResourcePython) error {
	// Decode Script if it is Base64-encoded
	if utf8.ValidString(pythonBlock.Script) && utils.IsBase64Encoded(pythonBlock.Script) {
		decodedScript, err := utils.DecodeBase64String(pythonBlock.Script)
		if err == nil {
			pythonBlock.Script = decodedScript
		}
	}

	// Decode Stderr if it is Base64-encoded
	if pythonBlock.Stderr != nil && utf8.ValidString(*pythonBlock.Stderr) && utils.IsBase64Encoded(*pythonBlock.Stderr) {
		decodedStderr, err := utils.DecodeBase64String(*pythonBlock.Stderr)
		if err == nil {
			pythonBlock.Stderr = &decodedStderr
		}
	}

	// Decode Stdout if it is Base64-encoded
	if pythonBlock.Stdout != nil && utf8.ValidString(*pythonBlock.Stdout) && utils.IsBase64Encoded(*pythonBlock.Stdout) {
		decodedStdout, err := utils.DecodeBase64String(*pythonBlock.Stdout)
		if err == nil {
			pythonBlock.Stdout = &decodedStdout
		}
	}

	// Decode Env map keys and values if they are Base64-encoded
	if pythonBlock.Env != nil {
		for key, value := range *pythonBlock.Env {
			stringKey := key
			decodedValue := value

			// Decode value if it is Base64-encoded
			if utf8.ValidString(value) && utils.IsBase64Encoded(value) {
				decodedValue, _ = utils.DecodeBase64String(value)
			}

			// Update the map with the decoded values
			(*pythonBlock.Env)[stringKey] = decodedValue
		}
	}

	go func() error {
		err := dr.processPythonBlock(actionID, pythonBlock)
		if err != nil {
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) processPythonBlock(actionID string, pythonBlock *pklPython.ResourcePython) error {
	if dr.AnacondaInstalled {
		if *pythonBlock.CondaEnvironment != "" {
			execCommand := execute.ExecTask{
				Command:     "conda",
				Args:        []string{"activate", "--name", *pythonBlock.CondaEnvironment},
				Shell:       false,
				StreamStdio: false,
			}

			if _, err := execCommand.Execute(context.Background()); err != nil {
				return fmt.Errorf("execution failed for command '%s': %w", execCommand.Command, err)
			}
		}
	}

	var env []string
	if pythonBlock.Env != nil {
		for key, value := range *pythonBlock.Env {
			// Append the environment variable in the desired format
			env = append(env, fmt.Sprintf(`%s=%s`, key, value))
		}
	}

	// Create a temporary file using afero
	tmpFile, err := afero.TempFile(dr.Fs, "", "script-*.py")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		_ = dr.Fs.Remove(tmpFile.Name()) // Clean up the file after execution
	}()

	// Write the Python script to the temporary file
	if _, err := tmpFile.Write([]byte(pythonBlock.Script)); err != nil {
		return fmt.Errorf("failed to write script to temporary file: %w", err)
	}

	// Ensure the script is written and the file is closed
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Log the command and environment variables
	dr.Logger.Info("running python", "script", tmpFile.Name(), "env", env)

	// Prepare the execution command
	cmd := execute.ExecTask{
		Command:     "python3",
		Args:        []string{tmpFile.Name()},
		Shell:       false,
		Env:         env,
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(context.Background())
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Capture stdout and stderr
	pythonBlock.Stdout = &result.Stdout
	pythonBlock.Stderr = &result.Stderr

	// Append the Python entry
	if err := dr.AppendPythonEntry(actionID, pythonBlock); err != nil {
		return fmt.Errorf("failed to append Python entry: %w", err)
	}

	if dr.AnacondaInstalled {
		if *pythonBlock.CondaEnvironment != "" {
			execCommand := execute.ExecTask{
				Command:     "conda",
				Args:        []string{"deactivate"},
				Shell:       false,
				StreamStdio: false,
			}

			if _, err := execCommand.Execute(context.Background()); err != nil {
				return fmt.Errorf("execution failed for command '%s': %w", execCommand.Command, err)
			}
		}
	}

	return nil
}

func (dr *DependencyResolver) WritePythonStdoutToFile(resourceID string, pythonStdoutEncoded *string) (string, error) {
	// Convert resourceID to be filename friendly
	resourceIDFile := utils.ConvertToFilenameFriendly(resourceID)
	// Define the file path using the FilesDir and resource ID
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	// Ensure the ResponseBody is not nil
	if pythonStdoutEncoded != nil {
		// Prepare the content to write
		var content string
		if utils.IsBase64Encoded(*pythonStdoutEncoded) {
			// Decode the Base64-encoded ResponseBody string
			decodedResponseBody, err := utils.DecodeBase64String(*pythonStdoutEncoded)
			if err != nil {
				return "", fmt.Errorf("failed to decode Base64 string for resource ID: %s: %w", resourceID, err)
			}
			content = decodedResponseBody
		} else {
			// Use the ResponseBody content as-is if not Base64-encoded
			content = *pythonStdoutEncoded
		}

		// Write the content to the file
		err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write Python Stdout to file for resource ID: %s: %w", resourceID, err)
		}
	} else {
		return "", nil
	}

	return outputFilePath, nil
}

func (dr *DependencyResolver) AppendPythonEntry(resourceID string, newPython *pklPython.ResourcePython) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "python/"+dr.RequestID+"__python_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklPython.LoadFromPath(dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.GetResources() // Dereference the pointer to get the map

	// Check and Base64 encode Script, Stderr, Stdout if not already encoded
	encodedScript := newPython.Script
	if !utils.IsBase64Encoded(newPython.Script) {
		encodedScript = utils.EncodeBase64String(newPython.Script)
	}

	var filePath, encodedStderr, encodedStdout string

	if newPython.Stderr != nil {
		if !utils.IsBase64Encoded(*newPython.Stderr) {
			encodedStderr = utils.EncodeBase64String(*newPython.Stderr)
		} else {
			encodedStderr = *newPython.Stderr
		}
	}

	if newPython.Stdout != nil {
		filePath, err = dr.WritePythonStdoutToFile(resourceID, newPython.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write Python stdout to file: %w", err)
		}
		newPython.File = &filePath

		if !utils.IsBase64Encoded(*newPython.Stdout) {
			encodedStdout = utils.EncodeBase64String(*newPython.Stdout)
		} else {
			encodedStdout = *newPython.Stdout
		}
	}

	// Base64 encode the Env map (keys and values)
	var encodedEnv *map[string]string
	if newPython.Env != nil {
		encodedEnvMap := make(map[string]string)
		for key, value := range *newPython.Env {
			stringKey := key
			encodedValue := value

			if !utils.IsBase64Encoded(value) {
				encodedValue = utils.EncodeBase64String(value)
			}
			encodedEnvMap[stringKey] = encodedValue
		}
		encodedEnv = &encodedEnvMap
	}

	// Create or update the ResourcePython entry
	existingResources[resourceID] = &pklPython.ResourcePython{
		Env:       encodedEnv,
		Script:    encodedScript,
		Stderr:    &encodedStderr,
		Stdout:    &encodedStdout,
		File:      &filePath,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Python.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    script = \"%s\"\n", resource.Script))
		pklContent.WriteString(fmt.Sprintf("    timeoutDuration = %d\n", resource.TimeoutDuration))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))

		// Write environment variables (if Env is not nil)
		if resource.Env != nil {
			pklContent.WriteString("    env {\n")
			for key, value := range *resource.Env {
				pklContent.WriteString(fmt.Sprintf("      [\"%s\"] = \"%s\"\n", key, value))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    env {[\"HELLO\"] = \"WORLD\"\n}\n")
		}

		// Dereference to pass Stderr and Stdout correctly
		if resource.Stderr != nil {
			pklContent.WriteString(fmt.Sprintf("    stderr = #\"\"\"\n%s\n\"\"\"#\n", *resource.Stderr))
		} else {
			pklContent.WriteString("    stderr = \"\"\n")
		}
		if resource.Stdout != nil {
			pklContent.WriteString(fmt.Sprintf("    stdout = #\"\"\"\n%s\n\"\"\"#\n", *resource.Stdout))
		} else {
			pklContent.WriteString("    stdout = \"\"\n")
		}

		pklContent.WriteString(fmt.Sprintf("    file = \"%s\"\n", filePath))

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	// Evaluate the PKL file using EvalPkl
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Python.pkl\"", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}
