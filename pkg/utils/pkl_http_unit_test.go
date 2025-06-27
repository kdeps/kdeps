package utils_test

import (
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestFormatRequestAndResponseHelpers(t *testing.T) {
	hdrs := map[string][]string{"X-Token": {"abc123"}}
	out := utils.FormatRequestHeaders(hdrs)
	if !strings.Contains(out, "headers") {
		t.Fatalf("expected headers block, got %s", out)
	}

	params := map[string][]string{"q": {"search"}}
	p := utils.FormatRequestParams(params)
	if !strings.Contains(p, "params") {
		t.Fatalf("expected params block")
	}

	rh := map[string]string{"Content-Type": "application/json"}
	resp := utils.FormatResponseHeaders(rh)
	if !strings.Contains(resp, "headers") {
		t.Fatalf("expected response headers block")
	}
}
