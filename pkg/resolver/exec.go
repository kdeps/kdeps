package resolver

import (
	"context"
	"fmt"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/schema"
	"kdeps/pkg/utils"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexellis/go-execute/v2"
	pklExec "github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleExec(actionId string, execBlock *pklExec.ResourceExec) error {
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

			// // Decode key if it is Base64-encoded
			// if utf8.ValidString(key) && utils.IsBase64Encoded(key) {
			//	decodedKey, _ = utils.DecodeBase64String(key)
			// }

			// Decode value if it is Base64-encoded
			if utf8.ValidString(value) && utils.IsBase64Encoded(value) {
				decodedValue, _ = utils.DecodeBase64String(value)
			}

			// Update the map with the decoded values
			(*execBlock.Env)[stringKey] = decodedValue
		}
	}

	go func() error {
		err := dr.processExecBlock(actionId, execBlock)
		if err != nil {
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) processExecBlock(actionId string, execBlock *pklExec.ResourceExec) error {
	var env []string
	if execBlock.Env != nil {
		for key, value := range *execBlock.Env {
			env = append(env, fmt.Sprintf("%s=\"%s\"", key, value))
		}
	}

	cmd := execute.ExecTask{
		Command:     execBlock.Command,
		Shell:       true,
		Env:         env,
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(context.Background())
	if err != nil {
		return err
	}

	execBlock.Stdout = &result.Stdout
	execBlock.Stderr = &result.Stderr

	if err := dr.AppendExecEntry(actionId, execBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) AppendExecEntry(resourceId string, newExec *pklExec.ResourceExec) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestId+"__exec_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
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

	var encodedStderr, encodedStdout string
	if newExec.Stderr != nil {
		if !utils.IsBase64Encoded(*newExec.Stderr) {
			encodedStderr = utils.EncodeBase64String(*newExec.Stderr)
		} else {
			encodedStderr = *newExec.Stderr
		}
	}
	if newExec.Stdout != nil {
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
	existingResources[resourceId] = &pklExec.ResourceExec{
		Env:       encodedEnv,
		Command:   encodedCommand,
		Stderr:    &encodedStderr,
		Stdout:    &encodedStdout,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"\n\n", schema.SchemaVersion))
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

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	// Evaluate the PKL file using EvalPkl
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"", schema.SchemaVersion), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}
