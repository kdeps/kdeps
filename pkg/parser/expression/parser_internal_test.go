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

func TestIsExpression_ArrayAccess(t *testing.T) {
	p := &Parser{}
	if !p.isExpression("arr[0]") {
		t.Error("expected isExpression true for 'arr[0]' (array access)")
	}
}

func TestLooksLikeAuthToken_DotExclusion(t *testing.T) {
	p := &Parser{}
	// "config.data.value" has >= 8 chars, contains dots,
	// matches ^[a-zA-Z0-9\-_\.]+$ AND property access pattern
	// so looksLikeAuthToken should return false
	if p.looksLikeAuthToken("config.data.value") {
		t.Error("expected false for property access pattern with dots")
	}
}

func TestParseSlice_Error(t *testing.T) {
	p := NewParser()

	// Unclosed braces in a string value causes ParseValue -> Parse to fail
	_, err := p.ParseSlice([]interface{}{"hello", "{{ unclosed"})
	if err == nil {
		t.Error("expected error for unclosed interpolation in slice")
	}
}

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

// TestOptionalBraces tests that {{ }} braces are optional for direct expressions
// but required for string interpolation.
func TestOptionalBraces(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name         string
		input        string
		expectedType domain.ExprType
		description  string
	}{
		// Direct expressions WITHOUT braces (new feature)
		{
			name:         "function_call_without_braces",
			input:        "get('q')",
			expectedType: domain.ExprTypeDirect,
			description:  "Function call without {{ }} should be direct expression",
		},
		{
			name:         "dot_notation_without_braces",
			input:        "user.email",
			expectedType: domain.ExprTypeDirect,
			description:  "Dot notation without {{ }} should be direct expression",
		},
		{
			name:         "nested_dot_notation_without_braces",
			input:        "user.profile.age",
			expectedType: domain.ExprTypeDirect,
			description:  "Nested dot notation without {{ }} should be direct expression",
		},
		{
			name:         "arithmetic_without_braces",
			input:        "2 + 3",
			expectedType: domain.ExprTypeDirect,
			description:  "Arithmetic without {{ }} should be direct expression",
		},
		{
			name:         "comparison_without_braces",
			input:        "count > 10",
			expectedType: domain.ExprTypeDirect,
			description:  "Comparison without {{ }} should be direct expression",
		},

		// Direct expressions WITH braces (backward compatibility)
		{
			name:         "function_call_with_braces",
			input:        "{{ get('q') }}",
			expectedType: domain.ExprTypeInterpolated,
			description:  "Function call with {{ }} should be interpolated",
		},
		{
			name:         "dot_notation_with_braces",
			input:        "{{user.email}}",
			expectedType: domain.ExprTypeInterpolated,
			description:  "Dot notation with {{ }} should be interpolated",
		},

		// String interpolation (braces REQUIRED)
		{
			name:         "string_interpolation_single",
			input:        "Hello {{name}}",
			expectedType: domain.ExprTypeInterpolated,
			description:  "String with interpolation requires {{ }}",
		},
		{
			name:         "string_interpolation_multiple",
			input:        "Hello {{first}} {{last}}",
			expectedType: domain.ExprTypeInterpolated,
			description:  "String with multiple interpolations requires {{ }}",
		},
		{
			name:         "string_interpolation_with_text",
			input:        "Score: {{score}} points",
			expectedType: domain.ExprTypeInterpolated,
			description:  "String with text and interpolation requires {{ }}",
		},

		// Literals (no braces needed)
		{
			name:         "numeric_literal",
			input:        "25",
			expectedType: domain.ExprTypeLiteral,
			description:  "Numeric literal should be literal",
		},
		{
			name:         "plain_string",
			input:        "hello world",
			expectedType: domain.ExprTypeLiteral,
			description:  "Plain string should be literal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if expr.Type != tt.expectedType {
				t.Errorf("Parse() type = %v, want %v\nDescription: %s\nInput: '%s'",
					expr.Type, tt.expectedType, tt.description, tt.input)
			}
		})
	}
}

// TestBracesRequiredForInterpolation verifies that {{ }} is required
// when interpolating within strings.
func TestBracesRequiredForInterpolation(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name         string
		input        string
		expectedType domain.ExprType
		reason       string
	}{
		{
			name:         "plain_text_with_placeholder",
			input:        "Hello name",
			expectedType: domain.ExprTypeLiteral,
			reason:       "Without {{ }}, it's just literal text",
		},
		{
			name:         "text_with_interpolation",
			input:        "Hello {{name}}",
			expectedType: domain.ExprTypeInterpolated,
			reason:       "With {{ }}, it becomes an interpolated string",
		},
		{
			name:         "single_expression",
			input:        "{{name}}",
			expectedType: domain.ExprTypeInterpolated,
			reason:       "Even a single {{ }} makes it interpolated",
		},
		{
			name:         "expression_without_braces",
			input:        "name",
			expectedType: domain.ExprTypeLiteral,
			reason:       "A simple word without {{ }} is literal",
		},
		{
			name:         "function_in_string_without_braces",
			input:        "Result: get('q')",
			expectedType: domain.ExprTypeDirect,
			reason:       "Function call is detected as expression even with prefix text",
		},
		{
			name:         "function_in_string_with_braces",
			input:        "Result: {{get('q')}}",
			expectedType: domain.ExprTypeInterpolated,
			reason:       "Function call in text with {{ }} is interpolated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if expr.Type != tt.expectedType {
				t.Errorf("Parse() type = %v, want %v\nReason: %s\nInput: '%s'",
					expr.Type, tt.expectedType, tt.reason, tt.input)
			}
		})
	}
}
