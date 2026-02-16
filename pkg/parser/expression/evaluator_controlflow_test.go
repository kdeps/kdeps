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

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestControlFlowTernary tests if-else using ternary operator.
func TestControlFlowTernary(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]interface{}
		expected interface{}
	}{
		{
			name: "simple_ternary",
			expr: "{{age >= 18 ? 'adult' : 'child'}}",
			env: map[string]interface{}{
				"age": 20,
			},
			expected: "adult",
		},
		{
			name: "ternary_false_branch",
			expr: "{{age >= 18 ? 'adult' : 'child'}}",
			env: map[string]interface{}{
				"age": 15,
			},
			expected: "child",
		},
		{
			name: "nested_ternary",
			expr: "{{age < 13 ? 'child' : (age < 20 ? 'teen' : 'adult')}}",
			env: map[string]interface{}{
				"age": 16,
			},
			expected: "teen",
		},
		{
			name: "ternary_with_numbers",
			expr: "{{score > 80 ? score * 2 : score}}",
			env: map[string]interface{}{
				"score": 90,
			},
			expected: 180,
		},
		{
			name: "ternary_with_dot_notation",
			expr: "{{user.premium ? user.price * 0.8 : user.price}}",
			env: map[string]interface{}{
				"user": map[string]interface{}{
					"premium": true,
					"price":   100.0,
				},
			},
			expected: 80.0,
		},
		{
			name: "ternary_with_get",
			expr: "{{get('discount') > 0 ? price * (1 - get('discount')) : price}}",
			env: map[string]interface{}{
				"price": 100.0,
			},
			expected: 90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &domain.UnifiedAPI{
				Get: func(name string, typeHint ...string) (interface{}, error) {
					store := map[string]interface{}{
						"discount": 0.1,
					}
					if val, ok := store[name]; ok {
						return val, nil
					}
					return nil, nil
				},
			}

			eval := NewEvaluator(api)
			parser := NewParser()
			
			expr, err := parser.Parse(tt.expr)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			result, err := eval.Evaluate(expr, tt.env)
			if err != nil {
				t.Fatalf("evaluation error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v (type %T), got %v (type %T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

// TestControlFlowLogicalOperators tests AND and OR operators.
func TestControlFlowLogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]interface{}
		expected interface{}
	}{
		{
			name: "logical_and_true",
			expr: "{{age >= 18 && verified}}",
			env: map[string]interface{}{
				"age":      20,
				"verified": true,
			},
			expected: true,
		},
		{
			name: "logical_and_false",
			expr: "{{age >= 18 && verified}}",
			env: map[string]interface{}{
				"age":      20,
				"verified": false,
			},
			expected: false,
		},
		{
			name: "logical_or_true",
			expr: "{{premium || trial}}",
			env: map[string]interface{}{
				"premium": false,
				"trial":   true,
			},
			expected: true,
		},
		{
			name: "logical_or_false",
			expr: "{{premium || trial}}",
			env: map[string]interface{}{
				"premium": false,
				"trial":   false,
			},
			expected: false,
		},
		{
			name: "complex_logical",
			expr: "{{(age >= 18 && verified) || admin}}",
			env: map[string]interface{}{
				"age":      15,
				"verified": false,
				"admin":    true,
			},
			expected: true,
		},
		{
			name: "logical_not",
			expr: "{{!disabled}}",
			env: map[string]interface{}{
				"disabled": false,
			},
			expected: true,
		},
		{
			name: "combined_with_comparison",
			expr: "{{score > 80 && grade == 'A'}}",
			env: map[string]interface{}{
				"score": 90,
				"grade": "A",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &domain.UnifiedAPI{}
			eval := NewEvaluator(api)
			parser := NewParser()
			
			expr, err := parser.Parse(tt.expr)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			result, err := eval.Evaluate(expr, tt.env)
			if err != nil {
				t.Fatalf("evaluation error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

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
			validate: func(t *testing.T, result interface{}) {
				arr, ok := result.([]interface{})
				if !ok {
					t.Fatalf("expected array, got %T", result)
				}
				if len(arr) != 2 {
					t.Errorf("expected 2 items, got %d", len(arr))
				}
			},
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
			validate: func(t *testing.T, result interface{}) {
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
			},
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
			validate: func(t *testing.T, result interface{}) {
				if result != true {
					t.Errorf("expected true, got %v", result)
				}
			},
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
			validate: func(t *testing.T, result interface{}) {
				if result != false {
					t.Errorf("expected false, got %v", result)
				}
			},
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
			validate: func(t *testing.T, result interface{}) {
				if result != true {
					t.Errorf("expected true, got %v", result)
				}
			},
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
			validate: func(t *testing.T, result interface{}) {
				if result != false {
					t.Errorf("expected false, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &domain.UnifiedAPI{}
			eval := NewEvaluator(api)
			parser := NewParser()
			
			expr, err := parser.Parse(tt.expr)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			result, err := eval.Evaluate(expr, tt.env)
			if err != nil {
				t.Fatalf("evaluation error: %v", err)
			}

			tt.validate(t, result)
		})
	}
}

// TestControlFlowCombined tests complex scenarios combining multiple features.
func TestControlFlowCombined(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		env      map[string]interface{}
		validate func(*testing.T, interface{})
	}{
		{
			name: "filter_with_ternary",
			expr: "{{filter(users, .age >= 18 ? true : false)}}",
			env: map[string]interface{}{
				"users": []map[string]interface{}{
					{"name": "Alice", "age": 20},
					{"name": "Bob", "age": 15},
				},
			},
			validate: func(t *testing.T, result interface{}) {
				arr, ok := result.([]interface{})
				if !ok {
					t.Fatalf("expected array, got %T", result)
				}
				if len(arr) != 1 {
					t.Errorf("expected 1 item, got %d", len(arr))
				}
			},
		},
		{
			name: "all_with_logical_operators",
			expr: "{{all(users, .age >= 18 && .verified)}}",
			env: map[string]interface{}{
				"users": []map[string]interface{}{
					{"age": 20, "verified": true},
					{"age": 25, "verified": true},
				},
			},
			validate: func(t *testing.T, result interface{}) {
				if result != true {
					t.Errorf("expected true, got %v", result)
				}
			},
		},
		{
			name: "map_with_ternary",
			expr: "{{map(items, .price > 100 ? .price * 0.9 : .price)}}",
			env: map[string]interface{}{
				"items": []map[string]interface{}{
					{"price": 150.0},
					{"price": 50.0},
				},
			},
			validate: func(t *testing.T, result interface{}) {
				arr, ok := result.([]interface{})
				if !ok {
					t.Fatalf("expected array, got %T", result)
				}
				if len(arr) != 2 {
					t.Errorf("expected 2 items, got %d", len(arr))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &domain.UnifiedAPI{}
			eval := NewEvaluator(api)
			parser := NewParser()
			
			expr, err := parser.Parse(tt.expr)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			result, err := eval.Evaluate(expr, tt.env)
			if err != nil {
				t.Fatalf("evaluation error: %v", err)
			}

			tt.validate(t, result)
		})
	}
}
