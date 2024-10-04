package resolver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	pklExec "github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleExec(actionId string, execBlock *pklExec.ResourceExec) error {
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
	pklPath := filepath.Join(dr.ActionDir, "exec/exec_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.Resource // Dereference the pointer to get the map

	// Create or update the ResourceChat entry
	existingResources[resourceId] = &pklExec.ResourceExec{
		Env:       newExec.Env, // Add Env field
		Command:   newExec.Command,
		Stderr:    newExec.Stderr,
		Stdout:    newExec.Stdout,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString("amends \"package://schema.kdeps.com/core@0.1.0#/Exec.pkl\"\n\n")
	pklContent.WriteString("resource {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    command = \"\"\"\n%s\n\"\"\"\n", resource.Command))
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
			pklContent.WriteString("    env {}\n") // Handle nil case for Env
		}

		// Dereference to pass Stderr and Stdout correctly
		if resource.Stderr != nil {
			pklContent.WriteString(fmt.Sprintf("    stderr = \"\"\"\n%s\n\"\"\"\n", *resource.Stderr))
		} else {
			pklContent.WriteString("    stderr = \"\"\n") // Handle nil case
		}
		if resource.Stdout != nil {
			pklContent.WriteString(fmt.Sprintf("    stdout = \"\"\"\n%s\n\"\"\"\n", *resource.Stdout))
		} else {
			pklContent.WriteString("    stdout = \"\"\n") // Handle nil case
		}

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	return nil
}
