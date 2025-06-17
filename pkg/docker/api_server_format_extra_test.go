package docker

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestFormatResponseJSONFormatTest(t *testing.T) {
	// Input where first element is JSON string and second is plain string.
	in := []byte(`{"response":{"data":["{\"x\":1}","plain"]}}`)
	out := formatResponseJSON(in)
	// The output should still be valid JSON and contain "x": 1 without escaped quotes.
	if !json.Valid(out) {
		t.Fatalf("output not valid JSON: %s", string(out))
	}
	if !bytes.Contains(out, []byte("\"x\": 1")) {
		t.Fatalf("expected object conversion in data array, got %s", string(out))
	}
}
