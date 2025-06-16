package docker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
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
