package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestFormatRequestHeadersAndParamsExtra(t *testing.T) {
	headers := map[string][]string{
		"X-Test": {" value1 ", "value2"},
	}
	params := map[string][]string{
		"query": {"foo", " bar "},
	}

	// Exercise helpers
	hdrOut := FormatRequestHeaders(headers)
	prmOut := FormatRequestParams(params)

	// Basic structural checks
	if !strings.HasPrefix(hdrOut, "headers {") || !strings.HasSuffix(hdrOut, "}") {
		t.Fatalf("unexpected headers formatting: %q", hdrOut)
	}
	if !strings.HasPrefix(prmOut, "params {") || !strings.HasSuffix(prmOut, "}") {
		t.Fatalf("unexpected params formatting: %q", prmOut)
	}

	// Verify that each value is Base64-encoded and trimmed
	encodedVal := base64.StdEncoding.EncodeToString([]byte("value1"))
	if !strings.Contains(hdrOut, encodedVal) {
		t.Errorf("expected encoded header value %q in %q", encodedVal, hdrOut)
	}
	encodedVal = base64.StdEncoding.EncodeToString([]byte("bar"))
	if !strings.Contains(prmOut, encodedVal) {
		t.Errorf("expected encoded param value %q in %q", encodedVal, prmOut)
	}
}

func TestFormatResponseHeadersAndPropertiesExtra(t *testing.T) {
	headers := map[string]string{"Content-Type": " application/json "}
	props := map[string]string{"status": " ok "}

	hdrOut := FormatResponseHeaders(headers)
	propOut := FormatResponseProperties(props)

	if !strings.Contains(hdrOut, `["Content-Type"] = "application/json"`) {
		t.Errorf("unexpected response headers output: %q", hdrOut)
	}
	if !strings.Contains(propOut, `["status"] = "ok"`) {
		t.Errorf("unexpected response properties output: %q", propOut)
	}
}
