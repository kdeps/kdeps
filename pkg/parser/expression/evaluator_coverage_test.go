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

// ---------------------------------------------------------------------------
// evaluateDirect — debug mode branch (line 129-137)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// isExprLangSyntax — double-quote branch (line 292-293)
// ---------------------------------------------------------------------------

func TestIsExprLangSyntax_DoubleQuoted(t *testing.T) {
	if !isExprLangSyntax(`"hello"`) {
		t.Error(`expected true for "hello"`)
	}
	if !isExprLangSyntax(`"foo bar"`) {
		t.Error(`expected true for "foo bar"`)
	}
}

// ---------------------------------------------------------------------------
// isSimpleIdentifier — !isValid branch (line 332)
// ---------------------------------------------------------------------------

func TestIsSimpleIdentifier_InvalidChars(t *testing.T) {
	if isSimpleIdentifier("foo@bar") {
		t.Error("expected false for foo@bar")
	}
	if isSimpleIdentifier("foo bar") {
		t.Error("expected false for 'foo bar'")
	}
	if isSimpleIdentifier("foo$bar") {
		t.Error("expected false for foo$bar")
	}
	if isSimpleIdentifier("foo!bar") {
		t.Error("expected false for foo!bar")
	}
}

// ---------------------------------------------------------------------------
// trySimpleVariable — !isSimpleIdentifier branch (line 353-355)
// reaches via Evaluate with a non-identifier, non-expr-lang expression
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// trySimpleVariable — api.Get success branch (line 359-361)
// identifier not in env but found in API storage
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// formatValue — jsonErr != nil branch (line 255-257)
// map with a channel value causes json.Marshal to fail
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — input() error branch (line 462-463)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — output() error branch (line 474-475)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — item.current/prev/next error branches
// (lines 499, 505, 512 — return nil on error)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — item env merging else branch (line 742 - env has item,
// but evalEnv does not yet because api.Item is nil)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — loop env merging else branch (line 753)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — fromJSON error branch (line 649-650)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — safe() nil obj / empty path branch (line 605-606)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — json() marshal error branch (line 596-597)
// a bare channel value causes json.Marshal to fail after stripFuncs
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — where() edge cases (lines 659-678, 682-699)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// EvaluateCondition — slice/array check (line 789-791) and default error
// (line 793)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// isExpression — array access pattern (line 218-220)
// ---------------------------------------------------------------------------

func TestIsExpression_ArrayAccess(t *testing.T) {
	p := &Parser{}
	if !p.isExpression("arr[0]") {
		t.Error("expected isExpression true for 'arr[0]' (array access)")
	}
}

// ---------------------------------------------------------------------------
// looksLikeAuthToken — dot exclusion branch (line 340-343)
// token with dots that also matches property access pattern
// ---------------------------------------------------------------------------

func TestLooksLikeAuthToken_DotExclusion(t *testing.T) {
	p := &Parser{}
	// "config.data.value" has >= 8 chars, contains dots,
	// matches ^[a-zA-Z0-9\-_\.]+$ AND property access pattern
	// so looksLikeAuthToken should return false
	if p.looksLikeAuthToken("config.data.value") {
		t.Error("expected false for property access pattern with dots")
	}
}

// ---------------------------------------------------------------------------
// ParseSlice — error path (line 106-108)
// ---------------------------------------------------------------------------

func TestParseSlice_Error(t *testing.T) {
	p := NewParser()

	// Unclosed braces in a string value causes ParseValue -> Parse to fail
	_, err := p.ParseSlice([]interface{}{"hello", "{{ unclosed"})
	if err == nil {
		t.Error("expected error for unclosed interpolation in slice")
	}
}

// ---------------------------------------------------------------------------
// ParseMap — error path (line 120-122)
// ---------------------------------------------------------------------------

func TestParseMap_Error(t *testing.T) {
	p := NewParser()

	// Unclosed braces in a map value causes ParseValue -> Parse to fail
	_, err := p.ParseMap(map[string]interface{}{
		"good": "hello",
		"bad":  "{{ unclosed",
	})
	if err == nil {
		t.Error("expected error for unclosed interpolation in map value")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// lookupSimpleValue — default branch (line 383-385)
// dot-notation navigation when intermediate value is not a map
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — input() function success branch (line 465)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — safe() nil intermediate check (line 611-613)
// safe(obj, "a.b.c") where obj.a.b is nil — next iteration hits nil check
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — safe() else branch (line 620-622)
// navigation through a non-map value
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — debug() marshal error branch (line 630-632)
// pass a non-JSON-serializable value (channel) to debug()
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — fromJSON() success branch (line 652)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildEnvironment — where() float64 threshold case (line 665-666)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// looksLikeAuthToken — final return false (line 345)
// value with special chars (like @) that is >= 8 chars but not a token
// ---------------------------------------------------------------------------

func TestLooksLikeAuthToken_FinalReturnFalse(t *testing.T) {
	p := &Parser{}

	// "user@example.com" has >= 8 chars, contains "@" which is not in
	// containsInvalidChars check (only parens/quotes/brackets/braces/spaces),
	// so it passes through but fails ^[a-zA-Z0-9\-_\.]+$ regex
	if p.looksLikeAuthToken("user@example.com") {
		t.Error("expected false for email-like string")
	}

	// "value$with$special" similarly should return false
	if p.looksLikeAuthToken("value$with$special") {
		t.Error("expected false for string with $")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// createMockAPIForCoverage returns a minimal API for coverage tests that need
// non-nil api to register helper functions (json, safe, fromJSON, where, etc.)
func createMockAPIForCoverage() *domain.UnifiedAPI {
	return &domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("not found")
		},
	}
}
