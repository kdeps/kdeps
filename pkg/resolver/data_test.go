package resolver_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
)

// MockContext implements context.Context.
type MockContext struct{}

func (mc *MockContext) Deadline() (time.Time, bool) {
	return time.Now(), false
}

func (mc *MockContext) Done() <-chan struct{} {
	return nil
}

func (mc *MockContext) Err() error {
	return nil
}

func (mc *MockContext) Value(_ interface{}) interface{} {
	return nil
}

func TestFormatDataValue(t *testing.T) {
	// Simple string value should embed jsonRenderDocument lines
	out := resolver.FormatDataValue("hello")
	if !strings.Contains(out, "jsonRenderDocument") {
		t.Errorf("expected jsonRenderDocument in output, got %s", out)
	}

	// Map value path should still produce block
	m := map[string]interface{}{"k": "v"}
	out2 := resolver.FormatDataValue(m)
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
	out := resolver.FormatErrors(errorsSlice, logger)
	if !strings.Contains(out, "Code = 400") || !strings.Contains(out, "Code = 500") {
		t.Errorf("codes missing: %s", out)
	}
	if !strings.Contains(out, "decoded msg") {
		t.Errorf("base64 not decoded: %s", out)
	}
}

// TestFormatValueVariantsBasic exercises several branches of the reflection-based
// formatValue helper to bump coverage and guard against panics when handling
// diverse inputs.
func TestFormatValueVariantsBasic(t *testing.T) {
	type custom struct{ X string }

	variants := []interface{}{
		nil,
		map[string]interface{}{"k": "v"},
		custom{X: "val"},
	}

	for _, v := range variants {
		out := resolver.FormatValue(v)
		if out == "" {
			t.Errorf("formatValue produced empty output for %+v", v)
		}
	}
}
