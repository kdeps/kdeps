package resolver

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

type sampleStruct struct {
	FieldA string
	FieldB int
}

func TestFormatValue_MiscTypes(t *testing.T) {
	// Map[string]interface{}
	m := map[string]interface{}{"k": "v"}
	out := formatValue(m)
	if !strings.Contains(out, "[\"k\"]") || !strings.Contains(out, "v") {
		t.Errorf("formatValue map missing expected content: %s", out)
	}

	// Nil pointer should render textual <nil>
	var ptr *sampleStruct
	if got := formatValue(ptr); !strings.Contains(got, "<nil>") {
		t.Errorf("expected output to contain <nil> for nil pointer, got %s", got)
	}

	// Struct pointer
	s := &sampleStruct{FieldA: "foo", FieldB: 42}
	out2 := formatValue(s)
	if !strings.Contains(out2, "FieldA") || !strings.Contains(out2, "foo") || !strings.Contains(out2, "42") {
		t.Errorf("formatValue struct output unexpected: %s", out2)
	}
}

func TestDecodeErrorMessage_Extra(t *testing.T) {
	orig := "hello world"
	enc := base64.StdEncoding.EncodeToString([]byte(orig))

	// base64 encoded
	if got := decodeErrorMessage(enc, logging.NewTestLogger()); got != orig {
		t.Errorf("expected decoded message %q, got %q", orig, got)
	}

	// plain string remains unchanged
	if got := decodeErrorMessage(orig, logging.NewTestLogger()); got != orig {
		t.Errorf("plain string should remain unchanged: got %q", got)
	}

	// empty string returns empty
	if got := decodeErrorMessage("", logging.NewTestLogger()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
