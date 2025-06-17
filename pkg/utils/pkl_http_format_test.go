package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestPKLHTTPFormattersAdditional(t *testing.T) {
	headers := map[string][]string{"X-Test": {" value "}}
	hStr := FormatRequestHeaders(headers)
	if !strings.Contains(hStr, "X-Test") {
		t.Fatalf("header name missing in output")
	}
	encoded := base64.StdEncoding.EncodeToString([]byte("value"))
	if !strings.Contains(hStr, encoded) {
		t.Fatalf("encoded value missing in output; got %s", hStr)
	}

	params := map[string][]string{"q": {"k &v"}}
	pStr := FormatRequestParams(params)
	encodedParam := base64.StdEncoding.EncodeToString([]byte("k &v"))
	if !strings.Contains(pStr, "q") || !strings.Contains(pStr, encodedParam) {
		t.Fatalf("param formatting incorrect: %s", pStr)
	}

	respHeaders := map[string]string{"Content-Type": "application/json"}
	rhStr := FormatResponseHeaders(respHeaders)
	if !strings.Contains(rhStr, "Content-Type") {
		t.Fatalf("response header missing")
	}

	props := map[string]string{"prop": "123"}
	propStr := FormatResponseProperties(props)
	if !strings.Contains(propStr, "prop") {
		t.Fatalf("response prop missing")
	}
}
