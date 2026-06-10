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
	"fmt"
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
				Get: func(name string, _ ...string) (interface{}, error) {
					store := map[string]interface{}{
						"discount": 0.1,
					}
					if val, ok := store[name]; ok {
						return val, nil
					}
					return nil, errors.New("not found")
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
				t.Errorf(
					"expected %v (type %T), got %v (type %T)",
					tt.expected,
					tt.expected,
					result,
					result,
				)
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

func TestEvaluateDirect_DebugMode(t *testing.T) {
	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			if name == "x" {
				return 42, nil
			}
			return nil, errors.New("not found")
		},
	}
	e := NewEvaluator(api)
	e.SetDebugMode(true)

	expr := &domain.Expression{Raw: "get('x')", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestTrySimpleVariable_InvalidIdentifier(t *testing.T) {
	e := NewEvaluator(nil)

	// {{ foo@bar }} -> not expr-lang syntax (no parens, ops, quotes),
	// not in env, not a simple identifier -> trySimpleVariable returns nil
	// -> falls to evaluateDirect which fails -> error
	expr := &domain.Expression{
		Raw:  "{{ foo@bar }}",
		Type: domain.ExprTypeInterpolated,
	}
	_, err := e.Evaluate(expr, map[string]interface{}{})
	if err == nil {
		t.Error("expected error for invalid identifier")
	}
}

func TestTrySimpleVariable_APISuccess(t *testing.T) {
	storage := map[string]interface{}{"mykey": "api-value"}
	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			if v, ok := storage[name]; ok {
				return v, nil
			}
			return nil, errors.New("not found")
		},
	}
	e := NewEvaluator(api)

	// "x: {{ mykey }}" forces multiple-interpolation path -> evaluateAndFormatExpression
	// -> trySimpleVariable("mykey", env) -> not in env, is simple identifier
	// -> e.api.Get("mykey") succeeds -> returns "api-value"
	expr := &domain.Expression{
		Raw:  "x: {{ mykey }}",
		Type: domain.ExprTypeInterpolated,
	}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "x: api-value" {
		t.Errorf("expected 'x: api-value', got %q", result)
	}
}

func TestFormatValue_JSONError(t *testing.T) {
	e := &Evaluator{}
	val := map[string]interface{}{
		"ch": make(chan int),
	}
	result := e.formatValue(val)
	// json.Marshal fails on chan values -> falls back to fmt.Sprintf("%v", val)
	// Go's fmt.Sprintf produces "map[ch:0x...]" for such values
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if result[0] != 'm' {
		t.Errorf("expected Go map formatting starting with 'm', got %q", result)
	}
}

func TestBuildEnvironment_InputFunction_Error(t *testing.T) {
	api := &domain.UnifiedAPI{
		Input: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("input error")
		},
	}
	e := NewEvaluator(api)

	expr := &domain.Expression{Raw: "input('key')", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil on input error, got %v", result)
	}
}

func TestBuildEnvironment_OutputFunction_Error(t *testing.T) {
	api := &domain.UnifiedAPI{
		Output: func(_ string) (interface{}, error) {
			return nil, errors.New("output error")
		},
	}
	e := NewEvaluator(api)

	expr := &domain.Expression{Raw: "output('res')", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil on output error, got %v", result)
	}
}

func TestBuildEnvironment_ItemCurrentPrevNext_Error(t *testing.T) {
	api := &domain.UnifiedAPI{
		Item: func(_ ...string) (interface{}, error) {
			return nil, errors.New("item error")
		},
	}
	e := NewEvaluator(api)

	tests := []struct {
		name string
		expr string
		want interface{}
	}{
		{"item.current error", "item.current()", nil},
		{"item.prev error", "item.prev()", nil},
		{"item.next error", "item.next()", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &domain.Expression{Raw: tt.expr, Type: domain.ExprTypeDirect}
			result, err := e.Evaluate(expr, nil)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}
			if result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestBuildEnvironment_ItemMergingElseBranch(t *testing.T) {
	api := &domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("not found")
		},
	}
	e := NewEvaluator(api)

	env := map[string]interface{}{
		"item": map[string]interface{}{"custom": "val"},
	}

	expr := &domain.Expression{Raw: "item.custom", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "val" {
		t.Errorf("expected 'val', got %v", result)
	}
}

func TestBuildEnvironment_LoopMergingElseBranch(t *testing.T) {
	api := &domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("not found")
		},
	}
	e := NewEvaluator(api)

	env := map[string]interface{}{
		"loop": map[string]interface{}{"custom": "lv"},
	}

	expr := &domain.Expression{Raw: "loop.custom", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "lv" {
		t.Errorf("expected 'lv', got %v", result)
	}
}

