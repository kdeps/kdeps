# Code Intelligence Resource

The `codeIntelligence:` resource provides structured code navigation using **LSP** (Language Server Protocol) for semantic accuracy, with automatic fallback to [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) when no LSP server is available. Supports Go, Python, Rust, TypeScript/JavaScript, C/C++, Ruby, and Java.

## How it works

```text
kdeps receives codeIntelligence request
  -> detects language from file extension
    -> tries to start LSP server (gopls, pyright, rust-analyzer, etc.)
      -> SUCCESS: sends LSP request, returns semantic results
      -> FAILURE: falls back to rg for text-based grep results
```

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In agent mode, it is also available as built-in tools — no YAML required:

| Tool | Operation | Example |
|------|-----------|---------|
| `code_search` | symbolSearch | `code_search(query="parseInput", path="/src")` |
| `code_definition` | definition | `code_definition(symbol="Config", path="/src/main.go")` |
| `code_references` | references | `code_references(symbol="Config", path="/src/main.go")` |
| `code_symbols` | documentSymbols | `code_symbols(path="/src/main.go")` |
| `code_hover` | hover | `code_hover(symbol="main", path="/src/main.go")` |
| `code_diagnostics` | diagnostics | `code_diagnostics(path="/src/main.go")` |
| `code_index_folder` | indexFolder | `code_index_folder()` — always indexes the CWD |
| `code_graph_file` | graphFile | `code_graph_file(path="/docs/intro.md")` |
| `code_graph_topic` | graphTopic | `code_graph_topic(topic="getting-started")` |
| `code_graph_all` | graphAll | `code_graph_all()` |

## Configuration Options

| Option | Operation | Description |
|--------|-----------|-------------|
| `operation` | all | The code-intelligence operation (required) |
| `path` | all | File or directory to search |
| `query` | symbolSearch | Symbol name or search pattern |
| `symbol` | definition, references, hover | Specific symbol name |
| `languageId` | all | Explicit LSP language ID: `go`, `python`, `rust`, `typescript`, `javascript`, `c`, `cpp`, `ruby`, `java`. Auto-detected from file extension if omitted. |
| `pattern` | all | Glob filter e.g. `"*.go"` (rg fallback only) |
| `language` | all | rg `--type` value (rg fallback only) |
| `context` | all | Lines of context before/after match (rg fallback only) |
| `limit` | all | Maximum results, 0 = unlimited (rg fallback only) |
| `include` | all | rg `--include` patterns (rg fallback only) |
| `exclude` | all | rg `--exclude` patterns (rg fallback only) |
| `recursive` | all | Search subdirectories (rg fallback only) |
| `topic` | graphTopic | Topic/tag to graph, as declared in a file's frontmatter |
| `extensions` | indexFolder | File extensions to index, e.g. `[".md", ".yaml"]`. Defaults to `.md`/`.markdown`/`.txt`/`.yaml`/`.yml`. |
| `graphDBPath` | indexFolder, graphFile, graphTopic, graphAll | Graph index db path. Defaults to `"<path>/.kdeps/graph.db"`, or `"<CWD>/.kdeps/graph.db"` for indexFolder/graphFile (which ignore `path`) and whenever `path` is also omitted. |

> **`path` is ignored for `indexFolder`.** It always indexes the process's current working directory — there is no way to point it at an arbitrary filesystem location, by design (an LLM or workflow can't be tricked into indexing something outside the project it's running in).

## Requirements

LSP servers are **optional but recommended** for semantic accuracy:

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

### Symbol Search

Searches for a symbol or pattern across files. Uses LSP `workspace/symbol` for semantic search, or `rg` regex as fallback.

```yaml
codeIntelligence:
  operation: symbolSearch
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

### Definition

Finds the definition of a symbol. Uses LSP `textDocument/definition` for semantic precision (no false positives on comments or similar names).

```yaml
codeIntelligence:
  operation: definition
  symbol: "ParseConfig"
  path: /path/to/project/main.go
```

### References

Finds all references to a symbol across the codebase. LSP returns cross-file results with type-level accuracy.

```yaml
codeIntelligence:
  operation: references
  symbol: "ParseConfig"
  path: /path/to/project/main.go
  pattern: "*.go"
```

### Document Symbols

Lists all symbols in a file with nesting (methods inside classes, etc.). LSP returns structured `DocumentSymbol[]` with children.

```yaml
codeIntelligence:
  operation: documentSymbols
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

### Hover

Retrieves documentation and type information for a symbol. LSP returns real doc comments and type signatures.

```yaml
codeIntelligence:
  operation: hover
  symbol: "ProcessData"
  path: /path/to/project/main.go
```

### Diagnostics

