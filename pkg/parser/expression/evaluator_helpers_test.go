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

package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeEnvObject_NewKey(t *testing.T) {
	evalEnv := map[string]interface{}{}
	src := map[string]interface{}{
		"item": map[string]interface{}{"k": "v"},
	}
	mergeEnvObject(evalEnv, src, "item")
	assert.Equal(t, "v", evalEnv["item"].(map[string]interface{})["k"])
}

func TestEvaluateCondition_SliceAndUnsupported(t *testing.T) {
	e := NewEvaluator(nil)

	// Slice result from expression: [1, 2, 3] is a slice/array type
	result, err := e.EvaluateCondition("[1, 2, 3]", nil)
	if err != nil {
		t.Fatalf("EvaluateCondition failed: %v", err)
	}
	if result != true {
		t.Errorf("expected true for non-empty slice, got %v", result)
	}

	// Unsupported type: map is not slice/array and not handled by earlier cases
	_, err = e.EvaluateCondition(`{"a": 1}`, nil)
	if err == nil {
		t.Error("expected error for unsupported type (map) in condition")
	}
}
