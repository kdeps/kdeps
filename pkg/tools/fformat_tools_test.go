// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tools/fformat"
)

func TestAllFormats(t *testing.T) {
	formats := allFormats()
	if len(formats) != 8 {
		t.Fatalf("expected 8 formats, got %d", len(formats))
	}
	expected := []string{"json", "yaml", "csv", "xml", "toml", "markdown", "sql", "html"}
	for i, f := range expected {
		if formats[i] != f {
			t.Errorf("allFormats[%d] = %q, want %q", i, formats[i], f)
		}
	}
}

func TestMarshalFFormatResult_Success(t *testing.T) {
	result := fformat.Result{Valid: true, Output: "test-output"}
	out, err := marshalFFormatResult(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var decoded fformat.Result
	if err = json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("marshalFFormatResult output is not valid JSON: %v", err)
	}
	if !decoded.Valid {
		t.Error("expected valid=true")
	}
	if decoded.Output != "test-output" {
		t.Errorf("expected output %q, got %q", "test-output", decoded.Output)
	}
}

func TestRegisterFFormatTools(t *testing.T) {
	r := NewRegistry()
	RegisterFFormatTools(r)

	for _, name := range []string{
		"fformat_validate",
		"fformat_format",
		"fformat_convert_to_json",
		"fformat_convert_from_json",
	} {
		if tool := r.Get(name); tool == nil {
			t.Errorf("expected tool %q to be registered, but it was not found", name)
		}
	}
	if len(r.List()) != 4 {
		t.Errorf("expected 4 registered tools, got %d", len(r.List()))
	}
}

func TestFFormatValidateTool_InputNotString(t *testing.T) {
	tool := fformatValidateTool()
	_, err := tool.Execute(map[string]interface{}{"input": 42, "format": "json"})
	if !errors.Is(err, errInputNotString) {
		t.Errorf("expected errInputNotString, got %v", err)
	}
}

func TestFFormatValidateTool_FormatNotString(t *testing.T) {
	tool := fformatValidateTool()
	_, err := tool.Execute(map[string]interface{}{"input": `{"a":1}`, "format": 42})
	if !errors.Is(err, errFormatNotString) {
		t.Errorf("expected errFormatNotString, got %v", err)
	}
}

func TestFFormatValidateTool_Success(t *testing.T) {
	tool := fformatValidateTool()
	result, err := tool.Execute(map[string]interface{}{"input": `{"a":1}`, "format": "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var res fformat.Result
	if err = json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if !res.Valid {
		t.Error("expected valid=true for valid JSON input")
	}
}

func TestFFormatFormatTool_InputNotString(t *testing.T) {
	tool := fformatFormatTool()
	_, err := tool.Execute(map[string]interface{}{"input": 42, "format": "json"})
	if !errors.Is(err, errInputNotString) {
		t.Errorf("expected errInputNotString, got %v", err)
	}
}

func TestFFormatFormatTool_FormatNotString(t *testing.T) {
	tool := fformatFormatTool()
	_, err := tool.Execute(map[string]interface{}{"input": `{"a":1}`, "format": 42})
	if !errors.Is(err, errFormatNotString) {
		t.Errorf("expected errFormatNotString, got %v", err)
	}
}

func TestFFormatFormatTool_Success(t *testing.T) {
	tool := fformatFormatTool()
	result, err := tool.Execute(map[string]interface{}{"input": `{"a":1}`, "format": "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var res fformat.Result
	if err = json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if !res.Valid {
		t.Error("expected valid=true for valid JSON input")
	}
}

func TestFFormatFormatTool_CSVPassThrough(t *testing.T) {
	tool := fformatFormatTool()
	result, err := tool.Execute(map[string]interface{}{"input": "a,b,c\n1,2,3", "format": "csv"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var res fformat.Result
	if err = json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if res.Output != "a,b,c\n1,2,3" {
		t.Errorf("expected CSV passthrough, got %q", res.Output)
	}
}

func TestFFormatConvertToJSONTool_InputNotString(t *testing.T) {
	tool := fformatConvertToJSONTool()
	_, err := tool.Execute(map[string]interface{}{"input": 42, "from": "yaml"})
	if !errors.Is(err, errInputNotString) {
		t.Errorf("expected errInputNotString, got %v", err)
	}
}

func TestFFormatConvertToJSONTool_FromNotString(t *testing.T) {
	tool := fformatConvertToJSONTool()
	_, err := tool.Execute(map[string]interface{}{"input": "a: 1\n", "from": 42})
	if !errors.Is(err, errFromNotString) {
		t.Errorf("expected errFromNotString, got %v", err)
	}
}

func TestFFormatConvertToJSONTool_Success(t *testing.T) {
	tool := fformatConvertToJSONTool()
	result, err := tool.Execute(map[string]interface{}{"input": "a: 1\n", "from": "yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var res fformat.Result
	if err = json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if !res.Valid {
		t.Error("expected valid=true for valid YAML input")
	}
}

func TestFFormatConvertFromJSONTool_InputNotString(t *testing.T) {
	tool := fformatConvertFromJSONTool()
	_, err := tool.Execute(map[string]interface{}{"input": 42, "to": "yaml"})
	if !errors.Is(err, errInputNotString) {
		t.Errorf("expected errInputNotString, got %v", err)
	}
}

func TestFFormatConvertFromJSONTool_ToNotString(t *testing.T) {
	tool := fformatConvertFromJSONTool()
	_, err := tool.Execute(map[string]interface{}{"input": `{"a":1}`, "to": 42})
	if !errors.Is(err, errToNotString) {
		t.Errorf("expected errToNotString, got %v", err)
	}
}

func TestFFormatConvertFromJSONTool_Success(t *testing.T) {
	tool := fformatConvertFromJSONTool()
	result, err := tool.Execute(map[string]interface{}{"input": `{"a":1}`, "to": "yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var res fformat.Result
	if err = json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if !res.Valid {
		t.Error("expected valid=true for valid JSON input")
	}
}

func TestMarshalFFormatResult_MarshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}
	_, err := marshalFFormatResult(fformat.Result{Valid: true, Output: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected marshal error")
}
