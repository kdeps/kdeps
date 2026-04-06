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

// ClassifyError maps an error to the closest FailureClass.
// The matching is intentionally broad — it looks for substrings that
// commonly appear in kdeps error messages for each failure domain.
func ClassifyError(err error) FailureClass {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())

	switch {
	case containsAny(msg, "timeout", "deadline exceeded", "context deadline"):
		return FailureClassTimeout
	case containsAny(msg, "provider", "llm", "model", "openai", "anthropic", "ollama", "gemini", "groq"):
		return FailureClassProvider
	case containsAny(msg, "validation", "invalid input", "required field", "schema"):
		return FailureClassValidation
	case containsAny(msg, "preflight", "unauthorized", "forbidden", "authentication", "authorization"):
		return FailureClassPreflight
	case containsAny(msg, "compile", "syntax", "parse error", "expression compilation"):
		return FailureClassCompile
	case containsAny(msg, "connection refused", "no route to host", "dial tcp", "network", "dns"):
		return FailureClassInfra
	default:
		return FailureClassToolRuntime
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
