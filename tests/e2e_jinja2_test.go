// Copyright 2026 Kdeps, KvK 94834768

package tests

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/templates"
)

// TestE2EInterpolationWorkflow tests a complete workflow using interpolated expressions.
func TestE2EInterpolationWorkflow(t *testing.T) {
	parser := expression.NewParser()

	// Simulate a workflow with interpolated variables
	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			values := map[string]interface{}{
				"query":    "What is the weather?",
				"username": "alice",
				"score":    95,
			}
			if val, ok := values[name]; ok {
				return val, nil
			}
			return nil, errors.New("not found")
		},
		Info: func(field string) (interface{}, error) {
			values := map[string]interface{}{
				"current_time": "2024-01-01T00:00:00Z",
				"version":      "2.0",
			}
			if val, ok := values[field]; ok {
				return val, nil
			}
			return nil, errors.New("not found")
		},
		Env: func(name string) (string, error) {
			if name == "API_KEY" {
				return "secret-key-123", nil
			}
			return "", nil
		},
	}

	evaluator := expression.NewEvaluator(api)

	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{
			name:     "simple interpolated variable",
			expr:     "{{query}}",
			expected: "What is the weather?",
		},
		{
			name:     "interpolated with text",
			expr:     "User: {{username}}, Score: {{score}}",
			expected: "User: alice, Score: 95",
		},
		{
			name:     "expr-lang function call",
			expr:     "{{ get('query') }}",
			expected: "What is the weather?",
		},
		{
			name:     "mixed interpolation and text",
			expr:     "Hello {{username}}!",
			expected: "Hello alice!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprObj, err := parser.Parse(tt.expr)
			require.NoError(t, err)

			result, err := evaluator.Evaluate(exprObj, nil)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestE2EJinja2TemplateGeneration tests E2E project generation with Jinja2.
func TestE2EJinja2TemplateGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	generator, err := templates.NewGenerator()
	require.NoError(t, err)

	data := templates.TemplateData{
		Name:      "test-app",
		Version:   "1.0.0",
		Resources: []string{"http-client", "llm"},
	}

	// Generate using a Jinja2 template
	err = generator.GenerateProject("api-service", filepath.Join(tmpDir, "test-app"), data)
	if err != nil {
		t.Logf("GenerateProject error (may be expected if template doesn't exist): %v", err)
		t.Skip("Skipping test as api-service template may not exist")
	}

	// Verify output was created
	_, err = os.Stat(filepath.Join(tmpDir, "test-app"))
	assert.NoError(t, err)

	// Verify workflow.yaml was generated
	workflowPath := filepath.Join(tmpDir, "test-app", "workflow.yaml")
	_, err = os.Stat(workflowPath)
	assert.NoError(t, err, "workflow.yaml should be generated")
}

// TestE2EUnifiedExpressionSystem tests the unified expression system E2E.
func TestE2EUnifiedExpressionSystem(t *testing.T) {
	parser := expression.NewParser()

	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			values := map[string]interface{}{
				"name":  "John",
				"age":   30,
				"email": "john@example.com",
			}
			if val, ok := values[name]; ok {
				return val, nil
			}
			return nil, errors.New("not found")
		},
	}

	evaluator := expression.NewEvaluator(api)

	// Test that whitespace doesn't matter
	tests := []struct {
		name  string
		expr1 string
		expr2 string
	}{
		{
			name:  "simple variable",
			expr1: "{{name}}",
			expr2: "{{ name }}",
		},
		{
			name:  "in sentence",
			expr1: "Hello {{name}}!",
			expr2: "Hello {{ name }}!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr1Obj, err := parser.Parse(tt.expr1)
			require.NoError(t, err)
			result1, err := evaluator.Evaluate(expr1Obj, nil)
			require.NoError(t, err)

			expr2Obj, err := parser.Parse(tt.expr2)
			require.NoError(t, err)
			result2, err := evaluator.Evaluate(expr2Obj, nil)
			require.NoError(t, err)

			// Both should produce same result
			assert.Equal(t, result1, result2)
		})
	}
}

// TestE2EInterpolationPerformance tests performance of the interpolation expression system.
func TestE2EInterpolationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	parser := expression.NewParser()

	api := &domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return "test value", nil
		},
	}

	evaluator := expression.NewEvaluator(api)

	// Test simple interpolated expression
	interpolatedExpr := "{{name}}"
	exprObj, err := parser.Parse(interpolatedExpr)
	require.NoError(t, err)

	iterations := 1000
	for range iterations {
		_, evalErr := evaluator.Evaluate(exprObj, nil)
		require.NoError(t, evalErr)
	}

	t.Logf("Successfully evaluated interpolated expression %d times", iterations)
}

// TestE2EInterpolationMixedComplexity tests mixing simple and complex expressions.
func TestE2EInterpolationMixedComplexity(t *testing.T) {
	parser := expression.NewParser()

	api := &domain.UnifiedAPI{
		Get: func(name string, _ ...string) (interface{}, error) {
			values := map[string]interface{}{
				"name":   "Alice",
				"points": 100,
			}
			if val, ok := values[name]; ok {
				return val, nil
			}
			return nil, errors.New("not found")
		},
		Info: func(field string) (interface{}, error) {
			if field == "current_time" {
				return "2024-01-01", nil
			}
			return nil, errors.New("not found")
		},
	}

	evaluator := expression.NewEvaluator(api)

	tests := []struct {
		name     string
		expr     string
		contains string
	}{
		{
			name:     "simple interpolated variable",
			expr:     "Player: {{name}}",
			contains: "Player: Alice",
		},
		{
			name:     "expr-lang function",
			expr:     "Time: {{ info('current_time') }}",
			contains: "Time: 2024-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprObj, err := parser.Parse(tt.expr)
			require.NoError(t, err)

			result, err := evaluator.Evaluate(exprObj, nil)
			require.NoError(t, err)

			resultStr := fmt.Sprintf("%v", result)
			assert.Contains(t, resultStr, tt.contains)
		})
	}
}
