package utils_test

import "testing"

func TestFormatRequestAndResponseHelpers(t *testing.T) {
	hdrs := map[string][]string{"X-Token": {"abc123"}}
	out := FormatRequestHeaders(hdrs)
	if !contains(out, "headers") {
		t.Fatalf("expected headers block, got %s", out)
	}

	params := map[string][]string{"q": {"search"}}
	p := FormatRequestParams(params)
	if !contains(p, "params") {
		t.Fatalf("expected params block")
	}

	rh := map[string]string{"Content-Type": "application/json"}
	resp := FormatResponseHeaders(rh)
	if !contains(resp, "headers") {
		t.Fatalf("expected response headers block")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[0:len(sub)] == sub || contains(s[1:], sub)))
}
