//go:build !js

package codeintelligence

import (
	"errors"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// --- result() tests ---

func TestResult_NilData(t *testing.T) {
	m := result(true, nil)
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m == nil {
		t.Fatal("expected non-nil result map")
	}
}

// --- documentSymbols() tests ---

func TestDocumentSymbols_NonGoFile(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --sort path ^(func |type |class |def |function |interface |struct |enum |trait |impl ) /path/file.py`: rgMatchJSON(
				"file.py",
				"def hello():",
				3,
			),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "/path/file.py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	symbols := m["symbols"].([]map[string]interface{})
	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
	if symbols[0]["name"] != "hello():" {
		t.Fatalf("expected name 'hello():', got %q", symbols[0]["name"])
	}
	if symbols[0]["kind"] != "function" {
		t.Fatalf("expected kind 'function', got %q", symbols[0]["kind"])
	}
}

// --- goDocumentSymbols() tests ---

func TestGoDocumentSymbols_EmptyMatches(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --sort path ^(func |type |var |const |struct |interface ) main.go`: "",
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "main.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	symbols := m["symbols"].([]map[string]interface{})
	if len(symbols) != 0 {
		t.Fatalf("expected 0 symbols, got %d", len(symbols))
	}
}

func TestGoDocumentSymbols_RGError(t *testing.T) {
	runner := &mockRunner{
		rgErr: errors.New("rg error"),
	}
	e := newTestExecutor(runner)
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "main.go",
	})
	if err == nil {
		t.Fatal("expected error for rg failure")
	}
}

// --- goVetDiagnostics() tests ---

// vetRunner returns fixed stderr output for go vet commands.
type vetRunner struct {
	stderr string
	err    error
}

func (r *vetRunner) Run(_ string, _ ...string) (string, string, error) {
	return "", r.stderr, r.err
}

func TestGoVetDiagnostics_WithStderr(t *testing.T) {
	runner := &vetRunner{
		stderr: "file.go:10:2: undefined: x\nfile.go:15:5: undefined: y",
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      "file.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	diags := m["diagnostics"].([]map[string]interface{})
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diags))
	}
	if diags[0]["tool"] != "go vet" {
		t.Fatalf("expected tool 'go vet', got %q", diags[0]["tool"])
	}
	if diags[0]["file"] != "file.go" {
		t.Fatalf("expected file 'file.go', got %q", diags[0]["file"])
	}
	if diags[0]["line"] != "10" {
		t.Fatalf("expected line '10', got %q", diags[0]["line"])
	}
	if diags[0]["message"] != "undefined: x" {
		t.Fatalf("expected message 'undefined: x', got %q", diags[0]["message"])
	}
}

func TestGoVetDiagnostics_EmptyStderr(t *testing.T) {
	runner := &mockRunner{}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      "clean.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	diags := m["diagnostics"].([]map[string]interface{})
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics, got %d", len(diags))
	}
}

// --- inferKind() remaining branches ---

func TestInferKind_RemainingBranches(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"interface Runnable {", "interface"},
		{"struct Pair {", "struct"},
		{"enum Color {", "enum"},
		{"trait Show {", "trait"},
		{"something else", "symbol"},
		{"", "symbol"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := inferKind(tc.line); got != tc.want {
				t.Fatalf("inferKind(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}
