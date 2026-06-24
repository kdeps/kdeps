package codeintelligence

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// --- resultLSP tests ---

func TestResultLSP_NilData(t *testing.T) {
	m := resultLSP(true, nil)
	require.NotNil(t, m)
	assert.Equal(t, true, m["success"])
}

func TestResultLSP_WithData(t *testing.T) {
	m := resultLSP(false, map[string]interface{}{"key": "value"})
	assert.Equal(t, false, m["success"])
	assert.Equal(t, "value", m["key"])
}

// --- lineFromPosition tests ---

func TestLineFromPosition_NotMap(t *testing.T) {
	assert.Equal(t, 0, lineFromPosition("string"))
	assert.Equal(t, 0, lineFromPosition(42))
	assert.Equal(t, 0, lineFromPosition(nil))
}

func TestLineFromPosition_NoStart(t *testing.T) {
	assert.Equal(t, 0, lineFromPosition(map[string]interface{}{}))
	assert.Equal(t, 0, lineFromPosition(map[string]interface{}{"other": "value"}))
}

func TestLineFromPosition_NoLine(t *testing.T) {
	assert.Equal(t, 0, lineFromPosition(map[string]interface{}{
		"start": map[string]interface{}{},
	}))
}

func TestLineFromPosition_LineNotFloat64(t *testing.T) {
	assert.Equal(t, 0, lineFromPosition(map[string]interface{}{
		"start": map[string]interface{}{
			"line": 5, // int, not float64
		},
	}))
}

func TestLineFromPosition_Success(t *testing.T) {
	result := lineFromPosition(map[string]interface{}{
		"start": map[string]interface{}{
			"line": float64(5),
		},
	})
	assert.Equal(t, 6, result) // +1 for LSP 0-based conversion
}

func TestLineFromPosition_ZeroLine(t *testing.T) {
	result := lineFromPosition(map[string]interface{}{
		"start": map[string]interface{}{
			"line": float64(0),
		},
	})
	assert.Equal(t, 1, result) // 0 + 1 = 1
}

// --- flattenLSPDocumentSymbols tests ---

func TestFlattenLSPDocumentSymbols_NotSlice(t *testing.T) {
	assert.Empty(t, flattenLSPDocumentSymbols("not a slice"))
	assert.Empty(t, flattenLSPDocumentSymbols(42))
	assert.Empty(t, flattenLSPDocumentSymbols(nil))
}

func TestFlattenLSPDocumentSymbols_FlatList(t *testing.T) {
	result := flattenLSPDocumentSymbols([]interface{}{
		map[string]interface{}{"name": "func1", "kind": "Function"},
		map[string]interface{}{"name": "var1", "kind": "Variable"},
	})
	require.Len(t, result, 2)
	assert.Equal(t, "func1", result[0]["name"])
	assert.Equal(t, "Function", result[0]["kind"])
	assert.Equal(t, "Variable", result[1]["kind"])
}

func TestFlattenLSPDocumentSymbols_WithChildren(t *testing.T) {
	result := flattenLSPDocumentSymbols([]interface{}{
		map[string]interface{}{
			"name": "MyStruct",
			"kind": "Struct",
			"children": []interface{}{
				map[string]interface{}{"name": "Field1", "kind": "Field"},
				map[string]interface{}{"name": "Field2", "kind": "Field"},
			},
		},
	})
	require.Len(t, result, 3) // parent + 2 children
	assert.Equal(t, "MyStruct", result[0]["name"])
	assert.Equal(t, "Field1", result[1]["name"])
	assert.Equal(t, "Field2", result[2]["name"])
}

func TestFlattenLSPDocumentSymbols_NestedChildren(t *testing.T) {
	result := flattenLSPDocumentSymbols([]interface{}{
		map[string]interface{}{
			"name": "Outer",
			"kind": "Class",
			"children": []interface{}{
				map[string]interface{}{
					"name": "Inner",
					"kind": "Method",
					"children": []interface{}{
						map[string]interface{}{"name": "Nested", "kind": "Function"},
					},
				},
			},
		},
	})
	require.Len(t, result, 3)
}

func TestFlattenLSPDocumentSymbols_SkipInvalidItems(t *testing.T) {
	result := flattenLSPDocumentSymbols([]interface{}{
		map[string]interface{}{"name": "valid", "kind": "Function"},
		"invalid string item",
		42,
	})
	require.Len(t, result, 1)
	assert.Equal(t, "valid", result[0]["name"])
}

// --- findSymbolPosition tests ---

func TestFindSymbolPosition_EmptyConfig(t *testing.T) {
	e := &Executor{}
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{})
	assert.Equal(t, 0, pos["line"])
	assert.Equal(t, 0, pos["character"])
}

