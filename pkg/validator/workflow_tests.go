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

package validator

import (
	"fmt"
	"strings"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// ValidateTestCases validates self-test case definitions.
func ValidateTestCases(tests []domain.TestCase) error {
	kdeps_debug.Log("enter: ValidateTestCases")
	for i, tc := range tests {
		if tc.Name == "" {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				fmt.Sprintf("tests[%d]: name is required", i),
				nil,
			)
		}
		if tc.Request.Path == "" {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				fmt.Sprintf("test %q: request.path is required", tc.Name),
				nil,
			)
		}
		method := strings.ToUpper(tc.Request.Method)
		if !domain.IsValidHTTPMethodAllowEmpty(method) {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				fmt.Sprintf(
					"test %q: invalid method %q (use %s)",
					tc.Name, tc.Request.Method, domain.StandardHTTPMethodsDisplay(),
				),
				nil,
			)
		}
		if tc.Assert.Status != 0 && (tc.Assert.Status < 100 || tc.Assert.Status > 599) {
			return domain.NewError(
				domain.ErrCodeInvalidWorkflow,
				fmt.Sprintf("test %q: assert.status %d out of range (100-599)", tc.Name, tc.Assert.Status),
				nil,
			)
		}
	}
	return nil
}
