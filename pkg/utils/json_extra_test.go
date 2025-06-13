package utils_test

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestFixJSON_EscapesAndWhitespace(t *testing.T) {
	// Contains newline inside quoted value and stray unescaped quote.
	input := "{\n  \"msg\": \"Hello\nWorld\",\n  \"quote\": \"She said \"Hi\"\"\n}"
	expected := "{\n\"msg\": \"Hello\\nWorld\",\n\"quote\": \"She said \\\"Hi\\\"\"\n}"

	if got := utils.FixJSON(input); got != expected {
		t.Errorf("FixJSON mismatch\nwant: %s\n got: %s", expected, got)
	}
}
