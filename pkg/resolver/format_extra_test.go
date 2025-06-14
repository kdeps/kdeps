package resolver

import (
	"reflect"
	"strings"
	"testing"
)

func TestFormatMapSimple(t *testing.T) {
	m := map[interface{}]interface{}{
		"foo": "bar",
		1:     2,
	}
	out := formatMap(m)
	if !containsAll(out, []string{"new Mapping {", "[\"foo\"]", "[\"1\"] ="}) {
		t.Errorf("unexpected mapping output: %s", out)
	}
}

// Helper to check substring presence
func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestFormatValueVariants(t *testing.T) {
	// Case 1: nil interface -> "null"
	var v interface{} = nil
	if out := formatValue(v); out != "null" {
		t.Errorf("expected 'null' for nil, got %s", out)
	}

	// Case 2: map[string]interface{}
	mp := map[string]interface{}{"k": "v"}
	mv := formatValue(mp)
	if !strings.Contains(mv, "new Mapping {") || !strings.Contains(mv, "[\"k\"]") {
		t.Errorf("unexpected map formatting: %s", mv)
	}

	// Case 3: pointer to struct -> should format struct fields via Mapping
	type sample struct{ Field string }
	s := &sample{Field: "data"}
	sv := formatValue(s)
	if !strings.Contains(sv, "Field") || !strings.Contains(sv, "data") {
		t.Errorf("struct pointer formatting missing content: %s", sv)
	}

	// Case 4: direct struct (non-pointer)
	sp := sample{Field: "x"}
	st := formatValue(sp)
	if !strings.Contains(st, "Field") {
		t.Errorf("struct formatting unexpected: %s", st)
	}

	// Ensure default path returns triple-quoted string for primitive
	prim := formatValue("plain")
	if !strings.Contains(prim, "\"\"\"") {
		t.Errorf("primitive formatting not triple-quoted: %s", prim)
	}

	// Sanity: reflect-based call shouldn't panic for pointer nil
	var nilPtr *sample
	_ = formatValue(nilPtr)
	// the return is acceptable, we just ensure no panic
	_ = reflect.TypeOf(nilPtr)
}
