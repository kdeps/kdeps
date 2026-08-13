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
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

//nolint:gochecknoglobals // afero filesystem abstraction; enables test injection
var AppFS afero.Fs = afero.NewOsFs()

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
	lsp    *lspManager
}

// NewExecutor creates a new codeIntelligence executor.
func NewExecutor() *Executor {
	return &Executor{
		runner: &DefaultRunner{},
		lsp:    newLSPManager(),
	}
}

// NewExecutorWithRunner creates a new executor with custom runner (for testing).
func NewExecutorWithRunner(runner Runner) *Executor {
	return &Executor{runner: runner}
}

// Execute dispatches to the appropriate code-intelligence operation.
// Tries LSP first; falls back to rg if no LSP server is available.
func (e *Executor) Execute(
	_ *executor.ExecutionContext,
	config *domain.CodeIntelligenceConfig,
) (interface{}, error) {
	if config.Operation == "" {
		return nil, errors.New("codeIntelligence: operation is required")
	}
	if err := validateSearchRoot(config.Path); err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	if isGraphOperation(config.Operation) {
		return e.executeGraph(config)
	}

	// Try LSP first if we can detect a language server for this file.
	langID := config.LanguageID
	if langID == "" && config.Path != "" {
		langID = languageIDFromPath(config.Path)
	}
	if e.lsp != nil && langID != "" {
		client, err := e.lsp.getServer(langID, "", config.Path)
		if err == nil {
			return e.executeLSP(client, config, langID)
		}
		// LSP unavailable — fall through to rg.
	}

	return e.executeRG(config)
}

// executeLSP dispatches an LSP operation.
//
//nolint:exhaustive // graph ops (indexFolder/graphFile/graphTopic/graphAll) are routed to executeGraph before this is reached
func (e *Executor) executeLSP(client *lspClient, config *domain.CodeIntelligenceConfig, _ string) (interface{}, error) {
	switch config.Operation {
	case domain.CodeIntOpSymbolSearch:
		return e.lspSymbolSearch(client, config)
	case domain.CodeIntOpDefinition:
		return e.lspDefinition(client, config)
	case domain.CodeIntOpReferences:
		return e.lspReferences(client, config)
	case domain.CodeIntOpDocumentSymbols:
		return e.lspDocumentSymbols(client, config)
	case domain.CodeIntOpHover:
		return e.lspHover(client, config)
	case domain.CodeIntOpDiagnostics:
		return e.lspDiagnostics(client, config)
	default:
		return nil, fmt.Errorf("codeIntelligence: unsupported operation %q", config.Operation)
	}
}

// executeRG dispatches an rg-based operation (fallback).
//
//nolint:exhaustive // graph ops (indexFolder/graphFile/graphTopic/graphAll) are routed to executeGraph before this is reached
func (e *Executor) executeRG(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	switch config.Operation {
	case domain.CodeIntOpSymbolSearch:
		return e.rgSymbolSearch(config)
	case domain.CodeIntOpDefinition:
		return e.rgDefinition(config)
	case domain.CodeIntOpReferences:
		return e.rgReferences(config)
	case domain.CodeIntOpDocumentSymbols:
		return e.rgDocumentSymbols(config)
	case domain.CodeIntOpHover:
		return e.rgHover(config)
	case domain.CodeIntOpDiagnostics:
		return e.rgDiagnostics(config)
	default:
		return nil, fmt.Errorf("codeIntelligence: unsupported operation %q", config.Operation)
	}
}

// --- helpers ---

// matchesToLocations converts rg match results into the standard {file, line, content} shape
// used by symbolSearch, definition, and references operations.
func matchesToLocations(matches []rgMatch) []map[string]interface{} {
	locs := make([]map[string]interface{}, 0, len(matches))
	for _, m := range matches {
		locs = append(locs, map[string]interface{}{
			"file":          m.Data.Path.Text,
			"line":          m.Data.LineNumber,
			ciResultContent: strings.TrimSpace(m.Data.Lines.Text),
		})
	}
	return locs
}

// symbolsFromMatches converts rg match results into the standard {name, kind, file, line, content}
// shape used by documentSymbols operations.
func symbolsFromMatches(matches []rgMatch) []map[string]interface{} {
	syms := make([]map[string]interface{}, 0, len(matches))
	for _, m := range matches {
		syms = append(syms, map[string]interface{}{
			"name":          extractSymbolName(m.Data.Lines.Text),
			"kind":          inferKind(m.Data.Lines.Text),
			"file":          m.Data.Path.Text,
			"line":          m.Data.LineNumber,
			ciResultContent: strings.TrimSpace(m.Data.Lines.Text),
		})
	}
	return syms
}

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

// rgError classifies a non-recoverable rg failure.
func (e *Executor) rgError(stderr string, err error) error {
	if strings.Contains(stderr, "executable file not found") || strings.Contains(err.Error(), "not found") {
		return errors.New(
			"ripgrep (rg) is not installed. Install it with: brew install ripgrep (macOS) or apt install ripgrep (Linux)",
		)
	}
	if strings.TrimSpace(stderr) == "" {
		return fmt.Errorf("rg failed: %w", err)
	}
	return fmt.Errorf("rg failed: %s", strings.TrimSpace(stderr))
}

// forbiddenSearchRoots are directories code_search must never recurse: walking
// them is slow and hits unreadable system paths (/dev/fd, caches).
//
//nolint:gochecknoglobals // static denylist
var forbiddenSearchRoots = map[string]bool{
	"/": true, "/dev": true, "/proc": true, "/sys": true,
	"/private": true, "/System": true, "/Library": true, "/usr": true,
	"/etc": true, "/var": true, "/bin": true, "/opt": true,
}

