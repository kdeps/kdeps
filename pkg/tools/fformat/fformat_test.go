package fformat

import (
	"testing"
)

func TestValidateString_JSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid object", `{"key": "value"}`, true},
		{"valid array", `[1, 2, 3]`, true},
		{"valid string", `"hello"`, true},
		{"valid number", `42`, true},
		{"invalid json", `{bad}`, false},
		{"empty", ``, false},
		{"trailing comma", `{"a": 1,}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ValidateString(tt.input, JSON)
			if r.Valid != tt.valid {
				t.Errorf("ValidateString() valid = %v, want %v (error: %s)", r.Valid, tt.valid, r.Error)
			}
		})
	}
}

func TestValidateString_YAML(t *testing.T) {
	r := ValidateString("key: value", YAML)
	if !r.Valid {
		t.Errorf("expected valid YAML, got: %s", r.Error)
	}

	r = ValidateString(": bad yaml", YAML)
	if r.Valid {
		t.Error("expected invalid YAML")
	}
}

func TestValidateString_CSV(t *testing.T) {
	r := ValidateString("a,b,c\n1,2,3", CSV)
	if !r.Valid {
		t.Errorf("expected valid CSV, got: %s", r.Error)
	}

	r = ValidateString("a,\"unclosed", CSV)
	if r.Valid {
		t.Error("expected invalid CSV")
	}
}

func TestValidateString_XML(t *testing.T) {
	r := ValidateString("<root><child/></root>", XML)
	if !r.Valid {
		t.Errorf("expected valid XML, got: %s", r.Error)
	}

	r = ValidateString("<open>", XML)
	if r.Valid {
		t.Error("expected invalid XML")
	}
}

func TestValidateString_PassThrough(t *testing.T) {
	r := ValidateString("anything", Markdown)
	if !r.Valid {
		t.Error("Markdown validation should pass through")
	}
}

func TestFormatString_JSON(t *testing.T) {
	r := FormatString(`{"b":2,"a":1}`, JSON)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
	if r.Output == "" {
		t.Error("expected formatted output")
	}
}

func TestFormatString_YAML(t *testing.T) {
	r := FormatString("key: value", YAML)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestFormatString_Invalid(t *testing.T) {
	r := FormatString("{bad", JSON)
	if r.Valid {
		t.Error("expected invalid")
	}
}

func TestConvertToJSON_YAML(t *testing.T) {
	r := ConvertToJSON(YAML, "key: value")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestConvertToJSON_CSV(t *testing.T) {
	r := ConvertToJSON(CSV, "name,age\nAlice,30\nBob,25")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestConvertToJSON_CSV_HeaderOnly(t *testing.T) {
	r := ConvertToJSON(CSV, "name,age")
	if r.Valid {
		t.Error("expected error for header-only CSV")
	}
}

func TestConvertToJSON_Unsupported(t *testing.T) {
	r := ConvertToJSON(Markdown, "# Title")
	if r.Valid {
		t.Error("expected error for unsupported conversion")
	}
}

func TestConvertFromJSON_YAML(t *testing.T) {
	r := ConvertFromJSON(YAML, `{"key": "value"}`)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestConvertFromJSON_CSV(t *testing.T) {
	r := ConvertFromJSON(CSV, `[{"name": "Alice", "age": "30"}]`)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestConvertFromJSON_CSV_Empty(t *testing.T) {
	r := ConvertFromJSON(CSV, `[]`)
	if r.Valid {
		t.Error("expected error for empty JSON array")
	}
}

func TestConvertFromJSON_Unsupported(t *testing.T) {
	r := ConvertFromJSON(XML, `{"key": "value"}`)
	if r.Valid {
		t.Error("expected error for unsupported conversion")
	}
}

func TestConvertToJSON_XML(t *testing.T) {
	r := ConvertToJSON(XML, "<root><item key=\"val\">text</item></root>")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestNormalizeForJSON(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{"string map", map[string]interface{}{"key": "value"}},
		{"iface map", map[interface{}]interface{}{"key": "value"}},
		{"slice", []interface{}{map[interface{}]interface{}{"nested": "val"}}},
		{"scalar", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeForJSON(tt.input)
			if result == nil {
				t.Error("normalizeForJSON returned nil")
			}
		})
	}
}
