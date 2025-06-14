package utils

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatRequestHeaders(t *testing.T) {

	tests := []struct {
		name     string
		input    map[string][]string
		expected string
	}{
		{
			name:     "EmptyHeaders",
			input:    map[string][]string{},
			expected: "headers {\n\n}",
		},
		{
			name: "SingleHeader",
			input: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expected: "headers {\n[\"Content-Type\"] = \"YXBwbGljYXRpb24vanNvbg==\"\n}",
		},
		{
			name: "MultipleHeaders",
			input: map[string][]string{
				"Content-Type": {"application/json"},
				"Accept":       {"text/plain"},
			},
			expected: "headers {\n[\"Content-Type\"] = \"YXBwbGljYXRpb24vanNvbg==\"\n[\"Accept\"] = \"dGV4dC9wbGFpbg==\"\n}",
		},
		{
			name: "MultipleValues",
			input: map[string][]string{
				"Accept": {"text/plain", "application/json"},
			},
			expected: "headers {\n[\"Accept\"] = \"dGV4dC9wbGFpbg==\"\n[\"Accept\"] = \"YXBwbGljYXRpb24vanNvbg==\"\n}",
		},
		{
			name: "SpecialCharacters",
			input: map[string][]string{
				"X-Custom": {"value with spaces and \"quotes\""},
			},
			expected: "headers {\n[\"X-Custom\"] = \"dmFsdWUgd2l0aCBzcGFjZXMgYW5kICJxdW90ZXMi\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		
			result := FormatRequestHeaders(tt.input)
			if tt.name == "MultipleHeaders" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["Content-Type"] = "YXBwbGljYXRpb24vanNvbg=="`)
				assert.Contains(t, result, `["Accept"] = "dGV4dC9wbGFpbg=="`)
				assert.Contains(t, result, "headers {")
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
			expected: "params {\n\n}",
		},
		{
			name: "SingleParam",
			input: map[string][]string{
				"query": {"search"},
			},
			expected: "params {\n[\"query\"] = \"c2VhcmNo\"\n}",
		},
		{
			name: "MultipleParams",
			input: map[string][]string{
				"query":  {"search"},
				"filter": {"active"},
			},
			expected: "params {\n[\"query\"] = \"c2VhcmNo\"\n[\"filter\"] = \"YWN0aXZl\"\n}",
		},
		{
			name: "MultipleValues",
			input: map[string][]string{
				"tags": {"tag1", "tag2"},
			},
			expected: "params {\n[\"tags\"] = \"dGFnMQ==\"\n[\"tags\"] = \"dGFnMg==\"\n}",
		},
		{
			name: "SpecialCharacters",
			input: map[string][]string{
				"search": {"value with spaces and \"quotes\""},
			},
			expected: "params {\n[\"search\"] = \"dmFsdWUgd2l0aCBzcGFjZXMgYW5kICJxdW90ZXMi\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		
			result := FormatRequestParams(tt.input)
			if tt.name == "MultipleParams" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["query"] = "c2VhcmNo"`)
				assert.Contains(t, result, `["filter"] = "YWN0aXZl"`)
				assert.Contains(t, result, "params {")
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
			expected: "headers {\n\n}",
		},
		{
			name: "SingleHeader",
			input: map[string]string{
				"Content-Type": "application/json",
			},
			expected: "headers {\n[\"Content-Type\"] = \"application/json\"\n}",
		},
		{
			name: "MultipleHeaders",
			input: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "text/plain",
			},
			expected: "headers {\n[\"Content-Type\"] = \"application/json\"\n[\"Accept\"] = \"text/plain\"\n}",
		},
		{
			name: "SpecialCharacters",
			input: map[string]string{
				"X-Custom": "value with spaces and \"quotes\"",
			},
			expected: "headers {\n[\"X-Custom\"] = \"value with spaces and \"quotes\"\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		
			result := FormatResponseHeaders(tt.input)
			if tt.name == "MultipleHeaders" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["Content-Type"] = "application/json"`)
				assert.Contains(t, result, `["Accept"] = "text/plain"`)
				assert.Contains(t, result, "headers {")
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
			expected: "properties {\n\n}",
		},
		{
			name: "SingleProperty",
			input: map[string]string{
				"status": "success",
			},
			expected: "properties {\n[\"status\"] = \"success\"\n}",
		},
		{
			name: "MultipleProperties",
			input: map[string]string{
				"status":  "success",
				"message": "operation completed",
			},
			expected: "properties {\n[\"status\"] = \"success\"\n[\"message\"] = \"operation completed\"\n}",
		},
		{
			name: "SpecialCharacters",
			input: map[string]string{
				"description": "value with spaces and \"quotes\"",
			},
			expected: "properties {\n[\"description\"] = \"value with spaces and \"quotes\"\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		
			result := FormatResponseProperties(tt.input)
			if tt.name == "MultipleProperties" {
				// Since map iteration order is not guaranteed, check that both lines are present
				assert.Contains(t, result, `["status"] = "success"`)
				assert.Contains(t, result, `["message"] = "operation completed"`)
				assert.Contains(t, result, "properties {")
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
	out := FormatRequestHeaders(headers)
	encoded := EncodeBase64String("application/json")
	assert.Contains(t, out, encoded)
	assert.Contains(t, out, "Content-Type")

	params := map[string][]string{"q": {"search"}}
	out2 := FormatRequestParams(params)
	encParam := EncodeBase64String("search")
	assert.Contains(t, out2, encParam)
	assert.Contains(t, out2, "q")
}

func TestFormatResponseHeadersAndProps(t *testing.T) {
	hdr := map[string]string{"X-Rate": "10"}
	out := FormatResponseHeaders(hdr)
	assert.Contains(t, out, "X-Rate")
	assert.Contains(t, out, "10")

	props := map[string]string{"k": "v"}
	outp := FormatResponseProperties(props)
	assert.Contains(t, outp, "k")
	assert.Contains(t, outp, "v")
}

func TestBase64EncodingHappens(t *testing.T) {
	value := "trim "
	hdr := map[string][]string{"H": {value}}
	out := FormatRequestHeaders(hdr)
	// Should contain base64 trimmed value not plain
	assert.NotContains(t, out, value)
	encoded := base64.StdEncoding.EncodeToString([]byte("trim"))
	assert.Contains(t, out, encoded)
}
