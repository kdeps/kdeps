package utils

import "testing"

func TestFormatRequestAndResponseHelpers(t *testing.T) {
	hdrs := map[string][]string{"X-Token": {"abc123"}}
	out := FormatRequestHeaders(hdrs)
	if !contains(out, "Headers") {
		t.Fatalf("expected Headers block, got %s", out)
	}

	params := map[string][]string{"q": {"search"}}
	p := FormatRequestParams(params)
	if !contains(p, "Params") {
		t.Fatalf("expected Params block")
	}

	rh := map[string]string{"Content-Type": "application/json"}
	resp := FormatResponseHeaders(rh)
	if !containsSubstring(resp, "Headers") {
		t.Fatalf("expected response Headers block")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[0:len(sub)] == sub || containsSubstring(s[1:], sub)))
}
