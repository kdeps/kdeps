//go:build !js

package codeintelligence

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// mockRunner captures rg commands and returns canned JSON output.
type mockRunner struct {
	entries map[string]string // command args -> rg JSON output
	rgErr   error
}

func (r *mockRunner) Run(_ string, args ...string) (string, string, error) {
	key := strings.Join(args, " ")
	if out, ok := r.entries[key]; ok {
		return out, "", nil
	}
	if r.rgErr != nil {
		return "", "", r.rgErr
	}
	return "", "", nil
}

func newTestExecutor(runner Runner) *Executor {
	return NewExecutorWithRunner(runner)
}

// rgMatchJSON produces a single rg JSON match line for a given file/line/match.
func rgMatchJSON(file, lineContent string, lineNum int) string {
	escapedContent := strings.ReplaceAll(lineContent, `"`, `\"`)
	return fmt.Sprintf(
		`{"type":"match","data":{"path":{"text":"%s"},"lines":{"text":"%s\n"},"line_number":%d,"absolute_offset":0,"submatches":[{"match":{"text":"%s"},"start":0,"end":0}]}}`,
		file,
		escapedContent,
		lineNum,
		lineContent,
	)
}

// createGoSource creates a temp Go source file for testing.
func createGoSource(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func hasRG() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

// --- Tests ---

func TestExecute_RequiresOperation(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{})
	if err == nil || !strings.Contains(err.Error(), "operation is required") {
		t.Fatalf("expected operation required error, got: %v", err)
	}
}

func TestExecute_UnsupportedOperation(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{Operation: "invalid"})
	if err == nil || !strings.Contains(err.Error(), `unsupported operation "invalid"`) {
		t.Fatalf("expected unsupported error, got: %v", err)
	}
}

func TestSymbolSearch_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --glob *.go parseFunc /path": rgMatchJSON("main.go", "func parseFunc() {", 10),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "parseFunc",
		Path:      "/path",
		Pattern:   "*.go",
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
	if symbols[0]["file"] != "main.go" {
		t.Fatalf("expected file 'main.go', got %q", symbols[0]["file"])
	}
}

func TestSymbolSearch_RequiresQuery(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
	})
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("expected query required error, got: %v", err)
	}
}

func TestSymbolSearch_WithRG(t *testing.T) {
	if !hasRG() {
		t.Skip("ripgrep (rg) not installed")
	}

	dir := t.TempDir()
	createGoSource(t, dir, "main.go", `package main

func hello() string {
	return "hello"
}

func main() {
	println(hello())
}
`)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "hello",
		Path:      dir,
		Pattern:   "*.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	symbols := m["symbols"].([]map[string]interface{})
	if len(symbols) < 1 {
		t.Fatal("expected at least 1 symbol match")
	}
}

func TestSymbolSearch_NoMatches(t *testing.T) {
	runner := &mockRunner{}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "nonexistent",
		Path:      "/path",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["count"].(int) != 0 {
		t.Fatalf("expected 0 matches, got %d", m["count"])
	}
}

func TestDefinition_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --sort path ^(func |type |var |const )?ParseConfig /path`: rgMatchJSON(
				"config.go",
				"func ParseConfig() {",
				5,
			),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDefinition,
		Symbol:    "ParseConfig",
		Path:      "/path",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	defs := m["definitions"].([]map[string]interface{})
	if len(defs) < 1 {
		t.Fatal("expected at least 1 definition")
	}
}

func TestDefinition_RequiresSymbol(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDefinition,
	})
	if err == nil || !strings.Contains(err.Error(), "symbol is required") {
		t.Fatalf("expected symbol required error, got: %v", err)
	}
}

func TestDefinition_WithRG(t *testing.T) {
	if !hasRG() {
		t.Skip("ripgrep (rg) not installed")
	}

	dir := t.TempDir()
	createGoSource(t, dir, "main.go", `package main

type Config struct {
	Name string
}

func LoadConfig() *Config {
	return &Config{Name: "test"}
}
`)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDefinition,
		Symbol:    "LoadConfig",
		Path:      dir,
		Pattern:   "*.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestReferences_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --sort path myFunc\. /path`: rgMatchJSON("main.go", "myFunc()", 3),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpReferences,
		Symbol:    "myFunc",
		Path:      "/path",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestReferences_RequiresSymbol(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpReferences,
	})
	if err == nil || !strings.Contains(err.Error(), "symbol is required") {
		t.Fatalf("expected symbol required error, got: %v", err)
	}
}

func TestReferences_WithRG(t *testing.T) {
	if !hasRG() {
		t.Skip("ripgrep (rg) not installed")
	}

	dir := t.TempDir()
	createGoSource(t, dir, "main.go", `package main

func hello() string {
	return "world"
}

func main() {
	msg := hello()
	println(msg)
}
`)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpReferences,
		Symbol:    "hello",
		Path:      dir,
		Pattern:   "*.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	refs := m["references"].([]map[string]interface{})
	if len(refs) < 1 {
		t.Fatal("expected at least 1 reference")
	}
}

