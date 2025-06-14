package resolver

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
)

func TestFormatDataValue(t *testing.T) {
	// Simple string value should embed JSONRenderDocument lines
	out := formatDataValue("hello")
	if !strings.Contains(out, "JSONRenderDocument") {
		t.Errorf("expected JSONRenderDocument in output, got %s", out)
	}

	// Map value path should still produce block
	m := map[string]interface{}{"k": "v"}
	out2 := formatDataValue(m)
	if !strings.Contains(out2, "k") {
		t.Errorf("map key lost in formatting: %s", out2)
	}
}

func TestFormatErrorsMultiple(t *testing.T) {
	logger := logging.NewTestLogger()
	msg := base64.StdEncoding.EncodeToString([]byte("decoded msg"))
	errorsSlice := &[]*apiserverresponse.APIServerErrorsBlock{
		{Code: 400, Message: "bad"},
		{Code: 500, Message: msg},
	}
	out := formatErrors(errorsSlice, logger)
	if !strings.Contains(out, "code = 400") || !strings.Contains(out, "code = 500") {
		t.Errorf("codes missing: %s", out)
	}
	if !strings.Contains(out, "decoded msg") {
		t.Errorf("base64 not decoded: %s", out)
	}
}
