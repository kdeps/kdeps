# embedding component

The `embedding` component runs one embedding operation and returns a structured result: a SQLite keyword store for `index`, `search`, `upsert`, and `delete`, an embedding backend for `vectorize` and `embed_query`, and Cohere for `rerank`. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its call is the kdeps API's `embedding` tool, which runs the native embedding executor. A workflow that uses `embedding:` directly - primary or in a `before:`/`after:` step - is rejected.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

```d2
direction: right
call: "component: embedding" {shape: oval}
api: "POST /tools/embedding/call"
sqlite: "SQLite keyword store\nindex, search, upsert, delete"
backend: "embedding backend\nvectorize, embed_query"
cohere: "Cohere\nrerank"
out: "result map" {shape: oval}
call -> api
api -> sqlite: "operation index/search/upsert/delete"
api -> backend: "operation vectorize/embed_query"
api -> cohere: "operation rerank"
sqlite -> out
backend -> out
cohere -> out
```

## Configuration

<div v-pre>

```yaml
# resources/store.yaml
actionId: store
name: Store Document
component:
  name: embedding                   # from the components asset set; a components/embedding/ of your own wins
  with:
    operation: index                # required: index | search | upsert | delete | vectorize | embed_query | rerank
    text: "{{ get('text') }}"       # text to index, search for, upsert or delete
    collection: docs                # optional namespace (default: "default")
    dbPath: /data/store.db          # optional SQLite path; default from the kdeps embedding defaults
    limit: 5                        # optional max search results
```

</div>

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `operation` | string | yes | - | `index`, `search`, `upsert`, `delete`, `vectorize`, `embed_query`, or `rerank` |
| `text` | string | see below | - | Text to index, search for, upsert, or delete. Required for `index`, `search`, and `upsert`; empty deletes the whole collection on `delete` |
| `collection` | string | no | `"default"` | Namespace inside the SQLite database |
| `dbPath` | string | no | kdeps embedding default | SQLite database file path |
| `limit` | number | no | kdeps embedding default | Max results for `search` |
| `inputs` | array | yes for `vectorize`, `embed_query` | - | Texts to embed |
| `model` | string | no | backend default | Embedding model, e.g. `text-embedding-3-small` |
| `backend` | string | no | kdeps LLM backend | `openai`, `ollama`, or `google` |
| `baseURL` | string | no | - | Custom base URL for an OpenAI-compatible backend |
| `rerankQuery` | string | yes for `rerank` | - | Query the candidate documents are ranked against |
| `rerankDocuments` | array | yes for `rerank` | - | Candidate documents to rerank |
| `rerankTopN` | number | no | `0` (all) | Number of top reranked results |

## Operations

| Operation | Backing | Description |
|-----------|---------|-------------|
| `index` | SQLite | Insert text (ignored if duplicate) |
| `upsert` | SQLite | Insert or replace text |
| `search` | SQLite | Case-insensitive keyword search via `LIKE` |
| `delete` | SQLite | Delete by text, or the whole collection when `text` is empty |
| `vectorize` | embedding backend | Convert `inputs` into vectors |
| `embed_query` | embedding backend | Convert `text` into a query vector |
| `rerank` | Cohere | Rank `rerankDocuments` against `rerankQuery` |

## Output

```json
{
  "operation": "search",
  "collection": "docs",
  "query": "compiled language",
  "results": ["Go is a statically typed, compiled programming language."],
  "count": 1,
  "json": "{\"collection\":\"docs\",...}"
}
```

Read fields with `output('store').results`, `output('store').count`, and so on.

| Key | Type | Description |
|-----|------|-------------|
| `operation` | string | The operation that was performed |
| `collection` | string | The collection used |
| `success` | bool | `true` on success |
| `results` | []string | Matching texts (search only) |
| `count` | integer | Number of results (search only) |
| `affected` | integer | Rows deleted (delete only) |
| `json` | string | Full result as a JSON string |

A failed operation does not return this map: the call fails and `onError:` on the resource decides what happens next.

## Examples

### Build a RAG pipeline

<div v-pre>

```yaml
# Step 1: Scrape content
actionId: fetch
component:
  name: web_scraper
  with:
    url: "{{ get('url') }}"

# Step 2: Index it
actionId: storeDoc
requires: [fetch]
component:
  name: embedding
  with:
    operation: index
    text: "{{ output('fetch').content }}"
    collection: knowledge
    dbPath: /data/store.db

# Step 3: Search on user query
actionId: findDocs
component:
  name: embedding
  with:
    operation: search
    text: "{{ get('query') }}"
    collection: knowledge
    dbPath: /data/store.db
    limit: 5

# Step 4: Answer with context
actionId: answer
requires: [findDocs]
chat:
  model: llama3.2:1b
  prompt: |
    Context: {{ output('findDocs').results }}
    Question: {{ get('query') }}
apiResponse:
  response: "{{ output('answer') }}"
```

</div>

### Vector embeddings

<div v-pre>

```yaml
# resources/embed.yaml
actionId: embed
component:
  name: embedding
  with:
    operation: vectorize
    inputs: ["{{ get('text') }}"]    # texts to embed
    model: text-embedding-3-small
    backend: openai                  # KDEPS_OPENAI_API_KEY or ~/.kdeps/config.yaml
```

</div>

## Collections

Use `collection` to namespace documents - useful for multi-tenant or multi-topic stores:

<div v-pre>

```yaml
# Index into separate collections
component:
  name: embedding
  with:
    operation: index
    text: "..."
    collection: contracts

# Search only within one collection
component:
  name: embedding
  with:
    operation: search
    text: termination clause
    collection: contracts
```

</div>

---

> **Note**: `index`, `search`, `upsert`, and `delete` use keyword (`LIKE`) matching, not vector similarity. Use `vectorize` and `embed_query` with a backend for vector embeddings.

## See also

- [Search resource](/workflow/resources/search) - search local files and the web
- [LLM resource](/workflow/resources/llm) - use search results as context
