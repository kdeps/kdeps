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

package events

import "strings"

type failureClassRule struct {
	class      FailureClass
	substrings []string
}

func failureClassRules() []failureClassRule {
	return []failureClassRule{
		{
			class:      FailureClassTimeout,
			substrings: []string{"timeout", "deadline exceeded", "context deadline"},
		},
		{
			class: FailureClassProvider,
			substrings: []string{
				"provider", "llm", "model", "openai", "anthropic", "ollama", "gemini", "groq",
			},
		},
		{
			class:      FailureClassValidation,
			substrings: []string{"validation", "invalid input", "required field", "schema"},
		},
		{
			class: FailureClassPreflight,
			substrings: []string{
				"preflight", "unauthorized", "forbidden", "authentication", "authorization",
			},
		},
		{
			class:      FailureClassCompile,
			substrings: []string{"compile", "syntax", "parse error", "expression compilation"},
		},
		{
			class: FailureClassInfra,
			substrings: []string{
				"connection refused", "no route to host", "dial tcp", "network", "dns",
			},
		},
	}
}

// ClassifyError maps an error to the closest FailureClass.
// The matching is intentionally broad — it looks for substrings that
// commonly appear in kdeps error messages for each failure domain.
func ClassifyError(err error) FailureClass {
	if err == nil {
		return ""
	}
	return classifyErrorMessage(strings.ToLower(err.Error()))
}

func classifyErrorMessage(msg string) FailureClass {
	for _, rule := range failureClassRules() {
		if containsAny(msg, rule.substrings...) {
			return rule.class
		}
	}
	return FailureClassToolRuntime
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
