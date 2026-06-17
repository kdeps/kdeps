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

func TestAsFloat64(t *testing.T) {
	t.Parallel()
	v, ok := asFloat64(float64(3.14))
	assert.True(t, ok)
	assert.InDelta(t, 3.14, v, 0.001)

	v, ok = asFloat64(int(5))
	assert.True(t, ok)
	assert.Equal(t, float64(5), v)

	v, ok = asFloat64("2.5")
	assert.True(t, ok)
	assert.Equal(t, 2.5, v)

	_, ok = asFloat64("notanumber")
	assert.False(t, ok)

	_, ok = asFloat64(true)
	assert.False(t, ok)
}

func TestParseWhereThreshold(t *testing.T) {
	t.Parallel()
	v, ok := parseWhereThreshold(float64(10))
	assert.True(t, ok)
	assert.Equal(t, float64(10), v)

	_, ok = parseWhereThreshold("bad")
	assert.False(t, ok)
}

func TestScoreFromMapValue(t *testing.T) {
	t.Parallel()
	v, ok := scoreFromMapValue(float64(7))
	assert.True(t, ok)
	assert.Equal(t, float64(7), v)

	_, ok = scoreFromMapValue("text")
	assert.False(t, ok)
}

func TestFilterWhereItems(t *testing.T) {
	t.Parallel()
	items := []interface{}{
		map[string]interface{}{"score": float64(9)},
		map[string]interface{}{"score": float64(3)},
		map[string]interface{}{"score": float64(7)},
		"not a map",
		map[string]interface{}{"other": float64(5)},
	}
	result := filterWhereItems(items, "score", 6.0)
	assert.Len(t, result, 2)
}

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