func TestFindSymbolPosition_EmptyPath(t *testing.T) {
	e := &Executor{}
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{
		Symbol: "myFunc",
	})
	assert.Equal(t, 0, pos["line"])
	assert.Equal(t, 0, pos["character"])
}

func TestFindSymbolPosition_RGError(t *testing.T) {
	runner := &mockRunner{
		rgErr: errors.New("rg error"),
	}
	e := newTestExecutor(runner)
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{
		Path:   "/path/file.go",
		Symbol: "myFunc",
	})
	assert.Equal(t, 0, pos["line"])
	assert.Equal(t, 0, pos["character"])
}

func TestFindSymbolPosition_NoMatches(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{
		Path:   "/path/file.go",
		Symbol: "nonexistent",
	})
	assert.Equal(t, 0, pos["line"])
	assert.Equal(t, 0, pos["character"])
}

func TestFindSymbolPosition_WithMatch(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 myFunc /path/file.go": rgMatchJSON("file.go", "func myFunc() {}", 10),
		},
	}
	e := newTestExecutor(runner)
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{
		Path:   "/path/file.go",
		Symbol: "myFunc",
	})
	assert.Equal(t, 9, pos["line"])      // 10 - 1 = 9 (0-based)
	assert.Equal(t, 5, pos["character"]) // index of "m" in "func myFunc() {}"
}

func TestFindSymbolPosition_ZeroLineNumber(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 myFunc /path/file.go": rgMatchJSON("file.go", "myFunc", 0),
		},
	}
	e := newTestExecutor(runner)
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{
		Path:   "/path/file.go",
		Symbol: "myFunc",
	})
	assert.Equal(t, 0, pos["line"]) // 0 - 1 = -1 clamped to 0
	assert.Equal(t, 0, pos["character"])
}

func TestFindSymbolPosition_SymbolNotFoundInLine(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 myFunc /path/file.go": rgMatchJSON(
				"file.go", "func otherFunc() {}", 10,
			),
		},
	}
	e := newTestExecutor(runner)
	pos := e.findSymbolPosition(&domain.CodeIntelligenceConfig{
		Path:   "/path/file.go",
		Symbol: "myFunc",
	})
	assert.Equal(t, 9, pos["line"])      // 10 - 1 = 9
	assert.Equal(t, 0, pos["character"]) // not found -> 0
}

// --- lspEnsureDocument tests ---

func TestLSPEnsureDocument_EmptyPath(t *testing.T) {
	e := &Executor{}
	err := e.lspEnsureDocument(nil, &domain.CodeIntelligenceConfig{})
	assert.NoError(t, err)
}

func TestLSPEnsureDocument_FileNotFound(t *testing.T) {
	e := &Executor{}
	err := e.lspEnsureDocument(nil, &domain.CodeIntelligenceConfig{
		Path: "/nonexistent/file.go",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lsp: read file")
}

func TestLSPEnsureDocument_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	var buf bytes.Buffer
	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &buf},
	}
	e := &Executor{}
	err := e.lspEnsureDocument(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"method":"textDocument/didOpen"`)
	assert.Contains(t, output, `"text":"package main`)
}

// --- lspSymbolSearch tests ---

