# Embedding Resource

The `embedding` executor is built into the `kdeps` binary — no installation required. It provides a SQLite-backed keyword store for indexing, searching, upserting, and deleting text documents. Use it as the storage layer for RAG pipelines that run fully on-prem.

## Configuration

```yaml
run:
  embedding:
    operation: "index"                    # required: index | search | upsert | delete
    text: "document content here"         # required for index/search/upsert (optional for delete-all)
    collection: "default"                 # optional namespace (default: "default")
    dbPath: "/data/kdeps-store.db"       # optional path (default: "kdeps-embedding.db")
    limit: 10                             # optional max search results (default: 10)
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `operation` | string | yes | — | `index`, `search`, `upsert`, or `delete` |
| `text` | string | yes* | — | Text to index, query, or delete. *Optional for `delete` (omit to delete entire collection) |
| `collection` | string | no | `"default"` | Namespace for documents |
| `dbPath` | string | no | `"kdeps-embedding.db"` | SQLite database file path |
| `limit` | integer | no | `10` | Max results for `search` |

## Operations

| Operation | Description |
|-----------|-------------|
| `index` | Insert text (ignored if duplicate) |
| `upsert` | Insert or replace text |
| `search` | Case-insensitive keyword search via LIKE |
| `delete` | Delete by text, or whole collection if text is empty |

## Output

| Key | Type | Description |
|-----|------|-------------|
| `operation` | string | The operation that was performed |
| `collection` | string | The collection used |
| `success` | bool | `true` on success |
| `results` | []string | Matching texts (search only) |
| `count` | integer | Number of results (search only) |
| `affected` | integer | Rows deleted (delete only) |
| `json` | string | Full result as JSON string |

## Examples

### Build a RAG pipeline

<div v-pre>

```yaml
# Step 1: Scrape content
metadata:
  actionId: fetch
run:
  scraper:
    url: "{{ get('url') }}"

# Step 2: Index it
metadata:
  actionId: storeDoc
  requires: [fetch]
run:
  embedding:
    operation: "index"
    text: "{{ output('fetch').content }}"
    collection: "knowledge"
    dbPath: "/data/store.db"

# Step 3: Search on user query
metadata:
  actionId: findDocs
run:
  embedding:
    operation: "search"
    text: "{{ get('query') }}"
    collection: "knowledge"
    dbPath: "/data/store.db"
    limit: 5

# Step 4: Answer with context
metadata:
  actionId: answer
  requires: [findDocs]
run:
  chat:
    model: llama3.2:1b
    prompt: |
      Context: {{ output('findDocs').results }}
      Question: {{ get('query') }}
  apiResponse:
    response: "{{ output('answer') }}"
```

</div>

## Collections

Use `collection` to namespace documents — useful for multi-tenant or multi-topic stores:

```yaml
# Index into separate collections
embedding:
  operation: "index"
  text: "..."
  collection: "contracts"

# Search only within one collection
embedding:
  operation: "search"
  text: "termination clause"
  collection: "contracts"
```

---

> **Note**: This uses keyword (LIKE) matching, not vector similarity. For OpenAI vector embeddings, install the component:
> ```bash
> kdeps component install embedding
> ```

## Next Steps

- [Scraper Resource](scraper) - Fetch content to index
- [Search Local Resource](search-local) - Search local files by keyword or glob
- [LLM Resource](llm) - Use search results as context
