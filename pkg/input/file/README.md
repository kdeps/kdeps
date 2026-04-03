# pkg/input/file

File input runner for KDeps workflows.

## What it does

When a workflow configures `sources: [file]`, the file runner:

1. Uses the **`--file` CLI argument** if provided (highest priority).
2. Reads file content from **stdin** (raw text *or* JSON `{"path":"…","content":"…"}`).
3. Falls back to the **`KDEPS_FILE_PATH`** environment variable if stdin is empty.
4. Falls back to the configured **`input.file.path`** field if neither is set.
5. If only a path was supplied (not inline content), reads the file from disk.
6. Executes the workflow **once** with the file content available to all resources.
7. Exits after execution (stateless single-shot mode).

## Files

| File | Purpose |
|------|---------|
| `runner.go` | `Run()` / `RunWithArg()` public entry points; `runWithReader()` testable core; `readFileInput()` resolution logic |
| `runner_test.go` | Black-box tests for `readFileInput` (all resolution paths, error cases) |
| `runner_internal_test.go` | White-box tests for `runWithReader`, `Run`, `RunWithArg` (success + error paths, 100% coverage) |

## Key types / functions

```go
// Public entry point — reads from os.Stdin (no explicit file arg).
func Run(ctx context.Context, workflow *domain.Workflow, engine *executor.Engine, logger *slog.Logger) error

// Public entry point with explicit file path — argPath takes highest priority.
func RunWithArg(ctx context.Context, workflow *domain.Workflow, engine *executor.Engine, logger *slog.Logger, argPath string) error

// Testable core — reads from any io.Reader.
func runWithReader(ctx context.Context, workflow *domain.Workflow, engine *executor.Engine, logger *slog.Logger, r io.Reader, argPath string) error

// Input resolution — returns parsed fileInput{Path, Content} or an error.
func readFileInput(r io.Reader, cfg *domain.InputConfig, argPath string) (fileInput, error)
```

## Configuration

```yaml
settings:
  input:
    sources: [file]
    file:
      path: /optional/default/path.txt  # fallback path when stdin and KDEPS_FILE_PATH are empty
```

## Usage

```bash
# Pass file path directly as a CLI argument (highest priority)
./kdeps run workflow.yaml --file /path/to/document.txt

# Pipe raw text via stdin
cat document.txt | ./kdeps run workflow.yaml

# Pipe JSON with a file path — file is read from disk
echo '{"path":"/tmp/doc.txt"}' | ./kdeps run workflow.yaml

# Pipe JSON with inline content
echo '{"path":"/tmp/doc.txt","content":"hello world"}' | ./kdeps run workflow.yaml

# Use environment variable
KDEPS_FILE_PATH=/tmp/doc.txt ./kdeps run workflow.yaml
```

## Accessing file data in resources

| Expression | Value |
|------------|-------|
| `input("content")` or `input("fileContent")` | The file's text content |
| `input("path")` or `input("filePath")` | The source file path (if known) |
| `get("inputFileContent")` | The file's text content (via `get()`) |
| `get("inputFilePath")` | The source file path (via `get()`) |

### Example resource

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: summarize

run:
  chat:
    model: llama3.2:3b
    prompt: "Summarize the following document:\n\n{{ input('fileContent') }}"
```

## Input resolution order

```
--file CLI argument (highest priority)
  └─ path is read from disk → content set
stdin (raw text or JSON)
  └─ JSON {"path": "...", "content": "..."}
       ├─ content present → use inline content directly
       └─ path only → read file from disk
  └─ raw text → use as content directly
  └─ empty → check KDEPS_FILE_PATH env var
               └─ empty → check input.file.path config
                           └─ empty → error: no file input provided
```

## See also

- [docs/v2/tutorials/file-input.md](../../../../docs/v2/tutorials/file-input.md) — step-by-step tutorial
- [docs/v2/concepts/input-sources.md](../../../../docs/v2/concepts/input-sources.md) — full input sources reference
- `pkg/input/bot/stateless.go` — similar single-shot execution pattern for bot input
