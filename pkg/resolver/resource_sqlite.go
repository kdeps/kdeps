package resolver

import (
	"fmt"
	"time"

	"github.com/kdeps/kdeps/pkg/resource"
	"github.com/kdeps/kdeps/pkg/utils"
	pklData "github.com/kdeps/schema/gen/data"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
)

// StoreExecResource stores an execution resource using SQLite instead of PKL files
func (dr *DependencyResolver) StoreExecResource(resourceID string, newExec *pklExec.ResourceExec, hasItems bool) error {
	if dr.ResourceReader == nil {
		return fmt.Errorf("resource reader not initialized")
	}

	// Convert PKL exec resource to our SQLite resource format
	execRes := &resource.ExecResource{
		BaseResource: resource.BaseResource{
			ID:        resourceID,
			RequestID: dr.RequestID,
			Type:      resource.TypeExec,
			Timestamp: time.Now(),
		},
		Command: newExec.Command,
		Stdout:  utils.SafeDerefString(newExec.Stdout),
		Stderr:  utils.SafeDerefString(newExec.Stderr),
		File:    utils.SafeDerefString(newExec.File),
	}

	// Set timeout duration if provided
	if newExec.TimeoutDuration != nil {
		timeout := newExec.TimeoutDuration.GoDuration()
		execRes.TimeoutDuration = &timeout
	}

	// Convert environment variables
	if newExec.Env != nil {
		execRes.Env = *newExec.Env
	}

	// Store using injectable function
	return StoreExecResourceFunc(dr.ResourceReader, execRes)
}

// StorePythonResource stores a Python resource using SQLite instead of PKL files
func (dr *DependencyResolver) StorePythonResource(resourceID string, newPython *pklPython.ResourcePython, hasItems bool) error {
	if dr.ResourceReader == nil {
		return fmt.Errorf("resource reader not initialized")
	}

	// Convert PKL Python resource to our SQLite resource format
	pythonRes := &resource.PythonResource{
		BaseResource: resource.BaseResource{
			ID:        resourceID,
			RequestID: dr.RequestID,
			Type:      resource.TypePython,
			Timestamp: time.Now(),
		},
		Script: newPython.Script,
		Stdout: utils.SafeDerefString(newPython.Stdout),
		Stderr: utils.SafeDerefString(newPython.Stderr),
		File:   utils.SafeDerefString(newPython.File),
	}

	// Set timeout duration if provided
	if newPython.TimeoutDuration != nil {
		timeout := newPython.TimeoutDuration.GoDuration()
		pythonRes.TimeoutDuration = &timeout
	}

	// Convert environment variables
	if newPython.Env != nil {
		pythonRes.Env = *newPython.Env
	}

	// Store using injectable function
	return StorePythonResourceFunc(dr.ResourceReader, pythonRes)
}

// StoreHTTPResource stores an HTTP resource using SQLite instead of PKL files
func (dr *DependencyResolver) StoreHTTPResource(resourceID string, client *pklHTTP.ResourceHTTPClient, hasItems bool) error {
	if dr.ResourceReader == nil {
		return fmt.Errorf("resource reader not initialized")
	}

	// Convert PKL HTTP resource to our SQLite resource format
	httpRes := &resource.HTTPResource{
		BaseResource: resource.BaseResource{
			ID:        resourceID,
			RequestID: dr.RequestID,
			Type:      resource.TypeHTTP,
			Timestamp: time.Now(),
		},
		Method: client.Method,
		URL:    client.Url,
		File:   utils.SafeDerefString(client.File),
	}

	// Set timeout duration if provided
	if client.TimeoutDuration != nil {
		timeout := client.TimeoutDuration.GoDuration()
		httpRes.TimeoutDuration = &timeout
	}

	// Convert headers
	if client.Headers != nil {
		httpRes.Headers = *client.Headers
	}

	// Convert data
	if client.Data != nil {
		httpRes.Data = *client.Data
	}

	// Convert params
	if client.Params != nil {
		httpRes.Params = *client.Params
	}

	// Convert response
	if client.Response != nil {
		httpRes.Response = &resource.HTTPResponse{
			Body: utils.SafeDerefString(client.Response.Body),
		}
		if client.Response.Headers != nil {
			httpRes.Response.Headers = *client.Response.Headers
		}
	}

	// Store using injectable function
	return StoreHTTPResourceFunc(dr.ResourceReader, httpRes)
}

