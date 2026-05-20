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
)

// ComponentToolDefs converts a slice of loaded components into Tool metadata entries.
// Execute functions are not set; callers inject them after wiring component execution.
func ComponentToolDefs(components []*domain.Component) []*Tool {
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
		tools = append(tools, &Tool{
			Name:        comp.Metadata.Name,
			Description: comp.Metadata.Description,
			Parameters:  params,
		})
	}
	return tools
}
