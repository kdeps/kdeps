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

// AgentToolDef wraps a single agent workflow as a callable Tool.
// When called, it executes the agent's target action with the given params
// injected as query/body fields (accessible via get('key') inside resources).
func AgentToolDef(
	agentWorkflow *domain.Workflow,
	eng *executor.Engine,
) *Tool {
	name, desc := resolveAgentToolMetadata(agentWorkflow)
	return buildAgentTool(name, desc, agentWorkflow, eng)
}

// AgentToolDefWithName is like AgentToolDef but uses an explicit name and description.
// Used for agencies where the tool name is the agency name, not the entry-point workflow name.
func AgentToolDefWithName(name, desc string, workflow *domain.Workflow, eng *executor.Engine) *Tool {
	return buildAgentTool(name, desc, workflow, eng)
}

func resolveAgentToolMetadata(agentWorkflow *domain.Workflow) (string, string) {
	name := agentWorkflow.Metadata.Name
	if name == "" {
		name = "agent"
	}
	desc := agentWorkflow.Metadata.Description
	if desc == "" {
		desc = fmt.Sprintf("Agent: %s v%s", name, agentWorkflow.Metadata.Version)
	}
	return name, desc
}

func buildAgentTool(name, desc string, workflow *domain.Workflow, eng *executor.Engine) *Tool {
	return &Tool{
		Name:        name,
		Description: desc,
		Parameters:  agentToolParameters(),
		Execute: func(args map[string]interface{}) (string, error) {
			return executeAgentTool(eng, workflow, args)
		},
	}
}

func agentToolParameters() map[string]domain.ToolParam {
	return map[string]domain.ToolParam{
		toolsFieldInput: {
			Type:        "string",
			Description: "Input message or data forwarded to the agent.",
			Required:    true,
		},
	}
}

// executeAgentTool runs an agent workflow with the provided args as request params.
func executeAgentTool(
	eng *executor.Engine,
	agentWorkflow *domain.Workflow,
	args map[string]interface{},
) (string, error) {
	reqCtx := buildAgentRequestContext(args)
	result, err := eng.Execute(agentWorkflow, reqCtx)
	if err != nil {
		return "", err
	}
	return marshalResult(result), nil
}

func buildAgentRequestContext(args map[string]interface{}) *executor.RequestContext {
	query := make(map[string]string, len(args))
	body := make(map[string]interface{}, len(args))
	for k, v := range args {
		query[k] = fmt.Sprintf("%v", v)
		body[k] = v
	}
	return &executor.RequestContext{
		Method: "GET",
		Query:  query,
		Body:   body,
	}
}

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
