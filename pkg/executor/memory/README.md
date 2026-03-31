# pkg/executor/memory

Implements the `memory:` resource executor for KDeps.

## What it does

Provides **semantic, persistent experience storage** for AI agents. Agents call this executor to:

- **consolidate** - embed content and insert it into a local SQLite vector store
- **recall** - embed a query, compute cosine similarity against all stored entries, return top-K results
- **forget** - delete entries by exact content match, or wipe an entire category

The DB schema, embedding pipeline, and cosine-similarity logic are identical to `pkg/executor/embedding/` — the distinction is conceptual (agent experience vs. document corpus) and reflected in defaults (`category` vs. `collection`, `/tmp/kdeps-memory/` vs. `/tmp/kdeps-embedding/`).

## Files

| File | Purpose |
|------|---------|
| `executor.go` | `Executor` struct, `Execute()` dispatch, all three operations, embedding HTTP clients for all four backends |
| `executor_test.go` | Unit tests using an `httptest.Server` mock (no real Ollama required) |

## Key types

```go
// Entry point - registered in cmd/run.go via registry.SetMemoryExecutor
func NewAdapter(logger *slog.Logger) executor.ResourceExecutor

// For tests - inject a mock *http.Client
func NewAdapterWithClient(logger *slog.Logger, client *http.Client) executor.ResourceExecutor
```

Input config type: `*domain.MemoryConfig` (defined in `pkg/domain/resource.go`).

## Return shapes

**consolidate:**
```json
{ "success": true, "operation": "consolidate", "id": 7, "category": "memories", "dimensions": 768 }
```

**recall:**
```json
{
  "operation": "recall", "category": "memories", "count": 2,
  "memories": [{ "id": 7, "content": "...", "similarity": 0.94, "metadata": {} }]
}
```

**forget:**
```json
{ "success": true, "operation": "forget", "category": "memories" }
```

## Backends

All four embedding providers supported: `ollama` (default), `openai`, `cohere`, `huggingface`. Backend selection and URL/key resolution mirrors `pkg/executor/embedding/`.

## Storage

One SQLite file per category: `/tmp/kdeps-memory/<category>.db` (default). Override with `dbPath`. The `consolidated_at` timestamp is injected into metadata automatically on every consolidate.

## Running tests

```bash
go test ./pkg/executor/memory/...
```

No external services required - the test suite starts an `httptest.Server` that returns deterministic mock embeddings.