func TestDocumentSymbols_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --sort path ^(func |type |class |def |function |interface |struct |enum |trait |impl ) /path/main.go`: rgMatchJSON(
				"main.go",
				"func main() {",
				1,
			),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "/path/main.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestDocumentSymbols_RequiresPath(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
	})
	if err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("expected path required error, got: %v", err)
	}
}

func TestDocumentSymbols_WithRG(t *testing.T) {
	if !hasRG() {
		t.Skip("ripgrep (rg) not installed")
	}

	dir := t.TempDir()
	createGoSource(t, dir, "main.go", `package main

type Result struct {
	Value string
}

func process() Result {
	return Result{Value: "ok"}
}

func main() {}
`)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      dir,
		Pattern:   "*.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	symbols := m["symbols"].([]map[string]interface{})
	if len(symbols) < 2 {
		t.Fatalf("expected at least 2 symbols, got %d", len(symbols))
	}
}

func TestHover_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number -B 5 -A 2 ^(func |type |var |const )?myFunc\\. /path`: rgMatchJSON(
				"main.go",
				"func myFunc() {",
				10,
			),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpHover,
		Symbol:    "myFunc",
		Path:      "/path",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestHover_RequiresPath(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpHover,
		Symbol:    "x",
	})
	if err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("expected path required error, got: %v", err)
	}
}

func TestHover_RequiresSymbol(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpHover,
		Path:      "/path",
	})
	if err == nil || !strings.Contains(err.Error(), "symbol is required") {
		t.Fatalf("expected symbol required error, got: %v", err)
	}
}

func TestDiagnostics_RequiresPath(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
	})
	if err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("expected path required error, got: %v", err)
	}
}

func TestDiagnostics_GoVet(t *testing.T) {
	dir := t.TempDir()
	createGoSource(t, dir, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestSymbolSearch_WithContextLines(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number -C 3 --max-count 5 parse /path`: rgMatchJSON("file.go", "parse()", 5),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "parse",
		Path:      "/path",
		Context:   3,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestSymbolSearch_WithLanguage(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --type go --max-count 10 func /path`: rgMatchJSON("main.go", "func main()", 1),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "func",
		Path:      "/path",
		Language:  "go",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestExtractSymbolName(t *testing.T) {
	if n := extractSymbolName("func main() {"); n != "main()" {
		t.Fatalf("expected 'main()', got %q", n)
	}
	if n := extractSymbolName("func parseInput(data string) error"); n != "parseInput(data" {
		t.Fatalf("expected 'parseInput(data', got %q", n)
	}
	if n := extractSymbolName(""); n != "" {
		t.Fatalf("expected empty, got %q", n)
	}
}

func TestInferKind(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"func main() {", "method"},
		{"func (c *Config) Get()", "method"},
		{"type Config struct", "struct"},
		{"type Reader interface", "interface"},
		{"type MyType string", "type"},
		{"var x int", "variable"},
		{"const max = 100", "constant"},
		{"class User {", "class"},
		{"def hello():", "function"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := inferKind(tc.line); got != tc.want {
				t.Fatalf("inferKind(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

func TestIsGoDir(t *testing.T) {
	if isGoDir("/nonexistent") {
		t.Fatal("expected false for nonexistent dir")
	}

	dir := t.TempDir()
	if isGoDir(dir) {
		t.Fatal("expected false for dir with no .go files")
	}

	createGoSource(t, dir, "main.go", "package main\n")
	if !isGoDir(dir) {
		t.Fatal("expected true for dir with .go files")
	}
}

func TestDefaultRunner_Run(t *testing.T) {
	r := &DefaultRunner{}
	stdout, stderr, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(stdout) != "hello" {
		t.Fatalf("expected 'hello', got %q", strings.TrimSpace(stdout))
	}
	_ = stderr
}

func TestDiagnostics_NonGoPath(t *testing.T) {
	dir := t.TempDir()
	createGoSource(t, dir, "data.txt", "just text")

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["count"].(int) != 0 {
		t.Fatalf("expected 0 diagnostics for non-Go dir, got %d", m["count"])
	}
}

func TestRGNotInstalled(t *testing.T) {
	// Create runner that simulates rg not being installed
	// We test the error path in runRG
	runner := &mockRunner{
		rgErr: errors.New("executable file not found"),
	}
	e := newTestExecutor(runner)
	// We just need to make sure it doesn't panic
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "test",
		Path:      "/path",
	})
	if err == nil {
		t.Fatal("expected error when rg not found")
	}
	_ = res
}