Runs compiler/linter diagnostics on a file. LSP returns errors and warnings from the language server (e.g., `gopls` for Go, `pyright` for Python). Falls back to `go vet` for Go files when no LSP server is available.

```yaml
codeIntelligence:
  operation: diagnostics
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

### Graphing an Indexed Folder

The four operations above are different from the LSP/rg operations: instead of searching live files, they build and query a small persistent graph database of the current working directory — tracking which files link to which, and which files share a topic. Run kdeps from inside a `docs/` directory (or a source tree, with `extensions` opted in — see below) and ask "what references this file?" or "show me everything tagged `auth`."

References are derived per file type:

| File type | How references are found |
|-----------|---------------------------|
| `.md` / `.markdown` / `.txt` | Markdown links (`[text](path)`) and wikilinks (`[[path]]`) |
| `.html` / `.htm` | `href`/`src` attribute values |
| Go, Python, Rust, TypeScript/JavaScript, C/C++, Ruby, Java | Import/include statements — relative (`from . import x`, `#include "local.h"`) and non-relative/module-style (`import "github.com/x/pkg"`), the latter matched against files actually present in the indexed folder. Heuristic regex extraction, not a full parser — ambiguous or external/stdlib imports are dropped, not guessed. |
| `.json` / `.yaml` / `.yml` (bare, no frontmatter) | No reference extraction |

All of the above (plus any file with a leading `---` YAML frontmatter block) get **topic** extraction from a `topics:`/`tags:` list — for a bare `.json`/`.yaml`/`.yml` file, a top-level `topics:`/`tags:` key works with no frontmatter markers needed.

The default `extensions` (`.md`/`.markdown`/`.txt`/`.yaml`/`.yml`) only cover docs — indexing source code, HTML, or JSON requires opting in explicitly, so pointing `indexFolder` at a project root never silently walks an entire source tree by surprise:

```yaml
codeIntelligence:
  operation: indexFolder
  extensions: [".go", ".md"]  # opt in to Go import graphing alongside docs
```

`indexFolder` must run once before `graphFile`/`graphTopic`/`graphAll` can query the same graph database (same `graphDBPath`, which defaults to `<CWD>/.kdeps/graph.db`).

```text
indexFolder() -> walks the CWD, extracts links + frontmatter topics
              -> writes them into <CWD>/.kdeps/graph.db
                   -> graphFile(path)   returns that file's references + topic-related files
                   -> graphTopic(topic) returns every file tagged `topic` + the full reference graph
                   -> graphAll()        returns the full reference graph + every root file
```

**Index the current working directory:**

```yaml
codeIntelligence:
  operation: indexFolder
  extensions: [".md", ".yaml"]  # optional; defaults to .md/.markdown/.txt/.yaml/.yml
```

`path` is not accepted here — see the note above. Run `kdeps run` (or the `code_index_folder` agent tool) from inside the directory you want indexed.

**Output:**
```json
{
  "success": true,
  "path": "/current/working/directory",
  "filesIndexed": 42
}
```

**Graph a single file** — its references, plus every other indexed file sharing a topic with it:

```yaml
codeIntelligence:
  operation: graphFile
  path: /path/to/docs/intro.md
```

**Output:**
```json
{
  "success": true,
  "path": "/path/to/docs/intro.md",
  "references": {
    "/path/to/docs/intro.md": ["/path/to/docs/setup.md"],
    "/path/to/docs/setup.md": []
  },
  "relatedByTopic": ["/path/to/docs/getting-started.md"]
}
```

**Graph everything tagged with a topic:**

```yaml
codeIntelligence:
  operation: graphTopic
  topic: getting-started
  # path: /path/to/docs  # optional: locate the graph db elsewhere; defaults to <CWD>/.kdeps/graph.db
```

**Graph the whole index** — the full reference graph plus every root file (files nothing else references):

```yaml
codeIntelligence:
  operation: graphAll
```

`references` is always a full adjacency map (`file -> files it links to`); the caller walks it to render a tree or diagram. This mirrors [kartographer](https://github.com/kdeps/kartographer)'s own `DependencyGraph` traversal machinery, reused here through `IndexedGraph`.

## Best Practices

1. **Install LSP servers** for semantic accuracy — avoids regex false positives
2. **Use `languageId` to override** auto-detection when the file extension is ambiguous
3. **Use `pattern` to narrow searches** with the rg fallback
4. **Agent mode tools are zero-config** — LLM can call them without YAML

## See Also

- [Agent Mode](/modes/agent-loop-mode) — built-in `code_*` tools
- [git resource](git) — version control operations
- [Exec Resource](exec) — shell commands
- [searchLocal Resource](search) — local file search