func TestFromJSON_Error(t *testing.T) {
	e := NewEvaluator(&domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("not found")
		},
	})

	expr := &domain.Expression{Raw: "fromJSON('not json')", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil from invalid JSON parse, got %v", result)
	}
}

func TestSafe_NilObjOrEmptyPath(t *testing.T) {
	env := map[string]interface{}{
		"obj": map[string]interface{}{"a": 1},
	}
	e := NewEvaluator(createMockAPIForCoverage())

	// safe(nil, "a") -> obj is nil -> return nil
	expr1 := &domain.Expression{Raw: "safe(nil, 'a')", Type: domain.ExprTypeDirect}
	r1, err := e.Evaluate(expr1, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if r1 != nil {
		t.Errorf("expected nil, got %v", r1)
	}

	// safe(obj, "") -> path is empty -> return nil
	expr2 := &domain.Expression{Raw: "safe(obj, '')", Type: domain.ExprTypeDirect}
	r2, err := e.Evaluate(expr2, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if r2 != nil {
		t.Errorf("expected nil, got %v", r2)
	}
}

func TestJSON_MarshalError(t *testing.T) {
	env := map[string]interface{}{
		"ch": make(chan int),
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{Raw: "json(ch)", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string result, got %T", result)
	}
	// On marshal error, json() returns the Go fmt string representation
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
}

func TestWhere_NonArrayInput(t *testing.T) {
	env := map[string]interface{}{"data": "string"}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{Raw: "where(data, 'k', 1)", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "string" {
		t.Errorf("expected passthrough 'string', got %v", result)
	}
}

func TestWhere_Int64Threshold(t *testing.T) {
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": float64(85)},
			map[string]interface{}{"score": float64(50)},
		},
		"thresh": int64(60),
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', thresh)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

func TestWhere_IntThreshold(t *testing.T) {
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": float64(85)},
		},
		"thresh": int(60),
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', thresh)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

func TestWhere_StringThreshold_ParseFail(t *testing.T) {
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": float64(85)},
		},
		"thresh": "not-a-number",
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', thresh)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	// Parse failure returns original array unchanged
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

func TestWhere_StringThreshold_ParseSuccess(t *testing.T) {
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": float64(85)},
			map[string]interface{}{"score": float64(50)},
		},
		"thresh": "60",
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', thresh)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

func TestWhere_DefaultThresholdType(t *testing.T) {
	// A boolean threshold hits the default branch in minVal switch
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": float64(85)},
		},
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', true)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	// Default branch returns arr unchanged
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

func TestWhere_MissingKeyAndNonMapItem(t *testing.T) {
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"other": 85},          // "score" key missing
			"not-a-map",                                  // non-map item skipped
			map[string]interface{}{"score": float64(90)}, // matches
		},
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', 60)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

func TestWhere_ScoreTypes(t *testing.T) {
	// Tests the case int, case int64, and default branches in score switch
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": int(85)},   // case int
			map[string]interface{}{"score": int64(72)}, // case int64
			map[string]interface{}{"score": "high"},    // default -> continue
		},
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', 60)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(arr))
	}
}

func TestLookupSimpleValue_NonMapIntermediate(t *testing.T) {
	e := NewEvaluator(nil)
	env := map[string]interface{}{
		"a": "notamap",
	}

	// {{ a.b }} -> trySimpleVariable -> lookupSimpleValue("a.b", env)
	// "a" is a string, not a map -> default branch -> returns nil
	// isSimpleIdentifier("a.b") is true -> api.Get("a.b") (nil api) -> returns ""
	expr := &domain.Expression{
		Raw:  "x: {{ a.b }}",
		Type: domain.ExprTypeInterpolated,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "x: " {
		t.Errorf("expected 'x: ', got %q", result)
	}
}

func TestBuildEnvironment_InputFunction_Success(t *testing.T) {
	api := &domain.UnifiedAPI{
		Input: func(_ string, _ ...string) (interface{}, error) {
			return "input-ok", nil
		},
	}
	e := NewEvaluator(api)

	expr := &domain.Expression{Raw: "input('key')", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != "input-ok" {
		t.Errorf("expected 'input-ok', got %v", result)
	}
}

func TestSafe_NilIntermediate(t *testing.T) {
	env := map[string]interface{}{
		"obj": map[string]interface{}{
			"a": map[string]interface{}{
				"b": nil,
			},
		},
	}
	e := NewEvaluator(createMockAPIForCoverage())

	// safe(obj, "a.b.c") -> after "b", current = nil -> returns nil
	expr := &domain.Expression{
		Raw:  "safe(obj, 'a.b.c')",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSafe_NonMapIntermediate(t *testing.T) {
	env := map[string]interface{}{
		"obj": map[string]interface{}{
			"a": "stringvalue",
		},
	}
	e := NewEvaluator(createMockAPIForCoverage())

	// safe(obj, "a.b") -> "a" is a string, not a map -> else branch -> returns nil
	expr := &domain.Expression{
		Raw:  "safe(obj, 'a.b')",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestDebug_MarshalError(t *testing.T) {
	env := map[string]interface{}{
		"ch": make(chan int),
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{Raw: "debug(ch)", Type: domain.ExprTypeDirect}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	s, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	// debug() returns "(error marshaling: ...)" on marshal failure
	if len(s) == 0 {
		t.Error("expected non-empty error string")
	}
}

func TestFromJSON_Success(t *testing.T) {
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "fromJSON('{\"a\": 1, \"b\": \"hello\"}')",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, nil)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	if m["a"] != float64(1) {
		t.Errorf("expected a=1, got %v", m["a"])
	}
	if m["b"] != "hello" {
		t.Errorf("expected b='hello', got %v", m["b"])
	}
}

func TestWhere_Float64Threshold(t *testing.T) {
	env := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"score": float64(85)},
			map[string]interface{}{"score": float64(50)},
		},
		"thresh": float64(60),
	}
	e := NewEvaluator(createMockAPIForCoverage())

	expr := &domain.Expression{
		Raw:  "where(items, 'score', thresh)",
		Type: domain.ExprTypeDirect,
	}
	result, err := e.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(arr))
	}
}

// TestHybridExpressions tests expressions that mix expr-lang functions and mustache-style dot notation.
func TestHybridExpressions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		env      map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "multiplication with dot notation only",
			template: "{{multiplier * user.age}}",
			env: map[string]interface{}{
				"multiplier": 2,
				"user": map[string]interface{}{
					"age": 30,
				},
			},
			expected: 60, // expr-lang returns int for integer operations
			wantErr:  false,
		},
		{
			name:     "addition with nested objects",
			template: "{{bonus + user.profile.score}}",
			env: map[string]interface{}{
				"bonus": 10,
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"score": 85,
					},
				},
			},
			expected: 95,
			wantErr:  false,
		},
		{
			name:     "string concatenation in template",
			template: "User {{user.name}} has {{points}} points",
			env: map[string]interface{}{
				"points": 100,
				"user": map[string]interface{}{
					"name": "Alice",
				},
			},
			expected: "User Alice has 100 points",
			wantErr:  false,
		},
		{
			name:     "complex arithmetic",
			template: "{{(price * quantity) + user.discount}}",
			env: map[string]interface{}{
				"price":    50,
				"quantity": 3,
				"user": map[string]interface{}{
					"discount": 10,
				},
			},
			expected: 160, // (50 * 3) + 10
			wantErr:  false,
		},
		{
			name:     "comparison with dot notation",
			template: "{{user.age > minAge}}",
			env: map[string]interface{}{
				"minAge": 18,
				"user": map[string]interface{}{
					"age": 25,
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name:     "ternary with dot notation",
			template: "{{user.premium ? premiumPrice : regularPrice}}",
			env: map[string]interface{}{
				"premiumPrice": 99,
				"regularPrice": 49,
				"user": map[string]interface{}{
					"premium": true,
				},
			},
			expected: 99,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create evaluator without API (we're just testing expression evaluation with env)
			evaluator := NewEvaluator(nil)

			// Parse the expression
			expr := &domain.Expression{
				Raw:  tt.template,
				Type: domain.ExprTypeInterpolated,
			}

			// Evaluate
			result, err := evaluator.Evaluate(expr, tt.env)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check result
			if !tt.wantErr && result != tt.expected {
				t.Errorf(
					"Evaluate() = %v (type %T), want %v (type %T)",
					result,
					result,
					tt.expected,
					tt.expected,
				)
			}
		})
	}
}

