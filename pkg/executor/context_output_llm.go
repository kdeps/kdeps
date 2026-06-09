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

package executor

import (
	"errors"
	"fmt"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// GetLLMResponse retrieves LLM response text from resource output.
func (ctx *ExecutionContext) GetLLMResponse(actionID string) (interface{}, error) {
	kdeps_debug.Log("enter: GetLLMResponse")
	output, ok := ctx.Outputs[actionID]
	if !ok {
		return nil, fmt.Errorf("output for resource '%s' not found", actionID)
	}

	// LLM output is typically a string (response text)
	if responseStr, okStr := output.(string); okStr {
		return responseStr, nil
	}

	// If it's a map (e.g., JSON response), try to extract response or data field
	if outputMap, okMap := output.(map[string]interface{}); okMap {
		return extractLLMResponseFromMap(outputMap), nil
	}

	return output, nil
}

// GetLLMPrompt retrieves LLM prompt text (not stored in output, would need to be from resource config).
func (ctx *ExecutionContext) GetLLMPrompt(_ string) (interface{}, error) {
	kdeps_debug.Log("enter: GetLLMPrompt")
	// Prompt is not stored in output, would need access to resource config
	// For now, return nil as this requires additional context
	return nil, errors.New("prompt not available from output (requires resource config access)")
}
