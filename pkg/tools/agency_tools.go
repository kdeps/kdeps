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
	"maps"
	"regexp"
	"slices"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

// AgentToolParameters is the input schema an agent or workflow tool takes:
// one parameter per input('<key>') the workflow's resources read, so the model
// sends the keys the workflow uses; a workflow that reads none takes one
// "input" message.
func AgentToolParameters(workflow *domain.Workflow) map[string]domain.ToolParam {
	keys := workflowInputKeys(workflow)
	if len(keys) == 0 {
		return map[string]domain.ToolParam{
			toolsFieldInput: {
				Type:        "string",
				Description: "Input message or data forwarded to the agent.",
				Required:    true,
			},
		}
	}
	params := make(map[string]domain.ToolParam, len(keys))
	for _, k := range keys {
		params[k] = domain.ToolParam{
			Type:        "string",
			Description: fmt.Sprintf("The workflow's %q input (read as input('%s')).", k, k),
			Required:    len(keys) == 1,
		}
	}
	return params
}

// inputKeyRe matches an input('<key>') or input("<key>") reference.
var inputKeyRe = regexp.MustCompile(`input\(\s*\\?['"]([A-Za-z0-9_.-]+)\\?['"]`)

// workflowInputKeys lists the keys the workflow's resources read with input(),
// sorted.
func workflowInputKeys(workflow *domain.Workflow) []string {
	if workflow == nil || len(workflow.Resources) == 0 {
		return nil
	}
	// Only scanned for input() references; field names do not matter.
	data, err := json.Marshal(workflow.Resources) //nolint:musttag // see above
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, m := range inputKeyRe.FindAllStringSubmatch(string(data), -1) {
		seen[m[1]] = true
	}
	return slices.Sorted(maps.Keys(seen))
}
