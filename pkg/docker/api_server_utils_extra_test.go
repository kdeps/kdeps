package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
)

// Ensure schema version gets referenced at least once in this test file.
func TestSchemaVersionReference(t *testing.T) {
	if v := schema.SchemaVersion(context.Background()); v == "" {
		t.Fatalf("SchemaVersion returned empty string")
	}
}

func TestValidateMethodUtilsExtra(t *testing.T) {
	_ = schema.SchemaVersion(nil)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	got, err := validateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil || got != `method = "GET"` {
		t.Fatalf("expected valid GET, got %q err %v", got, err)
	}

	reqEmpty, _ := http.NewRequest("", "http://example.com", nil)
	got2, err2 := validateMethod(reqEmpty, []string{http.MethodGet})
	if err2 != nil || got2 != `method = "GET"` {
		t.Fatalf("default method failed: %q err %v", got2, err2)
	}

	reqBad, _ := http.NewRequest(http.MethodDelete, "http://example.com", nil)
	if _, err := validateMethod(reqBad, []string{http.MethodGet}); err == nil {
		t.Fatalf("expected error for disallowed method")
	}
}

func TestDecodeResponseContentUtilsExtra(t *testing.T) {
	_ = schema.SchemaVersion(nil)

	helloB64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	invalidB64 := "@@invalid@@"
	raw := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{helloB64, invalidB64}},
		Meta:     ResponseMeta{RequestID: "abc"},
	}
	data, _ := json.Marshal(raw)
	logger := logging.NewTestLogger()
	decoded, err := decodeResponseContent(data, logger)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if decoded.Response.Data[0] != "hello" {
		t.Fatalf("expected \"hello\", got %q", decoded.Response.Data[0])
	}
	if decoded.Response.Data[1] != invalidB64 {
		t.Fatalf("invalid data should remain unchanged")
	}
}

func TestDecodeResponseContentFormattingUtilsExtra(t *testing.T) {
	jsonPayload := `{"foo":"bar"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(jsonPayload))

	resp := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{encoded}},
		Meta:     ResponseMeta{Headers: map[string]string{"X-Test": "1"}},
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	logger := logging.NewTestLogger()
	decoded, err := decodeResponseContent(raw, logger)
	if err != nil {
		t.Fatalf("decodeResponseContent error: %v", err)
	}

	if len(decoded.Response.Data) != 1 {
		t.Fatalf("expected 1 data entry, got %d", len(decoded.Response.Data))
	}

	first := decoded.Response.Data[0]
	if !bytes.Contains([]byte(first), []byte("foo")) || !bytes.Contains([]byte(first), []byte("bar")) {
		t.Fatalf("decoded data does not contain expected JSON: %s", first)
	}

	if first == encoded {
		t.Fatalf("base64 string not decoded")
	}
}
