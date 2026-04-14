# embedding

> Local document store with keyword similarity search (SQLite-backed).

The embedding component provides a lightweight vector-free document store. Documents are indexed by text content and searched by keyword matching. All data is persisted in a local SQLite database. Use it to build retrieval-augmented generation (RAG) pipelines, semantic caches, or simple knowledge bases without external infrastructure.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `operation` | string | yes | — | Operation: `index`, `search`, `delete`, or `upsert` |
| `text` | string | no | `""` | Text to index, search for, or delete |
| `collection` | string | no | `default` | Namespace / collection name |
| `dbPath` | string | no | `/tmp/kdeps-embedding.db` | Path to the SQLite database file |
| `limit` | integer | no | `10` | Max results for `search` operations |

## Operations

| Operation | Description |
|-----------|-------------|
| `index` | Add a new document. Always creates a new entry (no dedup). |
| `upsert` | Add document only if exact text is not already in the collection. |
| `search` | Keyword search — returns documents where `text` appears as a substring. |
| `delete` | Remove all documents whose content exactly matches `text`. |

## Outputs

Accessed via `output('<your-action-id>')`:

### `index`

```json
{ "success": true, "operation": "index", "id": "<uuid>", "dimensions": 4, "collection": "default" }
```

### `upsert`

```json
{ "success": true, "operation": "upsert", "id": "<uuid>", "skipped": false, "collection": "default" }
```

`skipped: true` means the document already existed — no new entry was created.

### `search`

```json
{
  "success": true,
  "operation": "search",
  "count": 2,
  "results": [
    { "id": "<uuid>", "text": "...", "score": 0.99 }
  ],
  "collection": "default"
}
```

### `delete`

```json
{ "success": true, "operation": "delete", "deleted": 1, "collection": "default" }
```

## Usage

Index a document:

```yaml
run:
  component:
    name: embedding
    with:
      operation: index
      text: "kdeps is an AI workflow engine for building local-first agents"
      collection: docs
      dbPath: /data/my-docs.db
  apiResponse:
    success: true
    response:
      docId: "{{ output('indexDoc').id }}"
```

Search for documents:

```yaml
run:
  component:
    name: embedding
    with:
      operation: search
      text: "workflow engine"
      collection: docs
      dbPath: /data/my-docs.db
      limit: 5
  apiResponse:
    success: true
    response:
      results: "{{ output('searchDocs').results }}"
      count: "{{ output('searchDocs').count }}"
```

Upsert (idempotent index):

```yaml
run:
  component:
    name: embedding
    with:
      operation: upsert
      text: "{{ output('prevStep').content }}"
      collection: cache
```

Delete a document:

```yaml
run:
  component:
    name: embedding
    with:
      operation: delete
      text: "document text to remove"
      collection: docs
```

## Requirements

- Python stdlib only (`sqlite3`, `uuid`, `json`, `os`) — no additional packages needed.
- Write access to the directory containing `dbPath`.

## Notes

- Search is substring-based, not semantic. For true semantic search, use a vector store or an LLM re-ranking step.
- The `score` field in search results is always `0.99` (keyword match) — it is a placeholder for future vector scoring.
- Different collections share the same database file but are isolated from each other.
- For persistent storage across runs, set `dbPath` to a stable location outside `/tmp`.