func TestLSPSymbolSearch_QueryRequired(t *testing.T) {
	e := &Executor{}
	_, err := e.lspSymbolSearch(nil, &domain.CodeIntelligenceConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

func TestLSPSymbolSearch_CallError(t *testing.T) {
	client := &lspClient{
		stdin: errWriteCloser{},
	}
	e := &Executor{}
	_, err := e.lspSymbolSearch(client, &domain.CodeIntelligenceConfig{
		Query: "test",
	})
	assert.Error(t, err)
}

func TestLSPSymbolSearch_Success(t *testing.T) {
	resp := `{"jsonrpc":"2.0","id":1,"result":[{"name":"main","kind":"Function","location":{"uri":"file:///path/main.go"}}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	e := &Executor{}
	result, err := e.lspSymbolSearch(client, &domain.CodeIntelligenceConfig{Query: "main"})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	symbols := m["symbols"].([]map[string]interface{})
	require.Len(t, symbols, 1)
	assert.Equal(t, "main", symbols[0]["name"])
	assert.Equal(t, "Function", symbols[0]["kind"])
	assert.Equal(t, "/path/main.go", symbols[0]["file"])
}

// --- lspDefinition tests ---

func TestLSPDefinition_SymbolRequired(t *testing.T) {
	e := &Executor{}
	_, err := e.lspDefinition(nil, &domain.CodeIntelligenceConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol is required")
}

func TestLSPDefinition_CallError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)
	client := &lspClient{
		stdin: errWriteCloser{},
	}
	_, err := e.lspDefinition(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	assert.Error(t, err)
}

func TestLSPDefinition_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":[{"uri":"file:///path/main.go","range":{"start":{"line":1,"character":5}}}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.lspDefinition(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	defs := m["definitions"].([]map[string]interface{})
	require.Len(t, defs, 1)
	assert.Equal(t, "/path/main.go", defs[0]["file"])
	assert.Equal(t, 2, defs[0]["line"]) // 1 + 1 = 2 (LSP 0-based -> 1-based)
}

// --- lspReferences tests ---

func TestLSPReferences_SymbolRequired(t *testing.T) {
	e := &Executor{}
	_, err := e.lspReferences(nil, &domain.CodeIntelligenceConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol is required")
}

func TestLSPReferences_CallError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)
	client := &lspClient{
		stdin: errWriteCloser{},
	}
	_, err := e.lspReferences(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	assert.Error(t, err)
}

func TestLSPReferences_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":[{"uri":"file:///path/main.go","range":{"start":{"line":1,"character":5}}}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.lspReferences(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	refs := m["references"].([]map[string]interface{})
	require.Len(t, refs, 1)
	assert.Equal(t, "/path/main.go", refs[0]["file"])
}

// --- lspHover tests ---

func TestLSPHover_PathRequired(t *testing.T) {
	e := &Executor{}
	_, err := e.lspHover(nil, &domain.CodeIntelligenceConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestLSPHover_CallError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	e := &Executor{}
	client := &lspClient{
		stdin: errWriteCloser{},
	}
	_, err := e.lspHover(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	assert.Error(t, err)
}

func TestLSPHover_ContentsString(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":{"contents":"Hello, this is hover info"}}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.lspHover(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	hover := m["hover"].(map[string]interface{})
	assert.Equal(t, "Hello, this is hover info", hover["contents"])
}

func TestLSPHover_ContentsMap(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":{"contents":{"kind":"markdown","value":"Hover details"}}}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.lspHover(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	hover := m["hover"].(map[string]interface{})
	assert.Equal(t, "Hover details", hover["contents"])
}

func TestLSPHover_ContentsMapNoValue(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644))

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":{"contents":{"kind":"markdown"}}}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.lspHover(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	hover := m["hover"].(map[string]interface{})
	assert.Equal(t, "<nil>", hover["contents"])
}

// --- lspDiagnostics tests ---

func TestLSPDiagnostics_PathRequired(t *testing.T) {
	e := &Executor{}
	_, err := e.lspDiagnostics(nil, &domain.CodeIntelligenceConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestLSPDiagnostics_CallError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	e := &Executor{}
	client := &lspClient{
		stdin: errWriteCloser{},
	}
	_, err := e.lspDiagnostics(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	assert.Error(t, err)
}

func TestLSPDiagnostics_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	resp := `{"jsonrpc":"2.0","id":1,"result":{"items":[{"message":"unused variable","severity":1,"source":"go"},{"message":"missing doc comment","severity":2,"source":"go lint"}]}}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	e := &Executor{}
	result, err := e.lspDiagnostics(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	diags := m["diagnostics"].([]map[string]interface{})
	require.Len(t, diags, 2)
	assert.Equal(t, "unused variable", diags[0]["message"])
	assert.Equal(t, "1", diags[0]["severity"])
	assert.Equal(t, "go", diags[0]["source"])
	assert.Equal(t, "missing doc comment", diags[1]["message"])
}

func TestLSPDiagnostics_EmptyItems(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	resp := `{"jsonrpc":"2.0","id":1,"result":{"items":[]}}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	e := &Executor{}
	result, err := e.lspDiagnostics(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	diags := m["diagnostics"].([]map[string]interface{})
	assert.Empty(t, diags)
}

// --- executeLSP tests ---

func TestExecuteLSP_UnsupportedOperation(t *testing.T) {
	e := &Executor{}
	_, err := e.executeLSP(nil, &domain.CodeIntelligenceConfig{
		Operation: "nonexistent",
	}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported operation "nonexistent"`)
}

// --- lspDocumentSymbols tests ---

func TestLSPDocumentSymbols_PathRequired(t *testing.T) {
	e := &Executor{}
	_, err := e.lspDocumentSymbols(nil, &domain.CodeIntelligenceConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestLSPDocumentSymbols_CallError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	e := &Executor{}
	client := &lspClient{
		stdin: errWriteCloser{},
	}
	_, err := e.lspDocumentSymbols(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	assert.Error(t, err)
}

func TestLSPDocumentSymbols_SuccessFlat(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	require.NoError(t, os.WriteFile(filePath, []byte("package main\n"), 0644))

	resp := `{"jsonrpc":"2.0","id":1,"result":[{"name":"main","kind":"Function"},{"name":"init","kind":"Function"}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	e := &Executor{}
	result, err := e.lspDocumentSymbols(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	require.NoError(t, err)

	m := result.(map[string]interface{})
	assert.True(t, m["success"].(bool))
	symbols := m["symbols"].([]map[string]interface{})
	assert.Len(t, symbols, 2)
}
