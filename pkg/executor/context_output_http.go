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
	"fmt"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// GetHTTPResponseBody retrieves HTTP response body from resource output.
func (ctx *ExecutionContext) GetHTTPResponseBody(actionID string) (interface{}, error) {
	kdeps_debug.Log("enter: GetHTTPResponseBody")
	output, err := ctx.resourceOutput(actionID)
	if err != nil {
		return nil, err
	}

	if outputMap, okMap := output.(map[string]interface{}); okMap {
		// Check for data field first (parsed JSON takes precedence)
		if data, okData := outputMap[contextFieldData]; okData {
			return data, nil
		}
		// Also check for body field (raw response)
		if body, okBody := outputMap[contextFieldBody].(string); okBody {
			return body, nil
		}
	}

	return "", nil
}

// GetHTTPResponseHeader retrieves HTTP response header from resource output.
func (ctx *ExecutionContext) GetHTTPResponseHeader(
	actionID, headerName string,
) (interface{}, error) {
	kdeps_debug.Log("enter: GetHTTPResponseHeader")
	output, err := ctx.resourceOutput(actionID)
	if err != nil {
		return nil, err
	}

	if outputMap, okMap := output.(map[string]interface{}); okMap {
		if headerValue, found := headerValueFromOutput(outputMap, headerName); found {
			return headerValue, nil
		}
	}

	// Return error when header not found (buildEvaluationEnvironment wrapper converts to nil)
	return nil, fmt.Errorf("header '%s' not found in response", headerName)
}
