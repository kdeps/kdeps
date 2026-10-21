# Code navigation

The `code_*` components provide structured code navigation using LSP (Language Server Protocol) for semantic accuracy, with automatic fallback to [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) when no LSP server is available. Supports Go, Python, Rust, TypeScript/JavaScript, C/C++, Ruby, and Java. Each operation is its own component, and the same tool is a built-in in agent mode. The work runs in the `code_*` kdeps API tools; the component only passes the inputs through.

## How it works

```text
component: code_search -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/code_search/call
  -> detects the language from the path extension
    -> tries an LSP server (gopls, pyright, rust-analyzer, ...)
      -> SUCCESS: sends the LSP request, returns semantic results
      -> FAILURE: falls back to rg for text-based grep results
```

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In workflow mode each operation is a `component:` call. In agent mode the same ten operations are built-in tools - no YAML required:

| Component | Operation | Agent tool |
|------|-----------|---------|
| `code_search` | symbolSearch | `code_search(query="parseInput", path="/src")` |
| `code_definition` | definition | `code_definition(symbol="Config", path="/src/main.go")` |
| `code_references` | references | `code_references(symbol="Config", path="/src/main.go")` |
| `code_symbols` | documentSymbols | `code_symbols(path="/src/main.go")` |
| `code_hover` | hover | `code_hover(symbol="main", path="/src/main.go")` |
| `code_diagnostics` | diagnostics | `code_diagnostics(path="/src/main.go")` |
| `code_index_folder` | indexFolder | `code_index_folder()` - always indexes the CWD |
| `code_graph_file` | graphFile | `code_graph_file(path="/docs/intro.md")` |
| `code_graph_topic` | graphTopic | `code_graph_topic(topic="getting-started")` |
| `code_graph_all` | graphAll | `code_graph_all()` |

## Basic usage

```yaml
# resources/search.yaml
actionId: search
name: Search
component:
  name: code_search
  with:
    query: "parseInput"       # symbol name or search pattern
    path: /path/to/project    # file or directory to search
    pattern: "*.go"           # optional: file glob filter (ripgrep fallback only)
    limit: 20                 # optional: matches per file (ripgrep fallback only)
```

The component returns the tool's JSON body as text. Read fields with `fromJSON`:

```yaml
# resources/respond.yaml
actionId: respond
name: Respond
requires: [search]
apiResponse:
  success: true
  response:
    count: "{{ fromJSON(output('search')).count }}"
    symbols: "{{ fromJSON(output('search')).symbols }}"
```

## Inputs

| Input | Used by | Default | Description |
|--------|-----------|-------------|-------------|
| `path` | `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics`, `code_graph_file` (required); `code_graph_topic`, `code_graph_all` (optional) | - | File or directory to search. Ignored by `code_index_folder` |
| `query` | `code_search` (required) | - | Symbol name or search pattern |
| `symbol` | `code_definition`, `code_references`, `code_hover` (required) | - | Specific symbol name |
| `languageId` | all search operations | auto-detected | Explicit LSP language ID: `go`, `python`, `rust`, `typescript`, `javascript`, `c`, `cpp`, `ruby`, `java` |
| `pattern` | `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_diagnostics` | - | File glob filter, e.g. `"*.go"` (rg fallback only) |
| `language` | `code_search`, `code_definition`, `code_references`, `code_symbols` | - | rg `--type` value, e.g. `go` (rg fallback only) |
| `context` | `code_search`, `code_definition`, `code_references`, `code_symbols` | `0` | Lines of context before/after each match (rg fallback only) |
| `limit` | `code_search`, `code_definition`, `code_references`, `code_symbols` | `0` | Maximum matches per file, 0 = unlimited (rg fallback only) |
| `topic` | `code_graph_topic` (required) | - | Topic/tag to graph, as declared in a file's frontmatter |
| `extensions` | `code_index_folder` | `.md`/`.markdown`/`.txt`/`.yaml`/`.yml` | File extensions to index, e.g. `[".md", ".yaml"]` |
| `graphDBPath` | `code_index_folder`, `code_graph_file`, `code_graph_topic`, `code_graph_all` | `~/.kdeps/projects/<folder>-<hash>/graph.db` | Graph index db path. Nothing is written into the indexed folder |

