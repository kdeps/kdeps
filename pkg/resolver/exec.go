package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklExec "github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleExec(actionID string, execBlock *pklExec.ResourceExec) error {
	// Decode Command if it is Base64-encoded
	if utf8.ValidString(execBlock.Command) && utils.IsBase64Encoded(execBlock.Command) {
		decodedCommand, err := utils.DecodeBase64String(execBlock.Command)
		if err == nil {
			execBlock.Command = decodedCommand
		}
	}

	// Decode Stderr if it is Base64-encoded
	if execBlock.Stderr != nil && utf8.ValidString(*execBlock.Stderr) && utils.IsBase64Encoded(*execBlock.Stderr) {
		decodedStderr, err := utils.DecodeBase64String(*execBlock.Stderr)
		if err == nil {
			execBlock.Stderr = &decodedStderr
		}
	}

	// Decode Stdout if it is Base64-encoded
	if execBlock.Stdout != nil && utf8.ValidString(*execBlock.Stdout) && utils.IsBase64Encoded(*execBlock.Stdout) {
		decodedStdout, err := utils.DecodeBase64String(*execBlock.Stdout)
		if err == nil {
			execBlock.Stdout = &decodedStdout
		}
	}

	// Decode Env map keys and values if they are Base64-encoded
	if execBlock.Env != nil {
		for key, value := range *execBlock.Env {
			stringKey := key
			decodedValue := value

			// Decode value if it is Base64-encoded
			if utf8.ValidString(value) && utils.IsBase64Encoded(value) {
				decodedValue, _ = utils.DecodeBase64String(value)
			}

			// Update the map with the decoded values
			(*execBlock.Env)[stringKey] = decodedValue
		}
	}

	go func() error {
		err := dr.processExecBlock(actionID, execBlock)
		if err != nil {
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) WriteStdoutToFile(resourceID string, stdoutEncoded *string) (string, error) {
	// Convert resourceID to be filename friendly
	resourceIDFile := utils.ConvertToFilenameFriendly(resourceID)
	// Define the file path using the FilesDir and resource ID
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	// Ensure the Stdout is not nil
	if stdoutEncoded != nil {
		// Prepare the content to write
		var content string
		if utils.IsBase64Encoded(*stdoutEncoded) {
			// Decode the Base64-encoded Stdout string
			decodedStdout, err := utils.DecodeBase64String(*stdoutEncoded)
			if err != nil {
				return "", fmt.Errorf("failed to decode Base64 string for resource ID: %s: %w", resourceID, err)
			}
			content = decodedStdout
		} else {
			// Use the Stdout content as-is if not Base64-encoded
			content = *stdoutEncoded
		}

		// Write the content to the file
		err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write Stdout to file for resource ID: %s: %w", resourceID, err)
		}
	} else {
		return "", nil
	}

	return outputFilePath, nil
}

func (dr *DependencyResolver) processExecBlock(actionID string, execBlock *pklExec.ResourceExec) error {
	var env []string
	if execBlock.Env != nil {
		for key, value := range *execBlock.Env {
			// Append the environment variable in the desired format
			env = append(env, fmt.Sprintf(`%s=%s`, key, value))
		}
	}

	// Log the command and environment variables
	dr.Logger.Info("executing command", "command", execBlock.Command, "env", env)

	cmd := execute.ExecTask{
		Command:     execBlock.Command,
		Shell:       true,
		Env:         env,
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(dr.Context)
	if err != nil {
		return err
	}

	execBlock.Stdout = &result.Stdout
	execBlock.Stderr = &result.Stderr

	if err := dr.AppendExecEntry(actionID, execBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) AppendExecEntry(resourceID string, newExec *pklExec.ResourceExec) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklExec.LoadFromPath(dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.GetResources() // Dereference the pointer to get the map

	// Check and Base64 encode Command, Stderr, Stdout if not already encoded
	encodedCommand := newExec.Command
	if !utils.IsBase64Encoded(newExec.Command) {
		encodedCommand = utils.EncodeBase64String(newExec.Command)
	}

	var filePath, encodedStderr, encodedStdout string
	if newExec.Stderr != nil {
		if !utils.IsBase64Encoded(*newExec.Stderr) {
			encodedStderr = utils.EncodeBase64String(*newExec.Stderr)
		} else {
			encodedStderr = *newExec.Stderr
		}
	}
	if newExec.Stdout != nil {
		filePath, err = dr.WriteStdoutToFile(resourceID, newExec.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write Stdout to file: %w", err)
		}
		newExec.File = &filePath

		if !utils.IsBase64Encoded(*newExec.Stdout) {
			encodedStdout = utils.EncodeBase64String(*newExec.Stdout)
		} else {
			encodedStdout = *newExec.Stdout
		}
	}

	// Base64 encode the Env map (keys and values)
	var encodedEnv *map[string]string
	if newExec.Env != nil {
		encodedEnvMap := make(map[string]string)
		for key, value := range *newExec.Env {
			stringKey := key
			encodedValue := value

			if !utils.IsBase64Encoded(value) {
				encodedValue = utils.EncodeBase64String(value)
			}
			encodedEnvMap[stringKey] = encodedValue
		}
		encodedEnv = &encodedEnvMap
	}

	// Create or update the ResourceExec entry
	existingResources[resourceID] = &pklExec.ResourceExec{
		Env:       encodedEnv,
		Command:   encodedCommand,
		Stderr:    &encodedStderr,
		Stdout:    &encodedStdout,
		File:      &filePath,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    command = \"%s\"\n", resource.Command))
		pklContent.WriteString(fmt.Sprintf("    timeoutSeconds = %d\n", resource.TimeoutSeconds))
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
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"", schema.SchemaVersion(dr.Context)), dr.Logger)
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