// TestHybridExpressionsWithAPI tests hybrid expressions with the UnifiedAPI.
func TestHybridExpressionsWithAPI(t *testing.T) {
	// Create a simple API with Get function
	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			// Simulate a simple key-value store
			store := map[string]interface{}{
				"multiplier": 5,
				"price":      25,
				"bonus":      15,
				"C":          3,
			}
			if val, ok := store[name]; ok {
				return val, nil
			}
			return nil, errors.New("not found")
		},
	}

	evaluator := NewEvaluator(api)

	tests := []struct {
		name     string
		template string
		env      map[string]interface{}
		expected interface{}
	}{
		{
			name:     "API get() with env variable",
			template: "{{get('multiplier') * quantity}}",
			env: map[string]interface{}{
				"quantity": 10,
			},
			expected: 50,
		},
		{
			name:     "API get() with nested env",
			template: "{{get('price') + order.shipping}}",
			env: map[string]interface{}{
				"order": map[string]interface{}{
					"shipping": 5,
				},
			},
			expected: 30,
		},
		{
			name:     "get('C') * user.email - exact example from requirement",
			template: "{{get('C') * user.rmail}}",
			env: map[string]interface{}{
				"user": map[string]interface{}{
					"rmail": 20, // Treating rmail as a numeric value for multiplication
				},
			},
			expected: 60, // 3 * 20
		},
		{
			name:     "complex hybrid: get() + dot notation + operators",
			template: "{{(get('price') * order.quantity) + user.discount}}",
			env: map[string]interface{}{
				"order": map[string]interface{}{
					"quantity": 2,
				},
				"user": map[string]interface{}{
					"discount": 10,
				},
			},
			expected: 60, // (25 * 2) + 10
		},
		{
			name:     "get() in ternary with dot notation",
			template: "{{user.premium ? get('price') * 2 : get('price')}}",
			env: map[string]interface{}{
				"user": map[string]interface{}{
					"premium": true,
				},
			},
			expected: 50, // 25 * 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &domain.Expression{
				Raw:  tt.template,
				Type: domain.ExprTypeInterpolated,
			}

			result, err := evaluator.Evaluate(expr, tt.env)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf(
					"Evaluate() = %v (type %T), want %v (type %T)",
					result,
					result,
					tt.expected,
					tt.expected,
				)
			}
		})
	}
}

