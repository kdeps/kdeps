package docker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestValidateMethodExtra2(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	methodStr, err := validateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if methodStr != `method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// invalid method
	badReq := httptest.NewRequest("DELETE", "/", nil)
	if _, err := validateMethod(badReq, []string{"GET"}); err == nil {
		t.Fatalf("expected error for disallowed method")
	}
}

func TestCleanOldFilesExtra2(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create dummy response file
	path := "/tmp/resp.json"
	afero.WriteFile(fs, path, []byte("dummy"), 0o644)

	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		Logger:             logger,
		ResponseTargetFile: path,
	}

	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, _ := afero.Exists(fs, path)
	if exists {
		t.Fatalf("file should have been removed")
	}
}

func TestDecodeResponseContentExtra2(t *testing.T) {
	logger := logging.NewTestLogger()

	// prepare APIResponse with base64 encoded JSON data
	apiResp := APIResponse{
		Success: true,
		Response: ResponseData{
			Data: []string{base64.StdEncoding.EncodeToString([]byte(`{"foo":"bar"}`))},
		},
		Meta: ResponseMeta{
			Headers: map[string]string{"X-Test": "yes"},
		},
	}
	encoded, _ := json.Marshal(apiResp)

	decResp, err := decodeResponseContent(encoded, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(decResp.Response.Data) != 1 || decResp.Response.Data[0] != "{\n  \"foo\": \"bar\"\n}" {
		t.Fatalf("unexpected decoded data: %+v", decResp.Response.Data)
	}
}

func TestFormatResponseJSONExtra2(t *testing.T) {
	// Response with data as JSON string
	raw := []byte(`{"response":{"data":["{\"a\":1}"]}}`)
	pretty := formatResponseJSON(raw)

	// Should be pretty printed and data element should be object not string
	if !bytes.Contains(pretty, []byte("\"a\": 1")) {
		t.Fatalf("pretty output missing expected content: %s", string(pretty))
	}
}

func TestFormatResponseJSONExtra(t *testing.T) {
	// Prepare response with data that is itself JSON string
	inner := map[string]any{"foo": "bar"}
	innerBytes, _ := json.Marshal(inner)
	resp := map[string]any{
		"response": map[string]any{
			"data": []string{string(innerBytes)},
		},
	}
	raw, _ := json.Marshal(resp)
	pretty := formatResponseJSON(raw)

	// It should now be pretty-printed and contain nested object without quotes
	require.Contains(t, string(pretty), "\"foo\": \"bar\"")
}

func TestCleanOldFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: "old.json"}

	// Case where file exists
	require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0644))
	require.NoError(t, cleanOldFiles(dr))
	exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
	require.False(t, exists)

	// Case where file does not exist should be no-op
	require.NoError(t, cleanOldFiles(dr))
}

func TestDecodeResponseContentExtra(t *testing.T) {
	// Prepare APIResponse JSON with Base64 encoded data
	dataJSON := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(dataJSON))
	respStruct := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{encoded}},
	}
	raw, _ := json.Marshal(respStruct)

	logger := logging.NewTestLogger()
	out, err := decodeResponseContent(raw, logger)
	require.NoError(t, err)
	require.Len(t, out.Response.Data, 1)
	require.JSONEq(t, dataJSON, out.Response.Data[0])
}
