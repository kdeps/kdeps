package utils

import (
	"testing"
)

func TestIsJSON(t *testing.T) {

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid JSON object",
			input: `{"name": "John", "age": 30}`,
			want:  true,
		},
		{
			name:  "valid JSON array",
			input: `["apple", "banana"]`,
			want:  true,
		},
		{
			name:  "invalid JSON missing brace",
			input: `{"name": "John", "age": 30`,
			want:  false,
		},
		{
			name:  "invalid JSON unquoted key",
			input: `{name: "John"}`,
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "string with unescaped quotes in value",
			input: `{"message": "Hello "world""}`,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := IsJSON(tt.input)
			if got != tt.want {
				t.Errorf("IsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFixJSON(t *testing.T) {

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unescaped quotes in value",
			input: `"message": "He said "Hello!""`,
			want:  `"message": "He said \"Hello!\""`,
		},
		{
			name:  "multiple lines with fixes",
			input: "{\n  \"name\": \"Alice \"Bob\"\",\n  \"age\": \"30\"\n}",
			want:  "{\n\"name\": \"Alice \\\"Bob\\\"\",\n\"age\": \"30\"\n}",
		},
		{
			name:  "value without quotes",
			input: `"number": 42`,
			want:  `"number": 42`,
		},
		{
			name:  "line with trailing comma",
			input: `"city": "New "York",`,
			want:  `"city": "New \"York",`,
		},
		{
			name:  "already escaped quotes",
			input: `"quote": "He said \"Hello!\""`,
			want:  `"quote": "He said \\"Hello!\\""`,
		},
		{
			name:  "no changes needed",
			input: `"valid": "json"`,
			want:  `"valid": "json"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := FixJSON(tt.input)
			if got != tt.want {
				t.Errorf("FixJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
