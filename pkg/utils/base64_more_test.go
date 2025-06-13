package utils

import (
	"testing"
)

func TestIsBase64Encoded_InvalidChar(t *testing.T) {
	str := "abcd#==" // '#' invalid
	if IsBase64Encoded(str) {
		t.Errorf("expected false for string with invalid char")
	}
}
