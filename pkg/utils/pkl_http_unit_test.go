package utils_test

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestFormatRequestAndResponseHelpers(t *testing.T) {
	hdrs := map[string][]string{"X-Token": {"abc123"}}
	out := utils.FormatRequestHeaders(hdrs)
	if !containsHelper(out, "Headers") {
		t.Fatalf("expected Headers block, got %s", out)
	}

	params := map[string][]string{"q": {"search"}}
	p := utils.FormatRequestParams(params)
	if !containsHelper(p, "Params") {
		t.Fatalf("expected Params block")
	}

	rh := map[string]string{"Content-Type": "application/json"}
	resp := utils.FormatResponseHeaders(rh)
	if !containsHelper(resp, "Headers") {
		t.Fatalf("expected response Headers block")
	}
}

func containsHelper(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[0:len(sub)] == sub || containsHelper(s[1:], sub)))
}
