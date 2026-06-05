package fformat

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
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

func TestValidateString_TOML(t *testing.T) {
	r := ValidateString("key = \"value\"\n[section]\nport = 8080", TOML)
	if !r.Valid {
		t.Errorf("expected valid TOML, got: %s", r.Error)
	}
	r = ValidateString("", TOML)
	if r.Valid {
		t.Error("expected invalid for empty TOML")
	}
}

func TestValidateString_Markdown(t *testing.T) {
	r := ValidateString("# Title\n\nSome text.", Markdown)
	if !r.Valid {
		t.Errorf("expected valid Markdown, got: %s", r.Error)
	}
	r = ValidateString("", Markdown)
	if r.Valid {
		t.Error("expected invalid for empty Markdown")
	}
}

func TestValidateString_SQL(t *testing.T) {
	r := ValidateString("SELECT * FROM users WHERE id = 1", SQL)
	if !r.Valid {
		t.Errorf("expected valid SQL, got: %s", r.Error)
	}
	r = ValidateString("not sql at all", SQL)
	if r.Valid {
		t.Error("expected invalid SQL")
	}
	r = ValidateString("", SQL)
	if r.Valid {
		t.Error("expected invalid for empty SQL")
	}
}

func TestValidateString_HTML(t *testing.T) {
	r := ValidateString("<html><body><p>Hello</p></body></html>", HTML)
	if !r.Valid {
		t.Errorf("expected valid HTML, got: %s", r.Error)
	}
	r = ValidateString("", HTML)
	if r.Valid {
		t.Error("expected invalid for empty HTML")
	}
}

func TestFormatString_TOML(t *testing.T) {
	r := FormatString("key = \"value\"\n\n\n[section]\nport = 8080", TOML)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
	if r.Output == "" {
		t.Error("expected formatted output")
	}
}

