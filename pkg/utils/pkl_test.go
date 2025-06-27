package utils_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// MockEvaluator implements pkl.Evaluator for testing
type MockEvaluator struct {
	shouldError bool
	result      interface{}
}

func (m *MockEvaluator) EvaluateModule(ctx context.Context, source *pkl.ModuleSource, out interface{}) error {
	return errors.New("not implemented")
}

func (m *MockEvaluator) EvaluateOutputText(ctx context.Context, source *pkl.ModuleSource) (string, error) {
	return "", errors.New("not implemented")
}

func (m *MockEvaluator) EvaluateOutputValue(ctx context.Context, source *pkl.ModuleSource, out interface{}) error {
	return errors.New("not implemented")
}

func (m *MockEvaluator) EvaluateOutputFiles(ctx context.Context, source *pkl.ModuleSource) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (m *MockEvaluator) EvaluateExpression(ctx context.Context, source *pkl.ModuleSource, expr string, out interface{}) error {
	if m.shouldError {
		return errors.New("evaluation failed")
	}
	// Set the out parameter to our mock result
	switch v := out.(type) {
	case *interface{}:
		*v = m.result
	}
	return nil
}

func (m *MockEvaluator) EvaluateExpressionRaw(ctx context.Context, source *pkl.ModuleSource, expr string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (m *MockEvaluator) Close() error {
	return nil
}

func (m *MockEvaluator) Closed() bool {
	return false
}

func TestEvaluateStringToJSON(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	tests := []struct {
		name           string
		input          string
		mockEvaluator  *MockEvaluator
		expectedOutput string
		expectContains string
	}{
		{
			name:  "valid JSON gets compacted",
			input: `{"valid": "json"}`,
			mockEvaluator: &MockEvaluator{
				shouldError: true, // PKL evaluation is no longer performed
			},
			expectedOutput: "{\"valid\":\"json\"}",
		},
		{
			name:  "invalid JSON gets quoted",
			input: "neither pkl nor json",
			mockEvaluator: &MockEvaluator{
				shouldError: true, // PKL evaluation is no longer performed
			},
			expectedOutput: "\"neither pkl nor json\"",
		},
		{
			name:  "malformed JSON gets fixed",
			input: `{"broken": "json"`,
			mockEvaluator: &MockEvaluator{
				shouldError: true, // PKL evaluation is no longer performed
			},
			expectContains: "broken",
		},
		{
			name:  "PKL expressions are treated as raw text",
			input: `"new Listing { \"item1\"; \"item2\" }"`,
			mockEvaluator: &MockEvaluator{
				shouldError: true, // PKL evaluation is no longer performed
			},
			expectedOutput: "\"new Listing { \\\"item1\\\"; \\\"item2\\\" }\"",
		},
		{
			name:  "complex strings return as-is",
			input: `some complex string with "quotes"`,
			mockEvaluator: &MockEvaluator{
				shouldError: true, // PKL evaluation is no longer performed
			},
			expectedOutput: `some complex string with "quotes"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.EvaluateStringToJSON(tt.input, logger, tt.mockEvaluator, ctx)

			assert.NoError(t, err)

			if tt.expectedOutput != "" {
				assert.Equal(t, tt.expectedOutput, result)
			}

			if tt.expectContains != "" {
				assert.Contains(t, result, tt.expectContains)
			}
		})
	}
}

func TestEncodePklMap(t *testing.T) {
	tests := []struct {
		name     string
		input    *map[string]string
		expected string
	}{
		{
			name:     "NilMap",
			input:    nil,
			expected: "{}",
		},
		{
			name:     "EmptyMap",
			input:    &map[string]string{},
			expected: "{}",
		},
		{
			name: "SingleEntry",
			input: &map[string]string{
				"key": "value",
			},
			expected: `{["key"]="dmFsdWU="}`,
		},
		{
			name: "SpecialCharacters",
			input: &map[string]string{
				"key with spaces": "value with \"quotes\"",
			},
			expected: `{["key with spaces"]="dmFsdWUgd2l0aCAicXVvdGVzIg=="}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.EncodePklMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Additional check for maps with multiple entries where ordering is not deterministic.
	t.Run("MultipleEntries", func(t *testing.T) {
		input := &map[string]string{"key1": "value1", "key2": "value2"}
		result := utils.EncodePklMap(input)
		assert.Contains(t, result, `["key1"]="dmFsdWUx"`)
		assert.Contains(t, result, `["key2"]="dmFsdWUy"`)
	})
}

func TestEncodePklSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    *[]string
		expected string
	}{
		{
			name:     "NilSlice",
			input:    nil,
			expected: "{}",
		},
		{
			name:     "EmptySlice",
			input:    &[]string{},
			expected: "{}",
		},
		{
			name:     "SingleEntry",
			input:    &[]string{"value"},
			expected: `{"dmFsdWU="}`,
		},
		{
			name:     "SpecialCharacters",
			input:    &[]string{"value with \"quotes\""},
			expected: `{"dmFsdWUgd2l0aCAicXVvdGVzIg=="}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.EncodePklSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "EmptyString",
			input:    "",
			expected: "",
		},
		{
			name:     "SimpleString",
			input:    "test",
			expected: "dGVzdA==",
		},
		{
			name:     "AlreadyEncoded",
			input:    "dGVzdA==",
			expected: "dGVzdA==",
		},
		{
			name:     "SpecialCharacters",
			input:    "test with spaces and \"quotes\"",
			expected: "dGVzdCB3aXRoIHNwYWNlcyBhbmQgInF1b3RlcyI=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.EncodeValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatRequestHeadersAndParamsExtra(t *testing.T) {
	headers := map[string][]string{
		"X-Test": {" value1 ", "value2"},
	}
	params := map[string][]string{
		"query": {"foo", " bar "},
	}

	// Exercise helpers
	hdrOut := utils.FormatRequestHeaders(headers)
	prmOut := utils.FormatRequestParams(params)

	// Basic structural checks
	if !strings.HasPrefix(hdrOut, "headers{") || !strings.HasSuffix(hdrOut, "}") {
		t.Fatalf("unexpected headers formatting: %q", hdrOut)
	}
	if !strings.HasPrefix(prmOut, "params{") || !strings.HasSuffix(prmOut, "}") {
		t.Fatalf("unexpected params formatting: %q", prmOut)
	}

	// Verify that each value is Base64-encoded and trimmed
	encodedVal := base64.StdEncoding.EncodeToString([]byte("value1"))
	if !strings.Contains(hdrOut, encodedVal) {
		t.Errorf("expected encoded header value %q in %q", encodedVal, hdrOut)
	}
	encodedVal = base64.StdEncoding.EncodeToString([]byte("bar"))
	if !strings.Contains(prmOut, encodedVal) {
		t.Errorf("expected encoded param value %q in %q", encodedVal, prmOut)
	}
}

func TestFormatResponseHeadersAndPropertiesExtra(t *testing.T) {
	headers := map[string]string{"Content-Type": " application/json "}
	props := map[string]string{"status": " ok "}

	hdrOut := utils.FormatResponseHeaders(headers)
	propOut := utils.FormatResponseProperties(props)

	if !strings.Contains(hdrOut, `["Content-Type"]="application/json"`) {
		t.Errorf("unexpected response headers output: %q", hdrOut)
	}
	if !strings.Contains(propOut, `["status"]="ok"`) {
		t.Errorf("unexpected response properties output: %q", propOut)
	}
}

func TestPKLHTTPFormattersAdditional(t *testing.T) {
	headers := map[string][]string{"X-Test": {" value "}}
	hStr := utils.FormatRequestHeaders(headers)
	if !strings.Contains(hStr, "X-Test") {
		t.Fatalf("header name missing in output")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("value"))
	if !strings.Contains(hStr, encoded) {
		t.Fatalf("encoded value missing in output; got %s", hStr)
	}

	params := map[string][]string{"q": {"k &v"}}
	pStr := utils.FormatRequestParams(params)
	encodedParam := base64.StdEncoding.EncodeToString([]byte("k &v"))
	if !strings.Contains(pStr, "q") || !strings.Contains(pStr, encodedParam) {
		t.Fatalf("param formatting incorrect: %s", pStr)
	}

	respHeaders := map[string]string{"Content-Type": "application/json"}
	rhStr := utils.FormatResponseHeaders(respHeaders)
	if !strings.Contains(rhStr, "Content-Type") {
		t.Fatalf("response header missing")
	}

	props := map[string]string{"prop": "123"}
	propStr := utils.FormatResponseProperties(props)
	if !strings.Contains(propStr, "prop") {
		t.Fatalf("response prop missing")
	}
}

func TestFormatRequestHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected string
	}{
		{
			name:     "EmptyHeaders",
			input:    map[string][]string{},
			expected: "headers{}",
		},
		{
			name: "SingleHeader",
			input: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expected: `headers{["Content-Type"]="YXBwbGljYXRpb24vanNvbg=="}`,
		},
		{
			name: "MultipleHeaders",
			input: map[string][]string{
				"Content-Type": {"application/json"},
				"Accept":       {"text/plain"},
			},
			expected: `headers{["Content-Type"]="YXBwbGljYXRpb24vanNvbg==";["Accept"]="dGV4dC9wbGFpbg=="}`,
		},
		{
			name: "MultipleValues",
			input: map[string][]string{
				"Accept": {"text/plain", "application/json"},
			},
			expected: `headers{["Accept"]="dGV4dC9wbGFpbg==";["Accept"]="YXBwbGljYXRpb24vanNvbg=="}`,
		},
		{
			name: "SpecialCharacters",
			input: map[string][]string{
				"X-Custom": {"value with spaces and \"quotes\""},
			},
			expected: `headers{["X-Custom"]="dmFsdWUgd2l0aCBzcGFjZXMgYW5kICJxdW90ZXMi"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatRequestHeaders(tt.input)
			if tt.name == "MultipleHeaders" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["Content-Type"]="YXBwbGljYXRpb24vanNvbg=="`)
				assert.Contains(t, result, `["Accept"]="dGV4dC9wbGFpbg=="`)
				assert.Contains(t, result, "headers{")
				assert.Contains(t, result, "}")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatRequestParams(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string][]string
		expected string
	}{
		{
			name:     "EmptyParams",
			input:    map[string][]string{},
			expected: "params{}",
		},
		{
			name: "SingleParam",
			input: map[string][]string{
				"query": {"search"},
			},
			expected: `params{["query"]="c2VhcmNo"}`,
		},
		{
			name: "MultipleParams",
			input: map[string][]string{
				"query":  {"search"},
				"filter": {"active"},
			},
			expected: `params{["query"]="c2VhcmNo";["filter"]="YWN0aXZl"}`,
		},
		{
			name: "MultipleValues",
			input: map[string][]string{
				"tags": {"tag1", "tag2"},
			},
			expected: `params{["tags"]="dGFnMQ==";["tags"]="dGFnMg=="}`,
		},
		{
			name: "SpecialCharacters",
			input: map[string][]string{
				"search": {"value with spaces and \"quotes\""},
			},
			expected: `params{["search"]="dmFsdWUgd2l0aCBzcGFjZXMgYW5kICJxdW90ZXMi"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatRequestParams(tt.input)
			if tt.name == "MultipleParams" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["query"]="c2VhcmNo"`)
				assert.Contains(t, result, `["filter"]="YWN0aXZl"`)
				assert.Contains(t, result, "params{")
				assert.Contains(t, result, "}")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatResponseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "EmptyHeaders",
			input:    map[string]string{},
			expected: "headers{}",
		},
		{
			name: "SingleHeader",
			input: map[string]string{
				"Content-Type": "application/json",
			},
			expected: `headers{["Content-Type"]="application/json"}`,
		},
		{
			name: "MultipleHeaders",
			input: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "text/plain",
			},
			expected: `headers{["Content-Type"]="application/json";["Accept"]="text/plain"}`,
		},
		{
			name: "SpecialCharacters",
			input: map[string]string{
				"X-Custom": "value with spaces and \"quotes\"",
			},
			expected: `headers{["X-Custom"]="value with spaces and "quotes""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatResponseHeaders(tt.input)
			if tt.name == "MultipleHeaders" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["Content-Type"]="application/json"`)
				assert.Contains(t, result, `["Accept"]="text/plain"`)
				assert.Contains(t, result, "headers{")
				assert.Contains(t, result, "}")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatResponseProperties(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "EmptyProperties",
			input:    map[string]string{},
			expected: "properties{}",
		},
		{
			name: "SingleProperty",
			input: map[string]string{
				"status": "success",
			},
			expected: `properties{["status"]="success"}`,
		},
		{
			name: "MultipleProperties",
			input: map[string]string{
				"status":  "success",
				"message": "operation completed",
			},
			expected: `properties{["status"]="success";["message"]="operation completed"}`,
		},
		{
			name: "SpecialCharacters",
			input: map[string]string{
				"description": "value with spaces and \"quotes\"",
			},
			expected: `properties{["description"]="value with spaces and "quotes""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatResponseProperties(tt.input)
			if tt.name == "MultipleProperties" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["status"]="success"`)
				assert.Contains(t, result, `["message"]="operation completed"`)
				assert.Contains(t, result, "properties{")
				assert.Contains(t, result, "}")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatRequestHeadersAndParams(t *testing.T) {
	headers := map[string][]string{
		"Content-Type": {"application/json"},
	}
	out := utils.FormatRequestHeaders(headers)
	encoded := utils.EncodeBase64String("application/json")
	assert.Contains(t, out, encoded)
	assert.Contains(t, out, "Content-Type")

	params := map[string][]string{"q": {"search"}}
	out2 := utils.FormatRequestParams(params)
	encParam := utils.EncodeBase64String("search")
	assert.Contains(t, out2, encParam)
	assert.Contains(t, out2, "q")
}

func TestFormatResponseHeadersAndProps(t *testing.T) {
	hdr := map[string]string{"X-Rate": "10"}
	out := utils.FormatResponseHeaders(hdr)
	assert.Contains(t, out, "X-Rate")
	assert.Contains(t, out, "10")

	props := map[string]string{"k": "v"}
	outp := utils.FormatResponseProperties(props)
	assert.Contains(t, outp, "k")
	assert.Contains(t, outp, "v")
}

func TestBase64EncodingHappens(t *testing.T) {
	value := "trim "
	hdr := map[string][]string{"H": {value}}
	out := utils.FormatRequestHeaders(hdr)
	// Should contain base64 trimmed value not plain
	assert.NotContains(t, out, value)
	encoded := base64.StdEncoding.EncodeToString([]byte("trim"))
	assert.Contains(t, out, encoded)
}
