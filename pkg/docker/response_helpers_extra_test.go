package docker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"encoding/base64"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestValidateMethodSimple(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	methodStr, err := validateMethod(req, []string{"GET", "POST"})
	if err != nil {
		t.Fatalf("validateMethod unexpected error: %v", err)
	}
	if methodStr != `method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// Unsupported method should error
	req.Method = "DELETE"
	if _, err := validateMethod(req, []string{"GET", "POST"}); err == nil {
		t.Fatalf("expected error for unsupported method")
	}
}

func TestCleanOldFilesMem(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Prepare dependency resolver stub with in-mem fs
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logger, ResponseTargetFile: "/tmp/old_resp.txt"}
	// Create dummy file
	afero.WriteFile(fs, dr.ResponseTargetFile, []byte("old"), 0o666)

	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error: %v", err)
	}
	if exists, _ := afero.Exists(fs, dr.ResponseTargetFile); exists {
		t.Fatalf("file still exists after cleanOldFiles")
	}
}

func TestDecodeAndFormatResponseSimple(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build sample APIResponse JSON with base64 encoded data
	sample := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{utils.EncodeBase64String(`{"foo":"bar"}`)}},
		Meta:     ResponseMeta{RequestID: "abc123"},
	}
	raw, _ := json.Marshal(sample)

	decoded, err := decodeResponseContent(raw, logger)
	if err != nil {
		t.Fatalf("decodeResponseContent error: %v", err)
	}
	if len(decoded.Response.Data) != 1 || decoded.Response.Data[0] != "{\n  \"foo\": \"bar\"\n}" {
		t.Fatalf("decodeResponseContent did not prettify JSON: %v", decoded.Response.Data)
	}

	// Marshal decoded struct then format
	marshaled, _ := json.Marshal(decoded)
	formatted := formatResponseJSON(marshaled)
	if !bytes.Contains(formatted, []byte("foo")) {
		t.Fatalf("formatResponseJSON missing field")
	}
}

func TestDecodeResponseContent_Success(t *testing.T) {
	logger := logging.NewTestLogger()

	// Prepare an APIResponse JSON with base64-encoded JSON payload in data.
	inner := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(inner))

	raw := APIResponse{
		Success: true,
		Response: ResponseData{
			Data: []string{encoded},
		},
		Meta: ResponseMeta{
			RequestID: "abc",
		},
	}

	rawBytes, err := json.Marshal(raw)
	assert.NoError(t, err)

	decoded, err := decodeResponseContent(rawBytes, logger)
	assert.NoError(t, err)
	assert.Equal(t, "abc", decoded.Meta.RequestID)
	assert.Contains(t, decoded.Response.Data[0], "\"hello\": \"world\"")
}

func TestDecodeResponseContent_InvalidJSON(t *testing.T) {
	logger := logging.NewTestLogger()
	_, err := decodeResponseContent([]byte(`not-json`), logger)
	assert.Error(t, err)
}

func TestFormatResponseJSONPretty(t *testing.T) {
	// Create a response that will be decodable by formatResponseJSON
	inner := map[string]string{"foo": "bar"}
	innerBytes, _ := json.Marshal(inner)

	resp := map[string]interface{}{
		"response": map[string]interface{}{
			"data": []interface{}{string(innerBytes)},
		},
	}
	bytesIn, _ := json.Marshal(resp)

	pretty := formatResponseJSON(bytesIn)

	// The formatted JSON should contain nested object without quotes around keys
	assert.Contains(t, string(pretty), "\"foo\": \"bar\"")
}