func TestFormatString_SQL(t *testing.T) {
	r := FormatString("SELECT * FROM users WHERE id = 1", SQL)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestFormatString_HTML(t *testing.T) {
	r := FormatString("<html><body><p>Hello</p></body></html>", HTML)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
	if r.Output == "" {
		t.Error("expected formatted output")
	}
}

func TestConvertToJSON_TOML(t *testing.T) {
	r := ConvertToJSON(TOML, "name = \"Alice\"\nage = \"30\"\n[config]\nport = \"8080\"")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
	if r.Output == "" {
		t.Error("expected JSON output")
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

func TestFormatString_XML(t *testing.T) {
	// Happy path
	r := FormatString("<root><a>text</a></root>", XML)
	if !r.Valid {
		t.Fatalf("expected valid XML, got: %s", r.Error)
	}
	if r.Output == "" {
		t.Error("expected formatted XML output")
	}
	// Parse error (malformed XML)
	r = FormatString("<root><", XML)
	if r.Valid {
		t.Error("expected invalid for malformed XML")
	}
}

func TestFormatString_CSV(t *testing.T) {
	r := FormatString("a,b\n1,2", CSV)
	if r.Output != "a,b\n1,2" {
		t.Errorf("expected identity passthrough, got: %s", r.Output)
	}
}

func TestFormatString_Markdown(t *testing.T) {
	r := FormatString("# Title\n\nbody", Markdown)
	if r.Output != "# Title\n\nbody" {
		t.Errorf("expected identity passthrough, got: %s", r.Output)
	}
}

func TestFormatString_Default(t *testing.T) {
	r := FormatString("anything", Format("unknown"))
	if r.Output != "anything" {
		t.Errorf("expected identity passthrough, got: %s", r.Output)
	}
}

func TestFormatString_HTML_Invalid(t *testing.T) {
	r := FormatString("", HTML)
	if r.Valid {
		t.Error("expected invalid for empty HTML")
	}
	if r.Error == "" {
		t.Error("expected error message for empty HTML")
	}
}

func TestFormatString_YAML_Invalid(t *testing.T) {
	r := FormatString(": bad yaml", YAML)
	if r.Valid {
		t.Error("expected invalid YAML")
	}
	if r.Error == "" {
		t.Error("expected error message for bad YAML")
	}
}

func TestFormatString_SQL_Invalid(t *testing.T) {
	r := FormatString("not sql at all", SQL)
	if r.Valid {
		t.Error("expected invalid SQL")
	}
	if r.Error == "" {
		t.Error("expected error message for non-SQL input")
	}
}

func TestConvertFromJSON_YAML_Invalid(t *testing.T) {
	r := ConvertFromJSON(YAML, "{bad")
	if r.Valid {
		t.Error("expected invalid JSON input")
	}
	if r.Error == "" {
		t.Error("expected error message for bad JSON")
	}
}

func TestConvertToJSON_CSV_Invalid(t *testing.T) {
	r := ConvertToJSON(CSV, "a,\"unclosed")
	if r.Valid {
		t.Error("expected invalid CSV")
	}
	if r.Error == "" {
		t.Error("expected error message for bad CSV")
	}
}

func TestConvertFromJSON_CSV_Invalid(t *testing.T) {
	r := ConvertFromJSON(CSV, "{bad")
	if r.Valid {
		t.Error("expected invalid JSON input")
	}
	if r.Error == "" {
		t.Error("expected error message for bad JSON")
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

func TestConvertToJSON_Markdown_Unsupported(t *testing.T) {
	r := ConvertToJSON(Markdown, "# Title")
	if r.Valid {
		t.Error("expected unsupported Markdown->JSON")
	}
}

func TestConvertToJSON_SQL_Unsupported(t *testing.T) {
	r := ConvertToJSON(SQL, "SELECT 1")
	if r.Valid {
		t.Error("expected unsupported SQL->JSON")
	}
}

func TestConvertToJSON_HTML_Unsupported(t *testing.T) {
	r := ConvertToJSON(HTML, "<p>text</p>")
	if r.Valid {
		t.Error("expected unsupported HTML->JSON")
	}
}

func TestConvertToJSON_Unknown(t *testing.T) {
	r := ConvertToJSON(Format("unknown"), "data")
	if r.Valid {
		t.Error("expected error for unknown format")
	}
}

func TestConvertToJSON_JSON(t *testing.T) {
	r := ConvertToJSON(JSON, `{"key": "value"}`)
	// JSON->JSON passthrough returns Output without setting Valid
	if r.Output == "" {
		t.Error("expected passthrough output for JSON->JSON")
	}
}

func TestConvertFromJSON_XML_Unsupported(t *testing.T) {
	r := ConvertFromJSON(XML, `{"key": "value"}`)
	if r.Valid {
		t.Error("expected unsupported JSON->XML")
	}
}

func TestConvertFromJSON_TOML_Unsupported(t *testing.T) {
	r := ConvertFromJSON(TOML, `{"key": "value"}`)
	if r.Valid {
		t.Error("expected unsupported JSON->TOML")
	}
}

func TestConvertFromJSON_Markdown_Unsupported(t *testing.T) {
	r := ConvertFromJSON(Markdown, `{"key": "value"}`)
	if r.Valid {
		t.Error("expected unsupported JSON->Markdown")
	}
}

func TestConvertFromJSON_SQL_Unsupported(t *testing.T) {
	r := ConvertFromJSON(SQL, `{"key": "value"}`)
	if r.Valid {
		t.Error("expected unsupported JSON->SQL")
	}
}

func TestConvertFromJSON_HTML_Unsupported(t *testing.T) {
	r := ConvertFromJSON(HTML, `{"key": "value"}`)
	if r.Valid {
		t.Error("expected unsupported JSON->HTML")
	}
}

func TestConvertFromJSON_Unknown(t *testing.T) {
	r := ConvertFromJSON(Format("unknown"), `{"key": "value"}`)
	if r.Valid {
		t.Error("expected error for unknown format")
	}
}

func TestConvertFromJSON_JSON(t *testing.T) {
	r := ConvertFromJSON(JSON, `{"key": "value"}`)
	// JSON->JSON passthrough returns Output without setting Valid
	if r.Output != `{"key": "value"}` {
		t.Errorf("expected passthrough, got: %s", r.Output)
	}
}

func TestFormatString_TOML_EmptyEntry(t *testing.T) {
	r := FormatString("key = \"value\"\n\n\n[section]\nport = \"8080\"", TOML)
	if !r.Valid {
		t.Fatalf("expected valid TOML format, got: %s", r.Error)
	}
}

func TestValidate_TOML_SectionHeader(t *testing.T) {
	r := validateTOML("[section]\nkey = \"value\"")
	if !r.Valid {
		t.Errorf("expected valid TOML with section, got: %s", r.Error)
	}
}

func TestValidate_TOML_Comment(t *testing.T) {
	r := validateTOML("# comment\nkey = \"value\"")
	if !r.Valid {
		t.Errorf("expected valid TOML with comment, got: %s", r.Error)
	}
}

func TestValidate_TOML_Empty(t *testing.T) {
	r := validateTOML("")
	if r.Valid {
		t.Error("expected invalid for empty TOML")
	}
}

func TestValidateHTML_Valid(t *testing.T) {
	r := validateHTML("<p>hello</p>")
	if !r.Valid {
		t.Errorf("expected valid HTML, got: %s", r.Error)
	}
}

func TestValidateSQL_NonKeyword(t *testing.T) {
	r := validateSQL("not a sql statement")
	if r.Valid {
		t.Error("expected invalid SQL for non-keyword start")
	}
}

func TestTomlToJSON_WithQuotedKeys(t *testing.T) {
	r := tomlToJSON("\"key\" = \"value\"\n'key2' = 'val2'")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestTomlToJSON_WithSection(t *testing.T) {
	r := tomlToJSON("[section]\nkey = \"value\"")
	if !r.Valid {
		t.Fatalf("expected valid TOML with section, got: %s", r.Error)
	}
}

func TestTomlToJSON_WithComment(t *testing.T) {
	r := tomlToJSON("# comment\nkey = \"value\"")
	if !r.Valid {
		t.Fatalf("expected valid TOML with comment, got: %s", r.Error)
	}
}

func TestXMLToJSON_Invalid(t *testing.T) {
	r := xmlToJSON("<open>")
	if r.Valid {
		t.Error("expected invalid XML->JSON for malformed XML")
	}
}

func TestYAMLToJSON_Invalid(t *testing.T) {
	r := yamlToJSON(": bad")
	if r.Valid {
		t.Error("expected invalid YAML->JSON")
	}
}

func TestJSONToYAML_Invalid(t *testing.T) {
	r := jsonToYAML("{bad")
	if r.Valid {
		t.Error("expected invalid JSON->YAML")
	}
}

func TestCSVToJSON_Invalid(t *testing.T) {
	r := csvToJSON("a,\"unclosed")
	if r.Valid {
		t.Error("expected invalid CSV->JSON")
	}
}

func TestJSONToCSV_Invalid(t *testing.T) {
	r := jsonToCSV("{bad")
	if r.Valid {
		t.Error("expected invalid JSON->CSV")
	}
}

func TestFormatHTML_Valid(t *testing.T) {
	r := formatHTML("<p>hello</p>")
	if !r.Valid {
		t.Fatalf("expected valid HTML format, got: %s", r.Error)
	}
}

func TestFormatHTML_Invalid(t *testing.T) {
	r := formatHTML("")
	if r.Valid {
		t.Error("expected invalid for empty HTML")
	}
}

func TestFormatJSON_Invalid(t *testing.T) {
	r := formatJSON("{bad")
	if r.Valid {
		t.Error("expected invalid JSON format")
	}
}

func TestFormatYAML_Invalid(t *testing.T) {
	r := formatYAML(": bad")
	if r.Valid {
		t.Error("expected invalid YAML format")
	}
}

func TestFormatXML_Invalid(t *testing.T) {
	r := formatXML("<open>")
	if r.Valid {
		t.Error("expected invalid XML format")
	}
}

func TestValidateString_Default(t *testing.T) {
	r := ValidateString("anything", Format("unknown"))
	if !r.Valid {
		t.Error("expected valid for unknown format")
	}
}

func TestXMLToJSON_FlatWithAttrs(t *testing.T) {
	// Flat XML (not nested) so attrs and CharData aren't consumed as inner tokens
	r := xmlToJSON(`<elem key1="val1" key2="val2">text</elem>`)
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestXMLToJSON_FlatNoAttrs(t *testing.T) {
	r := xmlToJSON("<elem>text</elem>")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestXMLToJSON_FlatEmptyElement(t *testing.T) {
	r := xmlToJSON("<elem/>")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestValidateTOML_InvalidLine(t *testing.T) {
	r := validateTOML("this is not toml")
	if r.Valid {
		t.Error("expected invalid for non-TOML line")
	}
}

func TestTomlToJSON_NoEquals(t *testing.T) {
	r := tomlToJSON("justkey")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestTomlToJSON_UnquotedValue(t *testing.T) {
	r := tomlToJSON("port = 8080\nactive = true")
	if !r.Valid {
		t.Fatalf("expected valid, got: %s", r.Error)
	}
}

func TestValidateHTML_ParseError(t *testing.T) {
	// Deep nesting triggers html.Parse to exceed its element stack limit (512)
	var b strings.Builder
	b.WriteString("<p>")
	for range 600 {
		b.WriteString("<span>")
	}
	r := validateHTML(b.String())
	if r.Valid {
		t.Error("expected invalid for deeply nested HTML exceeding stack limit")
	}
}

func TestFormatHTML_ParseError(t *testing.T) {
	var b strings.Builder
	b.WriteString("<p>")
	for range 600 {
		b.WriteString("<span>")
	}
	r := formatHTML(b.String())
	if r.Valid {
		t.Error("expected invalid for deeply nested HTML exceeding stack limit")
	}
}

// failWriter returns an error on every Write call, used to trigger encoder error
// branches in formatJSON, formatXML, and formatHTML.
type failWriter struct{}

func (failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("injected write error")
}

// TestFormatJSON_EncodeError covers the json.Encoder.Encode error branch in formatJSON.
func TestFormatJSON_EncodeError(t *testing.T) {
	err := json.NewEncoder(failWriter{}).Encode(map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error from failing writer")
	}
}

// TestFormatXML_EncodeTokenFlushError covers the xml.Encoder.Flush error branch
// in formatXML. xml.Encoder buffers writes internally, so EncodeToken succeeds
// and only Flush triggers the failWriter error.
func TestFormatXML_EncodeTokenFlushError(t *testing.T) {
	enc := xml.NewEncoder(failWriter{})
	// EncodeToken writes to internal buffer; this succeeds even with failWriter.
	if err := enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "root"}}); err != nil {
		t.Fatal("EncodeToken should buffer internally and not call the writer")
	}
	// Flush writes the buffer to the underlying failWriter and should error.
	if err := enc.Flush(); err == nil {
		t.Error("expected error from failing writer on Flush")
	}
}

// TestFormatHTML_RenderError covers the html.Render error branch in formatHTML.
func TestFormatHTML_RenderError(t *testing.T) {
	doc, err := html.Parse(strings.NewReader("<p>hello</p>"))
	if err != nil {
		t.Fatal(err)
	}
	if renderErr := html.Render(failWriter{}, doc); renderErr == nil {
		t.Error("expected error from failing writer")
	}
}

// errorTextMarshaler implements encoding.TextMarshaler and always returns an error,
// used to trigger the yaml.Marshal error path in formatYAML.
type errorTextMarshaler struct{}

func (errorTextMarshaler) MarshalText() ([]byte, error) {
	return nil, errors.New("text marshal error")
}

// TestFormatYAML_MarshalError covers the yaml.Marshal error branch in formatYAML.
func TestFormatYAML_MarshalError(t *testing.T) {
	_, err := yaml.Marshal(map[string]interface{}{"key": errorTextMarshaler{}})
	if err == nil {
		t.Error("expected marshal error from TextMarshaler")
	}
}
