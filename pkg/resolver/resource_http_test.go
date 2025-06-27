package resolver

import (
	"testing"

	pklHTTP "github.com/kdeps/schema/gen/http"
	"github.com/stretchr/testify/assert"
)

func TestIsMethodWithBody(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"POST", true},
		{"PUT", true},
		{"PATCH", true},
		{"DELETE", true},
		{"GET", false},
		{"HEAD", false},
		{"OPTIONS", false},
		{"post", true}, // Test case insensitive
		{"get", false}, // Test case insensitive
		{"CONNECT", false},
		{"TRACE", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := isMethodWithBody(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeResponseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		response *pklHTTP.ResponseBlock
		expected string
		contains []string // For tests where order doesn't matter
	}{
		{
			name:     "nil response",
			response: nil,
			expected: "    headers {[\"\"] = \"\"}\n",
		},
		{
			name:     "response with nil headers",
			response: &pklHTTP.ResponseBlock{Headers: nil},
			expected: "    headers {[\"\"] = \"\"}\n",
		},
		{
			name: "response with headers",
			response: &pklHTTP.ResponseBlock{
				Headers: &map[string]string{
					"Content-Type": "application/json",
					"X-Custom":     "test-value",
				},
			},
			// Since map iteration order is non-deterministic, check for individual parts
			contains: []string{
				"    headers {\n",
				"[\"Content-Type\"] = #\"\"\"\nYXBwbGljYXRpb24vanNvbg==\n\"\"\"#",
				"[\"X-Custom\"] = #\"\"\"\ndGVzdC12YWx1ZQ==\n\"\"\"#",
				"    }\n",
			},
		},
		{
			name: "response with empty headers map",
			response: &pklHTTP.ResponseBlock{
				Headers: &map[string]string{},
			},
			expected: "    headers {\n    }\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeResponseHeaders(tt.response)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else if len(tt.contains) > 0 {
				for _, part := range tt.contains {
					assert.Contains(t, result, part)
				}
			}
		})
	}
}

func TestEncodeResponseBody(t *testing.T) {
	tests := []struct {
		name     string
		response *pklHTTP.ResponseBlock
		expected string
	}{
		{
			name:     "nil response",
			response: nil,
			expected: "    body=\"\"\n",
		},
		{
			name:     "response with nil body",
			response: &pklHTTP.ResponseBlock{Body: nil},
			expected: "    body=\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeResponseBody(tt.response, nil, "test-resource")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
