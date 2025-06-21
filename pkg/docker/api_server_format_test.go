package docker_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
)

func TestFormatResponseJSON_Unit(t *testing.T) {
	// Input with a JSON-encoded string inside data array.
	raw := map[string]interface{}{
		"response": map[string]interface{}{
			"data": []string{`{"foo":"bar"}`},
		},
	}
	b, _ := json.Marshal(raw)

	formatted := docker.FormatResponseJSON(b)
	var out map[string]interface{}
	if err := json.Unmarshal(formatted, &out); err != nil {
		t.Fatalf("formatted content is not valid JSON: %v", err)
	}
	// Assert data was converted to an object.
	resp := out["response"].(map[string]interface{})
	first := resp["data"].([]interface{})[0].(map[string]interface{})
	if first["foo"] != "bar" {
		t.Errorf("expected inner JSON to be decoded, got %v", first)
	}

	// When content is not JSON, function should return original bytes.
	nonJSON := []byte("not-json")
	if got := docker.FormatResponseJSON(nonJSON); string(got) != string(nonJSON) {
		t.Errorf("non-JSON input should be unchanged")
	}
}

func TestDecodeResponseContent_Unit(t *testing.T) {
	logger := logging.NewTestLogger()

	// Prepare APIResponse with Base64 encoded data
	innerJSON := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(innerJSON))
	resp := docker.APIResponse{
		Response: docker.ResponseData{Data: []string{encoded}},
		Meta:     docker.ResponseMeta{},
	}
	b, _ := json.Marshal(resp)

	decoded, err := docker.DecodeResponseContent(b, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	str := decoded.Response.Data[0]
	if !containsJSON(str, "\"hello\"") {
		t.Errorf("decoded data does not contain expected content: %s", str)
	}
}

// containsJSON is a tiny helper to check substring while ignoring whitespace.
func containsJSON(s, substr string) bool {
	return len(s) >= len(substr) && jsonValidSubstring(s, substr)
}

// jsonValidSubstring checks containment without caring about whitespace.
func jsonValidSubstring(s, substr string) bool {
	i := 0
	for j := 0; j < len(s) && i < len(substr); j++ {
		if s[j] == substr[i] {
			i++
		}
	}
	return i == len(substr)
}
