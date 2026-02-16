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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package expression_test

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func TestMustacheExpressions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		env      map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "{{q}}",
			env: map[string]interface{}{
				"q": "hello world",
			},
			expected: "hello world",
			wantErr:  false,
		},
		{
			name:     "simple variable with text",
			template: "Query: {{q}}",
			env: map[string]interface{}{
				"q": "test query",
			},
			expected: "Query: test query",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "Hello {{name}}, you scored {{score}}",
			env: map[string]interface{}{
				"name":  "Alice",
				"score": 95,
			},
			expected: "Hello Alice, you scored 95",
			wantErr:  false,
		},
		{
			name:     "nested object",
			template: "{{user.name}}",
			env: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Bob",
					"age":  30,
				},
			},
			expected: "Bob",
			wantErr:  false,
		},
		{
			name:     "missing variable returns empty",
			template: "{{missing}}",
			env:      map[string]interface{}{},
			expected: "",
			wantErr:  false,
		},
		{
			name:     "integer value",
			template: "{{count}}",
			env: map[string]interface{}{
				"count": 42,
			},
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "boolean value",
			template: "{{active}}",
			env: map[string]interface{}{
				"active": true,
			},
			expected: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := expression.NewParser()
			expr, err := parser.Parse(tt.template)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Should detect as mustache
			if expr.Type != domain.ExprTypeMustache {
				t.Errorf("Expected ExprTypeMustache, got %v", expr.Type)
			}

			evaluator := expression.NewEvaluator(createMockAPI())
			result, err := evaluator.Evaluate(expr, tt.env)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Evaluate() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExprLangVsMustacheDetection(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantType domain.ExprType
	}{
		{
			name:     "expr-lang with spaces",
			template: "{{ get('q') }}",
			wantType: domain.ExprTypeInterpolated,
		},
		{
			name:     "mustache without spaces",
			template: "{{q}}",
			wantType: domain.ExprTypeMustache,
		},
		{
			name:     "expr-lang with function",
			template: "{{get('q')}}",
			wantType: domain.ExprTypeInterpolated,
		},
		{
			name:     "mustache nested",
			template: "{{user.name}}",
			wantType: domain.ExprTypeMustache,
		},
		{
			name:     "expr-lang mixed text",
			template: "Hello {{ get('name') }}",
			wantType: domain.ExprTypeInterpolated,
		},
		{
			name:     "mustache mixed text",
			template: "Hello {{name}}",
			wantType: domain.ExprTypeMustache,
		},
		{
			name:     "mustache section",
			template: "{{#items}}Item{{/items}}",
			wantType: domain.ExprTypeMustache,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := expression.NewParser()
			expr, err := parser.Parse(tt.template)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if expr.Type != tt.wantType {
				t.Errorf("Got type %v, want %v", expr.Type, tt.wantType)
			}
		})
	}
}

func TestMustacheWithUnifiedAPI(t *testing.T) {
	// Test that mustache can access values from the environment
	// which includes results from unified API functions
	parser := expression.NewParser()
	evaluator := expression.NewEvaluator(createMockAPI())

	// Set a value using the API (simulating a previous action result)
	env := map[string]interface{}{
		"username": "test_user",
		"count":    10,
	}

	expr, err := parser.Parse("User: {{username}}, Count: {{count}}")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	result, err := evaluator.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	expected := "User: test_user, Count: 10"
	if result != expected {
		t.Errorf("Got %v, want %v", result, expected)
	}
}