// TestOptionalBracesEvaluation tests that expressions evaluate correctly
// with or without braces.
func TestOptionalBracesEvaluation(t *testing.T) {
	parser := NewParser()

	// Create mock API
	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			switch name {
			case "q":
				return "test query", nil
			case "count":
				return 42, nil
			default:
				return nil, errors.New("not found")
			}
		},
		Info: func(field string) (interface{}, error) {
			switch field {
			case "name":
				return "Test Workflow", nil
			default:
				return nil, fmt.Errorf("unknown info field: %s", field)
			}
		},
	}

	evaluator := NewEvaluator(api)

	// Test environment
	env := map[string]interface{}{
		"user": map[string]interface{}{
			"email": "test@example.com",
			"name":  "John",
			"age":   30,
		},
		"score":  100,
		"count":  5,
		"active": true,
	}

	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		// Without braces
		{
			name:     "function_without_braces",
			input:    "get('q')",
			expected: "test query",
		},
		{
			name:     "dot_notation_without_braces",
			input:    "user.email",
			expected: "test@example.com",
		},
		{
			name:     "nested_dot_without_braces",
			input:    "user.name",
			expected: "John",
		},
		{
			name:     "arithmetic_without_braces",
			input:    "score + 10",
			expected: 110,
		},
		{
			name:     "comparison_without_braces",
			input:    "count > 3",
			expected: true,
		},

		// With braces (backward compatibility)
		{
			name:     "function_with_braces",
			input:    "{{ get('q') }}",
			expected: "test query",
		},
		{
			name:     "dot_notation_with_braces",
			input:    "{{user.email}}",
			expected: "test@example.com",
		},

		// String interpolation (braces required)
		{
			name:     "string_interpolation_single",
			input:    "Hello {{user.name}}",
			expected: "Hello John",
		},
		{
			name:     "string_interpolation_multiple",
			input:    "{{user.name}} has {{score}} points",
			expected: "John has 100 points",
		},
		{
			name:     "string_interpolation_with_function",
			input:    "Query: {{get('q')}}",
			expected: "Query: test query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result, err := evaluator.Evaluate(expr, env)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("Evaluate() = %v (%T), want %v (%T)",
					result, result, tt.expected, tt.expected)
			}
		})
	}
}
