package utils

import "testing"

func TestIsBase64Encoded_EdgeCasesAdditional(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},                         // empty
		{"abc", false},                      // length not multiple of 4
		{"@@@@", false},                     // invalid chars
		{EncodeBase64String("hello"), true}, // valid
	}
	for _, tt := range tests {
		got := IsBase64Encoded(tt.in)
		if got != tt.want {
			t.Fatalf("IsBase64Encoded(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