// StoreLLMResource stores an LLM/Chat resource using SQLite instead of PKL files
func (dr *DependencyResolver) StoreLLMResource(resourceID string, newChat *pklLLM.ResourceChat, hasItems bool) error {
	if dr.ResourceReader == nil {
		return fmt.Errorf("resource reader not initialized")
	}

	// Convert PKL LLM resource to our SQLite resource format
	llmRes := &resource.LLMResource{
		BaseResource: resource.BaseResource{
			ID:        resourceID,
			RequestID: dr.RequestID,
			Type:      resource.TypeLLM,
			Timestamp: time.Now(),
		},
		Model:        newChat.Model, // Model is already a string, not a pointer
		Prompt:       utils.SafeDerefString(newChat.Prompt),
		Role:         utils.SafeDerefString(newChat.Role),
		JSONResponse: utils.SafeDerefBool(newChat.JSONResponse),
		Response:     utils.SafeDerefString(newChat.Response),
		File:         utils.SafeDerefString(newChat.File),
	}

	// Handle Scenario field - it's a *[]*pklLLM.MultiChat, not a *string
	if newChat.Scenario != nil && len(*newChat.Scenario) > 0 {
		// For now, just store a simple representation of the scenario
		// In a full implementation, you'd want to serialize the entire scenario structure
		llmRes.Scenario = fmt.Sprintf("scenario with %d entries", len(*newChat.Scenario))
	}

	// Set timeout duration if provided
	if newChat.TimeoutDuration != nil {
		timeout := newChat.TimeoutDuration.GoDuration()
		llmRes.TimeoutDuration = &timeout
	}

	// Convert JSON response keys
	if newChat.JSONResponseKeys != nil {
		llmRes.JSONResponseKeys = *newChat.JSONResponseKeys
	}

	// Convert files - it's a *[]string, not a *map[string]string
	if newChat.Files != nil {
		// Convert []string to map[string]string for storage
		// For now, use index as key
		fileMap := make(map[string]string)
		for i, file := range *newChat.Files {
			fileMap[fmt.Sprintf("file_%d", i)] = file
		}
		llmRes.Files = fileMap
	}

	// Convert tools
	if newChat.Tools != nil {
		for _, tool := range *newChat.Tools {
			llmTool := resource.LLMTool{
				Name:        utils.SafeDerefString(tool.Name), // tool.Name is *string
				Description: utils.SafeDerefString(tool.Description),
			}
			// tool.Parameters is *map[string]*pklLLM.ToolProperties, we need map[string]interface{}
			if tool.Parameters != nil {
				params := make(map[string]interface{})
				for key, toolProp := range *tool.Parameters {
					// Convert ToolProperties to interface{}
					if toolProp != nil {
						paramMap := make(map[string]interface{})
						if toolProp.Type != nil {
							paramMap["type"] = *toolProp.Type
						}
						if toolProp.Description != nil {
							paramMap["description"] = *toolProp.Description
						}
						params[key] = paramMap
					}
				}
				llmTool.Parameters = params
			}
			llmRes.Tools = append(llmRes.Tools, llmTool)
		}
	}

	// Store using injectable function
	return StoreLLMResourceFunc(dr.ResourceReader, llmRes)
}

// StoreDataResource stores a data resource using SQLite instead of PKL files
func (dr *DependencyResolver) StoreDataResource(resourceID string, newData *pklData.DataImpl, hasItems bool) error {
	if dr.ResourceReader == nil {
		return fmt.Errorf("resource reader not initialized")
	}

	// Convert PKL data resource to our SQLite resource format
	dataRes := &resource.DataResource{
		BaseResource: resource.BaseResource{
			ID:        resourceID,
			RequestID: dr.RequestID,
			Type:      resource.TypeData,
			Timestamp: time.Now(),
		},
	}

	// Convert files map
	if newData.Files != nil {
		dataRes.Files = *newData.Files
	}

	// Store using injectable function
	return StoreDataResourceFunc(dr.ResourceReader, dataRes)
}
