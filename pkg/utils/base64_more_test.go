package utils_test

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestIsBase64Encoded_InvalidChar(t *testing.T) {
	str := "abcd#==" // '#' invalid
	if utils.IsBase64Encoded(str) {
		t.Errorf("expected false for string with invalid char")
	}
}
