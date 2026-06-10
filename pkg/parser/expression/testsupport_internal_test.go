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
	"errors"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestControlFlowListOperations tests filter, map, all, any as loop alternatives.
func TestControlFlowListOperations(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]interface{}
		validate func(*testing.T, interface{})
	}{
		{
			name: "filter_array",
			expr: "{{filter(users, .age >= 18)}}",
			env: map[string]interface{}{
				"users": []map[string]interface{}{
					{"name": "Alice", "age": 20},
					{"name": "Bob", "age": 15},
					{"name": "Charlie", "age": 30},
				},
			},
			validate: validateFilterResult,
		},
		{
			name: "map_array",
			expr: "{{map(users, .name)}}",
			env: map[string]interface{}{
				"users": []map[string]interface{}{
					{"name": "Alice", "age": 20},
					{"name": "Bob", "age": 15},
				},
			},
			validate: validateMapResult,
		},
		{
			name: "all_predicate_true",
			expr: "{{all(items, .valid)}}",
			env: map[string]interface{}{
				"items": []map[string]interface{}{
					{"valid": true},
					{"valid": true},
				},
			},
			validate: validateBoolResult(true),
		},
		{
			name: "all_predicate_false",
			expr: "{{all(items, .valid)}}",
			env: map[string]interface{}{
				"items": []map[string]interface{}{
					{"valid": true},
					{"valid": false},
				},
			},
			validate: validateBoolResult(false),
		},
		{
			name: "any_predicate_true",
			expr: "{{any(items, .active)}}",
			env: map[string]interface{}{
				"items": []map[string]interface{}{
					{"active": false},
					{"active": true},
				},
			},
			validate: validateBoolResult(true),
		},
		{
			name: "any_predicate_false",
			expr: "{{any(items, .active)}}",
			env: map[string]interface{}{
				"items": []map[string]interface{}{
					{"active": false},
					{"active": false},
				},
			},
			validate: validateBoolResult(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateExpression(t, tt.expr, tt.env)
			tt.validate(t, result)
		})
	}
}

func evaluateExpression(t *testing.T, expr string, env map[string]interface{}) interface{} {
	t.Helper()
	api := &domain.UnifiedAPI{}
	eval := NewEvaluator(api)
	parser := NewParser()

	exprObj, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	result, err := eval.Evaluate(exprObj, env)
	if err != nil {
		t.Fatalf("evaluation error: %v", err)
	}
	return result
}

func validateFilterResult(t *testing.T, result interface{}) {
	t.Helper()
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 items, got %d", len(arr))
	}
}

func validateMapResult(t *testing.T, result interface{}) {
	t.Helper()
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 items, got %d", len(arr))
	}
	if arr[0] != "Alice" {
		t.Errorf("expected 'Alice', got %v", arr[0])
	}
}

func validateBoolResult(expected bool) func(*testing.T, interface{}) {
	return func(t *testing.T, result interface{}) {
		t.Helper()
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	}
}

// createMockAPIForCoverage returns a minimal API for coverage tests that need
// non-nil api to register helper functions (json, safe, fromJSON, where, etc.)
func createMockAPIForCoverage() *domain.UnifiedAPI {
	return &domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("not found")
		},
	}
}
