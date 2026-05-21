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
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// ComponentToolDefs wraps each component as a callable Tool.
// The engine executes the component via a synthetic component resource.
func ComponentToolDefs(
	components []*domain.Component,
	workflow *domain.Workflow,
	eng *executor.Engine,
) []*Tool {
	tools := make([]*Tool, 0, len(components))
	for _, comp := range components {
		if comp == nil {
			continue
		}
		params := map[string]domain.ToolParam{}
		if comp.Interface != nil {
			for _, input := range comp.Interface.Inputs {
				params[input.Name] = domain.ToolParam{
					Type:        input.Type,
					Description: input.Description,
					Required:    input.Required,
				}
			}
		}
		c := comp
		t := &Tool{
			Name:        c.Metadata.Name,
			Description: c.Metadata.Description,
			Parameters:  params,
		}
		if eng != nil && workflow != nil {
			t.Execute = func(args map[string]interface{}) (string, error) {
				return executeComponentTool(eng, workflow, c, args)
			}
		}
		tools = append(tools, t)
	}
	return tools
}

// executeComponentTool runs a component via a synthetic component resource.
func executeComponentTool(
	eng *executor.Engine,
	workflow *domain.Workflow,
	comp *domain.Component,
	args map[string]interface{},
) (string, error) {
	with := make(map[string]interface{}, len(args))
	for k, v := range args {
		with[k] = v
	}
	actionID := "agent_component_" + comp.Metadata.Name
	syntheticResource := &domain.Resource{
		ActionID: actionID,
		Name:     comp.Metadata.Name,
		Component: &domain.ComponentCallConfig{
			Name: comp.Metadata.Name,
			With: with,
		},
	}
	single := &domain.Workflow{
		APIVersion: workflow.APIVersion,
		Kind:       workflow.Kind,
		Metadata: domain.WorkflowMetadata{
			Name:           workflow.Metadata.Name,
			Version:        workflow.Metadata.Version,
			TargetActionID: actionID,
		},
		Settings:   workflow.Settings,
		Components: workflow.Components,
		Resources:  []*domain.Resource{syntheticResource},
	}
	result, err := eng.Execute(single, nil)
	if err != nil {
		return "", err
	}
	return marshalResult(result), nil
}
