package utils_test

import (
	"encoding/base64"
	"strings"
	"testing"

	. "github.com/kdeps/kdeps/pkg/utils"
)

func TestEncodePklMap_Internal(t *testing.T) {
	m := map[string]string{"foo": "bar"}
	out := EncodePklMap(&m)
	if !strings.Contains(out, "foo") || !strings.Contains(out, base64.StdEncoding.EncodeToString([]byte("bar"))) {
		t.Fatalf("EncodePklMap did not encode as expected: %s", out)
	}
	if got := EncodePklMap(nil); got != "{}\n" {
		t.Fatalf("EncodePklMap(nil) = %q, want {}\\n", got)
	}
}

func TestEncodePklSlice_Internal(t *testing.T) {
	s := []string{"baz"}
	out := EncodePklSlice(&s)
	if !strings.Contains(out, base64.StdEncoding.EncodeToString([]byte("baz"))) {
		t.Fatalf("EncodePklSlice did not encode as expected: %s", out)
	}
	if got := EncodePklSlice(nil); got != "{}\n" {
		t.Fatalf("EncodePklSlice(nil) = %q, want {}\\n", got)
	}
}

func TestFormatRequestHeaders_Internal(t *testing.T) {
	headers := map[string][]string{"X": {"val"}}
	out := FormatRequestHeaders(headers)
	if !strings.Contains(out, "headers {") || !strings.Contains(out, "X") {
		t.Fatalf("FormatRequestHeaders missing header name: %s", out)
	}
	if !strings.Contains(out, base64.StdEncoding.EncodeToString([]byte("val"))) {
		t.Fatalf("FormatRequestHeaders missing encoded value: %s", out)
	}
}

func TestFormatRequestParams_Internal(t *testing.T) {
	params := map[string][]string{"q": {"search"}}
	out := FormatRequestParams(params)
	if !strings.Contains(out, "params {") || !strings.Contains(out, "q") {
		t.Fatalf("FormatRequestParams missing param name: %s", out)
	}
	if !strings.Contains(out, base64.StdEncoding.EncodeToString([]byte("search"))) {
		t.Fatalf("FormatRequestParams missing encoded value: %s", out)
	}
}

func TestFormatResponseHeaders_Internal(t *testing.T) {
	h := map[string]string{"Content-Type": "application/json"}
	out := FormatResponseHeaders(h)
	if !strings.Contains(out, "Content-Type") || !strings.Contains(out, "application/json") {
		t.Fatalf("FormatResponseHeaders missing content: %s", out)
	}
}

func TestFormatResponseProperties_Internal(t *testing.T) {
	props := map[string]string{"status": "ok"}
	out := FormatResponseProperties(props)
	if !strings.Contains(out, "status") || !strings.Contains(out, "ok") {
		t.Fatalf("FormatResponseProperties missing content: %s", out)
	}
}
