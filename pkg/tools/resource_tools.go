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

package tools

import (
	"encoding/json"
	"fmt"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// ResourceToolDefs wraps each resource in workflow as a callable Tool.
// The tool name is the resource's actionId; arguments are forwarded as query
// params and body fields so the resource can read them via get('key').
func ResourceToolDefs(workflow *domain.Workflow, eng *executor.Engine) []*Tool {
	tools := make([]*Tool, 0, len(workflow.Resources))
	for _, resource := range workflow.Resources {
		r := resource
		tools = append(tools, &Tool{
			Name:        r.ActionID,
			Description: resourceDescription(r),
			Parameters:  inferResourceParams(r),
			Execute: func(args map[string]interface{}) (string, error) {
				return executeResourceTool(eng, workflow, r, args)
			},
		})
	}
	return tools
}

// executeResourceTool runs a single resource via the engine with the given
// tool call arguments injected as query params and body fields.
func executeResourceTool(
	eng *executor.Engine,
	workflow *domain.Workflow,
	resource *domain.Resource,
	args map[string]interface{},
) (string, error) {
	// Build a minimal single-resource workflow targeting this resource.
	single := &domain.Workflow{
		APIVersion: workflow.APIVersion,
		Kind:       workflow.Kind,
		Metadata: domain.WorkflowMetadata{
			Name:           workflow.Metadata.Name,
			Version:        workflow.Metadata.Version,
			TargetActionID: resource.ActionID,
		},
		Settings:   workflow.Settings,
		Components: workflow.Components,
		Resources:  []*domain.Resource{resource},
	}

	// Inject args as query params (accessible via get('key')) and body.
	query := make(map[string]string, len(args))
	body := make(map[string]interface{}, len(args))
	for k, v := range args {
		query[k] = fmt.Sprintf("%v", v)
		body[k] = v
	}
	reqCtx := &executor.RequestContext{
		Method: "GET",
		Query:  query,
		Body:   body,
	}

	result, err := eng.Execute(single, reqCtx)
	if err != nil {
		return "", err
	}
	return marshalResult(result), nil
}

// resourceDescription returns a description string for a resource tool.
func resourceDescription(r *domain.Resource) string {
	if r.Name != "" {
		return r.Name
	}
	return "Resource " + r.ActionID
}

// inferResourceParams returns a minimal parameter schema for a resource tool.
// We expose a single generic 'input' param since resources read arbitrary keys
// via get(). Callers that know a resource's interface can override this.
func inferResourceParams(r *domain.Resource) map[string]domain.ToolParam {
	_ = r
	return map[string]domain.ToolParam{
		"input": {
			Type:        "string",
			Description: "Input data for the resource.",
			Required:    false,
		},
	}
}

// marshalResult converts an arbitrary result to a JSON string.
func marshalResult(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