> `path` is ignored for `code_index_folder`. It always indexes the process's current working directory - there is no way to point it at an arbitrary filesystem location, by design (an LLM or workflow cannot be tricked into indexing something outside the project it is running in).

## Requirements

LSP servers are optional but recommended for semantic accuracy:

| Language | LSP Server | Install |
|----------|-----------|---------|
| Go | `gopls` | `go install golang.org/x/tools/gopls@latest` |
| Python | `pyright` | `npm install -g pyright` |
| Rust | `rust-analyzer` | `rustup component add rust-analyzer` |
| TypeScript/JS | `typescript-language-server` | `npm install -g typescript-language-server` |
| C/C++ | `clangd` | OS package manager |
| Ruby | `solargraph` | `gem install solargraph` |

`ripgrep` (`rg`) is required for the fallback path:

```bash
brew install ripgrep       # macOS
sudo apt install ripgrep   # Ubuntu/Debian
```

## Operations

## Symbol search

Searches for a symbol or pattern across files. Uses LSP `workspace/symbol` for semantic search, or `rg` regex as fallback.

```yaml
component:
  name: code_search
  with:
    query: "parseInput"
    path: /path/to/project
    pattern: "*.go"
    limit: 20
```

**Output:**
```json
{
  "success": true,
  "symbols": [
    {"name": "parseInput", "kind": 12, "file": "/path/to/project/src/main.go"}
  ],
  "count": 1
}
```

## Definition

Finds the definition of a symbol. Uses LSP `textDocument/definition` for semantic precision (no false positives on comments or similar names).

```yaml
component:
  name: code_definition
  with:
    symbol: "ParseConfig"
    path: /path/to/project/main.go
```

## References

Finds all references to a symbol across the codebase. LSP returns cross-file results with type-level accuracy.

```yaml
component:
  name: code_references
  with:
    symbol: "ParseConfig"
    path: /path/to/project/main.go
    pattern: "*.go"
```

## Document symbols

Lists all symbols in a file with nesting (methods inside classes, etc.). LSP returns structured `DocumentSymbol[]` with children.

```yaml
component:
  name: code_symbols
  with:
    path: /path/to/project/main.go
```

**Output (LSP, Go):**
```json
{
  "success": true,
  "symbols": [
    {"name": "main", "kind": "function"},
    {"name": "Greeter", "kind": "struct"},
    {"name": "Greet", "kind": "method"}
  ],
  "count": 3
}
```

## Hover

Retrieves documentation and type information for a symbol. LSP returns real doc comments and type signatures.

```yaml
component:
  name: code_hover
  with:
    symbol: "ProcessData"
    path: /path/to/project/main.go
```

## Diagnostics

Runs compiler/linter diagnostics on a file. LSP returns errors and warnings from the language server (e.g., `gopls` for Go, `pyright` for Python). Falls back to `go vet` for Go files when no LSP server is available.

```yaml
component:
  name: code_diagnostics
  with:
    path: /path/to/project/main.go
```

**Output:**
```json
{
  "success": true,
  "diagnostics": [
    {"message": "unused variable x", "severity": "2", "source": "compiler"}
  ],
  "count": 1
}
```

## Graphing an indexed folder

`code_index_folder`/`code_graph_file`/`code_graph_topic`/`code_graph_all` are different from the LSP/rg operations above: instead of searching live files, they build and query a small persistent graph database of the current working directory, tracking which files link to which and which share a topic (from `topics:`/`tags:` frontmatter).

See [Graphing an Indexed Folder](/workflow/resources/code-graph) for the full reference: which file types get reference extraction, the `extensions` opt-in, and worked examples for all four operations.

## Best practices

1. **Install LSP servers** for semantic accuracy - avoids regex false positives
2. **Use languageId to override** auto-detection when the file extension is ambiguous
3. **Use pattern to narrow searches** with the rg fallback
4. **Agent mode tools are zero-config** - LLM can call them without YAML

## See also

- [Agent mode](/agent/) - built-in `code_*` tools
- [Git component](/workflow/resources/git) - version control operations
- [exec component](/workflow/resources/exec) - shell commands
- [search_local component](/workflow/resources/search-local) - local file search
