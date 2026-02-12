// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"syscall/js"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// Runtime holds the initialized WASM workflow runtime.
type Runtime struct {
	workflow *domain.Workflow
	engine   *executor.Engine
	envVars  map[string]string
}

// NewRuntime creates a new WASM runtime from workflow YAML.
func NewRuntime(
	workflowYAML string,
	envVars map[string]string,
	registry *executor.Registry,
) (*Runtime, error) {
	// Parse workflow from YAML string.
	workflow, err := parseWorkflowFromString(workflowYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Set environment variables if provided.
	if envVars != nil {
		for k, v := range envVars {
			os.Setenv(k, v) //nolint:errcheck // best-effort env setup in WASM
		}
	}

	// Create execution engine.
	logger := slog.Default()
	engine := executor.NewEngine(logger)
	engine.SetRegistry(registry)

	return &Runtime{
		workflow: workflow,
		engine:   engine,
		envVars:  envVars,
	}, nil
}

// Execute runs the workflow with the given input JSON.
// The input can be either a plain body object (e.g. {"prompt": "Hello"}) or a full
// request context from the fetch interceptor (with _kdeps_request, method, path, etc.).
func (r *Runtime) Execute(inputJSON string, callback *js.Value) (interface{}, error) {
	// Parse input into request context.
	var req interface{}
	if inputJSON != "" {
		var inputData map[string]interface{}
		if err := json.Unmarshal([]byte(inputJSON), &inputData); err != nil {
			return nil, fmt.Errorf("failed to parse input JSON: %w", err)
		}

		// Check if this is a full request context from the fetch interceptor.
		if _, ok := inputData["_kdeps_request"]; ok {
			rc := &executor.RequestContext{
				Method:  stringFromMap(inputData, "method", "POST"),
				Path:    stringFromMap(inputData, "path", "/"),
				Headers: stringMapFromMap(inputData, "headers"),
				Query:   stringMapFromMap(inputData, "query"),
			}
			if body, ok := inputData["body"].(map[string]interface{}); ok {
				rc.Body = body
			}
			req = rc
		} else {
			req = &executor.RequestContext{
				Method:  "POST",
				Path:    "/",
				Headers: make(map[string]string),
				Query:   make(map[string]string),
				Body:    inputData,
			}
		}
	}

	// Execute the workflow.
	result, err := r.engine.Execute(r.workflow, req)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Send result via callback if provided.
	if callback != nil {
		invokeCallback(callback, map[string]interface{}{
			"type": "result",
			"data": result,
		})
	}

	return result, nil
}

// parseWorkflowFromString parses a workflow from a YAML string.
func parseWorkflowFromString(yamlStr string) (*domain.Workflow, error) {
	var workflow domain.Workflow
	if err := yaml.Unmarshal([]byte(yamlStr), &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Initialize Resources if nil.
	if workflow.Resources == nil {
		workflow.Resources = make([]*domain.Resource, 0)
	}

	return &workflow, nil
}

// ValidateWorkflow validates a workflow YAML string and returns any errors.
func ValidateWorkflow(yamlStr string) []string {
	var errors []string

	// Parse YAML.
	workflow, err := parseWorkflowFromString(yamlStr)
	if err != nil {
		return []string{fmt.Sprintf("YAML parse error: %v", err)}
	}

	// Check required fields.
	if workflow.Metadata.Name == "" {
		errors = append(errors, "metadata.name is required")
	}
	if workflow.Metadata.TargetActionID == "" {
		errors = append(errors, "metadata.targetActionId is required")
	}

	// Check for WASM-incompatible resources.
	for _, res := range workflow.Resources {
		if res == nil {
			continue
		}
		errors = append(errors, validateWASMResource(res)...)
	}

	return errors
}

// stringFromMap extracts a string from a map with a default fallback.
func stringFromMap(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// stringMapFromMap extracts a map[string]string from a map[string]interface{}.
func stringMapFromMap(m map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]interface{}); ok {
			for k, val := range sub {
				if s, ok := val.(string); ok {
					result[k] = s
				}
			}
		}
	}
	return result
}

// validateWASMResource checks a resource for WASM compatibility.
func validateWASMResource(res *domain.Resource) []string {
	var errors []string
	actionID := res.Metadata.ActionID

	if res.Run.Exec != nil {
		errors = append(errors, fmt.Sprintf(
			"resource '%s': exec is not supported in WASM builds",
			actionID,
		))
	}

	if res.Run.Python != nil {
		errors = append(errors, fmt.Sprintf(
			"resource '%s': python is not supported in WASM builds",
			actionID,
		))
	}

	// Check for Ollama backend (not supported in WASM).
	if res.Run.Chat != nil {
		backend := res.Run.Chat.Backend
		if backend == "" || backend == "ollama" {
			errors = append(errors, fmt.Sprintf(
				"resource '%s': ollama backend is not supported in WASM; use an online LLM backend (openai, anthropic, google, etc.)",
				actionID,
			))
		}
	}

	return errors
}
