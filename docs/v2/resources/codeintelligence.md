# Code Intelligence Resource

The `codeIntelligence:` resource provides structured code navigation operations for searching symbols, finding definitions and references, listing document symbols, viewing hover documentation, and running diagnostics. It uses [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) for fast, cross-language code search and integrates with `go vet` for Go-specific diagnostics.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Basic Usage

```yaml
# resources/search.yaml
actionId: searchCode
name: Search for Symbol
codeIntelligence:
  operation: symbolSearch
  query: "parseRequest"
  path: /path/to/project
```

```yaml
# resources/def.yaml
actionId: findDefinition
name: Find Definition
codeIntelligence:
  operation: definition
  symbol: "ParseConfig"
  path: /path/to/project/main.go
```

## Operations

| Operation | Description |
|-----------|-------------|
| `symbolSearch` | Search for a symbol or pattern across files |
| `definition` | Find symbol definitions |
| `references` | Find all references to a symbol |
| `documentSymbols` | List all symbols in a file or directory |
| `hover` | Show documentation context for a symbol |
| `diagnostics` | Run code diagnostics (Go vet) |

## Configuration Options

| Option | Operation | Description |
|--------|-----------|-------------|
| `operation` | all | The code-intelligence operation (required) |
| `path` | all | File or directory to search |
| `query` | symbolSearch | Symbol name or search pattern |
| `symbol` | definition, references, hover | Specific symbol name |
| `pattern` | all | Glob filter (e.g. `"*.go"`) |
| `language` | all | rg `--type` value (e.g. `"go"`, `"py"`, `"js"`) |
| `context` | all | Lines of context before/after match |
| `limit` | all | Maximum results (0 = unlimited) |
| `include` | all | rg `--include` patterns |
| `exclude` | all | rg `--exclude` patterns |
| `recursive` | all | Search subdirectories |

## Requirements

`codeIntelligence` requires [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) to be installed:

```bash
# macOS
brew install ripgrep

# Ubuntu/Debian
sudo apt install ripgrep

# Arch Linux
sudo pacman -S ripgrep

# Nix
nix-env -i ripgrep
```

If `rg` is not installed, the resource returns a clear error with installation instructions.

## Operation Details

### Symbol Search

Searches for a symbol or pattern across files using rg. Supports glob filtering, language-aware search, and context lines.

```yaml
# resources/symbol-search.yaml
codeIntelligence:
  operation: symbolSearch
  query: "parseInput"
  path: /path/to/project
  pattern: "*.go"
  context: 2
  limit: 20
```

**Output:**
```json
{
  "success": true,
  "symbols": [
    {"file": "src/main.go", "line": 42, "content": "func parseInput(data string) *Config {"}
  ],
  "count": 1
}
```

### Definition

Finds symbol definitions by searching for function, type, variable, and constant declarations. Best results for Go files where declarations follow a `func`/`type`/`var`/`const` pattern.

```yaml
# resources/definition.yaml
codeIntelligence:
  operation: definition
  symbol: "ParseConfig"
  path: /path/to/project
```

### References

Finds all usages of a symbol across the codebase:

```yaml
# resources/references.yaml
codeIntelligence:
  operation: references
  symbol: "ParseConfig"
  path: /path/to/project
  pattern: "*.go"
```

### Document Symbols

Lists all symbols (functions, types, variables, constants) in a file or directory:

```yaml
# resources/document-symbols.yaml
codeIntelligence:
  operation: documentSymbols
  path: /path/to/project/main.go
```

**Output:**
```json
{
  "success": true,
  "symbols": [
    {"name": "main", "kind": "function", "file": "/path/to/project/main.go", "line": 1, "content": "func main() {"},
    {"name": "Config", "kind": "struct", "file": "/path/to/project/main.go", "line": 10, "content": "type Config struct {"}
  ],
  "count": 2
}
```

### Hover

Retrieves the definition of a symbol with surrounding context to show documentation:

```yaml
# resources/hover.yaml
codeIntelligence:
  operation: hover
  symbol: "ProcessData"
  path: /path/to/project
```

### Diagnostics

Runs `go vet` on the specified path and returns any diagnostics:

```yaml
# resources/diagnostics.yaml
codeIntelligence:
  operation: diagnostics
  path: /path/to/project
```

For Go files, this runs `go vet` and parses the output into structured diagnostics:

```json
{
  "success": true,
  "diagnostics": [
    {"file": "src/main.go", "line": "42", "message": "ineffective assignment", "tool": "go vet"}
  ],
  "count": 1
}
```

## Best Practices

1. **Install ripgrep** - required for all operations
2. **Use `pattern` to narrow searches** - limit to relevant file types with globs
3. **Set `context` for readability** - adds surrounding lines to results
4. **Use `limit` on large codebases** - prevent excessive output
5. **Combine with `chat:` in coding agent workflows** - search code, then analyze with an LLM

## See Also

- [git resource](git) - version control operations
- [Exec Resource](exec) - shell commands
- [searchLocal Resource](search) - local file search
