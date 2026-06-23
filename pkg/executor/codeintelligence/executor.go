//go:build !js

package codeintelligence

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// Runner runs rg commands (overridable for testing).
type Runner interface {
	Run(name string, args ...string) (stdout string, stderr string, err error)
}

// DefaultRunner implements Runner using os/exec.
type DefaultRunner struct{}

func (r *DefaultRunner) Run(name string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Executor performs code-intelligence operations.
type Executor struct {
	runner Runner
}

// NewExecutor creates a new codeIntelligence executor.
func NewExecutor() *Executor {
	return &Executor{runner: &DefaultRunner{}}
}

// NewExecutorWithRunner creates a new executor with custom runner (for testing).
func NewExecutorWithRunner(runner Runner) *Executor {
	return &Executor{runner: runner}
}

// Execute dispatches to the appropriate code-intelligence operation.
func (e *Executor) Execute(
	_ *executor.ExecutionContext,
	config *domain.CodeIntelligenceConfig,
) (interface{}, error) {
	if config.Operation == "" {
		return nil, errors.New("codeIntelligence: operation is required")
	}

	switch config.Operation {
	case domain.CodeIntOpSymbolSearch:
		return e.symbolSearch(config)
	case domain.CodeIntOpDefinition:
		return e.definition(config)
	case domain.CodeIntOpReferences:
		return e.references(config)
	case domain.CodeIntOpDocumentSymbols:
		return e.documentSymbols(config)
	case domain.CodeIntOpHover:
		return e.hover(config)
	case domain.CodeIntOpDiagnostics:
		return e.diagnostics(config)
	default:
		return nil, fmt.Errorf("codeIntelligence: unsupported operation %q", config.Operation)
	}
}

// --- helpers ---

func result(success bool, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["success"] = success
	return data
}

// rgMatch is one match from rg --json output.
type rgMatch struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber     int `json:"line_number"`
		AbsoluteOffset int `json:"absolute_offset"`
		Submatches     []struct {
			Match struct {
				Text string `json:"text"`
			} `json:"match"`
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"submatches"`
	} `json:"data"`
}

func (e *Executor) buildRGArgs(config *domain.CodeIntelligenceConfig, extra ...string) []string {
	args := []string{"--json", "--line-number"}
	if config.Context > 0 {
		args = append(args, "-C", strconv.Itoa(config.Context))
	}
	if config.Pattern != "" {
		args = append(args, "--glob", config.Pattern)
	}
	if config.Language != "" {
		args = append(args, "--type", config.Language)
	}
	if config.Limit > 0 {
		args = append(args, "--max-count", strconv.Itoa(config.Limit))
	}
	args = append(args, extra...)
	return args
}

func (e *Executor) runRG(args []string) ([]rgMatch, error) {
	stdout, stderr, err := e.runner.Run("rg", args...)
	if err != nil {
		// rg exits with code 1 when no matches found — that's not a real error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		// Check if rg is not installed
		if strings.Contains(stderr, "executable file not found") || strings.Contains(err.Error(), "not found") {
			return nil, errors.New(
				"ripgrep (rg) is not installed. Install it with: brew install ripgrep (macOS) or apt install ripgrep (Linux)",
			)
		}
		return nil, fmt.Errorf("rg failed: %s", stderr)
	}

	var matches []rgMatch
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		var m rgMatch
		if unmarshalErr := json.Unmarshal([]byte(scanner.Text()), &m); unmarshalErr != nil {
			continue
		}
		if m.Type == "match" {
			matches = append(matches, m)
		}
	}
	return matches, nil
}

// --- Operations ---

func (e *Executor) symbolSearch(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Query == "" {
		return nil, errors.New("codeIntelligence: query is required for symbolSearch")
	}

	args := e.buildRGArgs(config, config.Query)
	if config.Path != "" {
		args = append(args, config.Path)
	}

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{"error": err.Error()}), err
	}

	var symbols []map[string]interface{}
	for _, m := range matches {
		symbols = append(symbols, map[string]interface{}{
			"file":    m.Data.Path.Text,
			"line":    m.Data.LineNumber,
			"content": strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	}), nil
}

func (e *Executor) definition(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Symbol == "" {
		return nil, errors.New("codeIntelligence: symbol is required for definition")
	}

	// Try to find Go definitions with function/type/variable declarations
	// Pattern: look for lines starting with "func ", "type ", "var ", "const "
	pattern := fmt.Sprintf("^(func |type |var |const )?%s", regexEscape(config.Symbol))
	args := e.buildRGArgs(config, "--sort", "path", pattern)
	if config.Path != "" {
		args = append(args, config.Path)
	}

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{"error": err.Error()}), err
	}

	var defs []map[string]interface{}
	for _, m := range matches {
		defs = append(defs, map[string]interface{}{
			"file":    m.Data.Path.Text,
			"line":    m.Data.LineNumber,
			"content": strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"definitions": defs,
		"count":       len(defs),
	}), nil
}

func (e *Executor) references(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Symbol == "" {
		return nil, errors.New("codeIntelligence: symbol is required for references")
	}

	args := e.buildRGArgs(config, "--sort", "path", regexEscape(config.Symbol))
	if config.Path != "" {
		args = append(args, config.Path)
	}

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{"error": err.Error()}), err
	}

	var refs []map[string]interface{}
	for _, m := range matches {
		refs = append(refs, map[string]interface{}{
			"file":    m.Data.Path.Text,
			"line":    m.Data.LineNumber,
			"content": strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"references": refs,
		"count":      len(refs),
	}), nil
}

func (e *Executor) documentSymbols(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for documentSymbols")
	}

	// Check if it's a Go file — use Go parser for structured output
	if strings.HasSuffix(config.Path, ".go") {
		return e.goDocumentSymbols(config)
	}

	// For other languages, use rg to find function/class/method declarations
	args := e.buildRGArgs(config,
		"--sort", "path",
		"^(func |type |class |def |function |interface |struct |enum |trait |impl )",
		config.Path,
	)

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{"error": err.Error()}), err
	}

	var symbols []map[string]interface{}
	for _, m := range matches {
		symbols = append(symbols, map[string]interface{}{
			"name":    extractSymbolName(m.Data.Lines.Text),
			"kind":    inferKind(m.Data.Lines.Text),
			"file":    m.Data.Path.Text,
			"line":    m.Data.LineNumber,
			"content": strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	}), nil
}

func (e *Executor) goDocumentSymbols(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	// Fall back to rg-based approach for now; Go parser can be added later
	args := e.buildRGArgs(config, "--sort", "path",
		"^(func |type |var |const |struct |interface )",
		config.Path,
	)

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{"error": err.Error()}), err
	}

	var symbols []map[string]interface{}
	for _, m := range matches {
		symbols = append(symbols, map[string]interface{}{
			"name":    extractSymbolName(m.Data.Lines.Text),
			"kind":    inferKind(m.Data.Lines.Text),
			"file":    m.Data.Path.Text,
			"line":    m.Data.LineNumber,
			"content": strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	}), nil
}

func (e *Executor) hover(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for hover")
	}
	if config.Symbol == "" {
		return nil, errors.New("codeIntelligence: symbol is required for hover")
	}

	// Find the symbol definition, then extract the comment block above it
	const hoverContextBefore = "5"
	const hoverContextAfter = "2"
	pattern := fmt.Sprintf("^(func |type |var |const )?%s", regexEscape(config.Symbol))
	args := []string{"--json", "--line-number", "-B", hoverContextBefore, "-A", hoverContextAfter, pattern}
	if config.Path != "" {
		args = append(args, config.Path)
	}

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{"error": err.Error()}), err
	}

	var hoverInfo []map[string]interface{}
	for _, m := range matches {
		hoverInfo = append(hoverInfo, map[string]interface{}{
			"file":    m.Data.Path.Text,
			"line":    m.Data.LineNumber,
			"content": strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"hover": hoverInfo,
		"count": len(hoverInfo),
	}), nil
}

const (
	vetSplitMax = 4
	vetMinParts = 3
)

func (e *Executor) goVetDiagnostics(config *domain.CodeIntelligenceConfig) []map[string]interface{} {
	args := []string{"vet"}
	switch {
	case config.Pattern != "":
		args = append(args, config.Pattern)
	case strings.HasSuffix(config.Path, ".go"):
		args = append(args, config.Path)
	default:
		args = append(args, "./...")
	}

	stdout, stderr, err := e.runner.Run("go", args...)
	if err != nil {
		_ = stdout
	}

	var diagnostics []map[string]interface{}
	if stderr == "" {
		return diagnostics
	}
	scanner := bufio.NewScanner(strings.NewReader(stderr))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", vetSplitMax)
		if len(parts) >= vetMinParts {
			diagnostics = append(diagnostics, map[string]interface{}{
				"file":    parts[0],
				"line":    parts[1],
				"message": strings.TrimSpace(parts[len(parts)-1]),
				"tool":    "go vet",
			})
		}
	}
	return diagnostics
}

func (e *Executor) diagnostics(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for diagnostics")
	}

	var diagnostics []map[string]interface{}
	if strings.HasSuffix(config.Path, ".go") || isGoDir(config.Path) {
		diagnostics = e.goVetDiagnostics(config)
	}

	return result(true, map[string]interface{}{
		"diagnostics": diagnostics,
		"count":       len(diagnostics),
	}), nil
}

// --- utility functions ---

func regexEscape(s string) string {
	return strings.ReplaceAll(s, ".", "\\.")
}

const symbolNameMinFields = 2

func extractSymbolName(line string) string {
	fields := strings.Fields(line)
	if len(fields) >= symbolNameMinFields {
		return fields[1]
	}
	return ""
}

func inferKind(line string) string {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "func "):
		if strings.Contains(line, "(") {
			return "method"
		}
		return "function" //nolint:goconst // symbol kind label
	case strings.HasPrefix(line, "type "):
		if strings.Contains(line, "struct") {
			return "struct"
		}
		if strings.Contains(line, "interface") {
			return "interface"
		}
		return "type"
	case strings.HasPrefix(line, "var "):
		return "variable"
	case strings.HasPrefix(line, "const "):
		return "constant"
	case strings.HasPrefix(line, "class "):
		return "class"
	case strings.HasPrefix(line, "def "):
		return "function"
	case strings.HasPrefix(line, "interface "):
		return "interface"
	case strings.HasPrefix(line, "struct "):
		return "struct"
	case strings.HasPrefix(line, "enum "):
		return "enum"
	case strings.HasPrefix(line, "trait "):
		return "trait"
	}
	return "symbol"
}

func isGoDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && !e.IsDir() {
			return true
		}
	}
	return false
}

var _ = filepath.Join