// forbiddenSearchRootNames are directory names that are dangerous to recurse
// into when found as an immediate child of a Windows drive root (e.g.
// C:\Windows, or C:\dev / C:\Library / C:\usr / C:\etc as resolved from
// Unix-style paths via filepath.Abs on Windows).
//
//nolint:gochecknoglobals // static denylist
var forbiddenSearchRootNames = map[string]bool{
	"dev": true, "proc": true, "sys": true, "private": true,
	"system": true, "library": true, "usr": true, "etc": true,
	"var": true, "bin": true, "opt": true, "windows": true,
	"program files": true, "program files (x86)": true, "programdata": true,
}

// validateSearchRoot rejects filesystem roots and system directories so a
// stray or over-broad path never turns code_search into a whole-disk walk.
func validateSearchRoot(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path // Abs only fails if the cwd is unavailable; check the raw path
	}
	clean := filepath.Clean(abs)
	if forbiddenSearchRoots[clean] || isForbiddenWindowsSearchRoot(clean) {
		return fmt.Errorf(
			"codeIntelligence: refusing to search system directory %q — pass a project path",
			clean,
		)
	}
	if home, herr := os.UserHomeDir(); herr == nil && home != "" && clean == filepath.Clean(home) {
		return fmt.Errorf(
			"codeIntelligence: refusing to search the entire home directory %q — pass a project subdirectory",
			clean,
		)
	}
	return nil
}

// isForbiddenWindowsSearchRoot reports whether clean is a Windows drive root
// (e.g. C:\) or an immediate child of one matching forbiddenSearchRootNames
// (covers both a real system dir like C:\Windows and the Windows-resolved
// form of Unix-style literals like C:\dev, C:\usr, C:\etc).
func isForbiddenWindowsSearchRoot(clean string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	vol := filepath.VolumeName(clean)
	if vol == "" {
		return false
	}
	root := vol + string(filepath.Separator)
	if clean == root {
		return true
	}
	return filepath.Dir(clean) == root && forbiddenSearchRootNames[strings.ToLower(filepath.Base(clean))]
}

func (e *Executor) buildRGArgs(config *domain.CodeIntelligenceConfig, extra ...string) []string {
	// --no-messages: don't fail on unreadable paths (permission denied, bad
	// inherited descriptors under /dev/fd) — skip them and keep searching.
	args := []string{"--json", "--line-number", "--no-messages"}
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

// ripgrep exit codes: 1 = no matches, 2 = a recoverable error occurred during
// the search (e.g. an unreadable path) but results were still produced.
const (
	rgExitNoMatch = 1
	rgExitPartial = 2
)

func (e *Executor) runRG(args []string) ([]rgMatch, error) {
	stdout, stderr, err := e.runner.Run("rg", args...)
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, e.rgError(stderr, err)
		}
		switch exitErr.ExitCode() {
		case rgExitNoMatch:
			return nil, nil
		case rgExitPartial:
			// Some paths were unreadable, but rg searched the rest; fall through
			// and use whatever it output rather than failing the whole call.
		default:
			return nil, e.rgError(stderr, err)
		}
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

func (e *Executor) rgSymbolSearch(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Query == "" {
		return nil, errors.New("codeIntelligence: query is required for symbolSearch")
	}
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for symbolSearch")
	}

	args := e.buildRGArgs(config, config.Query)
	args = append(args, config.Path)

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	symbols := matchesToLocations(matches)
	return result(true, map[string]interface{}{
		"symbols":     symbols,
		ciResultCount: len(symbols),
	}), nil
}

func (e *Executor) rgDefinition(config *domain.CodeIntelligenceConfig) (interface{}, error) {
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
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	defs := matchesToLocations(matches)
	return result(true, map[string]interface{}{
		"definitions": defs,
		ciResultCount: len(defs),
	}), nil
}

func (e *Executor) rgReferences(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Symbol == "" {
		return nil, errors.New("codeIntelligence: symbol is required for references")
	}

	args := e.buildRGArgs(config, "--sort", "path", regexEscape(config.Symbol))
	if config.Path != "" {
		args = append(args, config.Path)
	}

	matches, err := e.runRG(args)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	refs := matchesToLocations(matches)
	return result(true, map[string]interface{}{
		"references":  refs,
		ciResultCount: len(refs),
	}), nil
}

func (e *Executor) rgDocumentSymbols(config *domain.CodeIntelligenceConfig) (interface{}, error) {
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
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	symbols := symbolsFromMatches(matches)
	return result(true, map[string]interface{}{
		"symbols":     symbols,
		ciResultCount: len(symbols),
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
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	symbols := symbolsFromMatches(matches)
	return result(true, map[string]interface{}{
		"symbols":     symbols,
		ciResultCount: len(symbols),
	}), nil
}

func (e *Executor) rgHover(config *domain.CodeIntelligenceConfig) (interface{}, error) {
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
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}

	var hoverInfo []map[string]interface{}
	for _, m := range matches {
		hoverInfo = append(hoverInfo, map[string]interface{}{
			"file":          m.Data.Path.Text,
			"line":          m.Data.LineNumber,
			ciResultContent: strings.TrimSpace(m.Data.Lines.Text),
		})
	}

	return result(true, map[string]interface{}{
		"hover":       hoverInfo,
		ciResultCount: len(hoverInfo),
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

func (e *Executor) rgDiagnostics(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for diagnostics")
	}

	var diagnostics []map[string]interface{}
	if strings.HasSuffix(config.Path, ".go") || isGoDir(config.Path) {
		diagnostics = e.goVetDiagnostics(config)
	}

	return result(true, map[string]interface{}{
		"diagnostics": diagnostics,
		ciResultCount: len(diagnostics),
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
		return "function"
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
	info, err := AppFS.Stat(path)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	entries, err := afero.ReadDir(AppFS, path)
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
