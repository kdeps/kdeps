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

package events_test

import (
	"errors"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/events"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		err  string
		want events.FailureClass
	}{
		{"context deadline exceeded", events.FailureClassTimeout},
		{"timeout: 60s expired", events.FailureClassTimeout},
		{"llm provider returned 503", events.FailureClassProvider},
		{"openai API error", events.FailureClassProvider},
		{"validation: required field missing", events.FailureClassValidation},
		{"invalid input: must be non-empty", events.FailureClassValidation},
		{"preflight check failed: unauthorized", events.FailureClassPreflight},
		{"expression compilation failed: syntax error", events.FailureClassCompile},
		{"connection refused: dial tcp 127.0.0.1", events.FailureClassInfra},
		{"unknown resource execution failure", events.FailureClassToolRuntime},
	}
	for _, tc := range cases {
		got := events.ClassifyError(errors.New(tc.err))
		if got != tc.want {
			t.Errorf("ClassifyError(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestClassifyNilError(t *testing.T) {
	if got := events.ClassifyError(nil); got != "" {
		t.Errorf("ClassifyError(nil) should return empty string, got %q", got)
	}
}
