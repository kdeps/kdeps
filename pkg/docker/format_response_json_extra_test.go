package docker

import (
	"bytes"
	"testing"
)

// TestFormatResponseJSONInlineData ensures that when the "data" field contains
// string elements that are themselves valid JSON objects, formatResponseJSON
// converts those elements into embedded objects within the final JSON.
func TestFormatResponseJSONInlineData(t *testing.T) {
	raw := []byte(`{"response": {"data": ["{\"foo\": \"bar\"}", "plain text"]}}`)

	pretty := formatResponseJSON(raw)

	if !bytes.Contains(pretty, []byte("\"foo\": \"bar\"")) {
		t.Fatalf("expected pretty JSON to contain inlined object, got %s", string(pretty))
	}
}
