# Embedding (Vector DB) Resource

The `embedding` resource converts text into vector embeddings and stores or queries them in a local SQLite-backed vector index. It is the building block for Retrieval-Augmented Generation (RAG) pipelines — index your documents once, then search for the most relevant passages at query time.

It can be used as a primary resource or as an [inline resource](../concepts/inline-resources) inside `before` / `after` blocks.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: indexDocument
  name: Index Document

run:
  embedding:
    model: nomic-embed-text   # embedding model (Ollama default)
    input: "The quick brown fox jumps over the lazy dog."
    collection: documents
    operation: index          # store the embedding
```

---

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `model` | string | — | **Required.** Embedding model name (e.g. `nomic-embed-text`, `text-embedding-3-small`). |
| `backend` | string | `ollama` | Provider: `ollama`, `openai`, `cohere`, `huggingface`. |
| `baseUrl` | string | backend default | Override the backend API base URL. |
| `apiKey` | string | — | API key for cloud providers (OpenAI, Cohere, HuggingFace). |
| `input` | string | — | **Required** for `index` / `search`. Text to embed. Supports [expressions](../concepts/expressions). |
| `operation` | string | `index` | Operation to perform: `index`, `search`, or `delete`. |
| `collection` | string | `embeddings` | Collection name (one SQLite table per collection). |
| `dbPath` | string | `/tmp/kdeps-embedding/<collection>.db` | Explicit path to the SQLite DB file. |
| `topK` | int | `10` | Maximum results returned by `search`. |
| `metadata` | object | — | Key-value metadata stored with each indexed entry. |
| `timeoutDuration` | string | `60s` | API call timeout (e.g. `30s`, `2m`). Alias: `timeout`. |

---

## Operations

### `index` (default)

Embeds `input` and stores the vector + original text in the collection.

```yaml
run:
  embedding:
    model: nomic-embed-text
    input: "KDeps is a YAML-based workflow framework."
    collection: docs
    operation: index
    metadata:
      source: readme
      page: 1
```

**Returns:**

```json
{
  "success": true,
  "operation": "index",
  "collection": "docs",
  "id": 42,
  "dimensions": 768
}
```

### `search`

Embeds `input` (the query) and returns the `topK` most similar entries by cosine similarity.

```yaml
run:
  embedding:
    model: nomic-embed-text
    input: "What is KDeps?"
    collection: docs
    operation: search
    topK: 5
```

**Returns:**

```json
{
  "success": true,
  "operation": "search",
  "collection": "docs",
  "count": 5,
  "results": [
    {
      "id": 42,
      "text": "KDeps is a YAML-based workflow framework.",
      "similarity": 0.97,
      "metadata": { "source": "readme", "page": 1 }
    }
  ]
}
```

Access results in downstream resources:

<div v-pre>

```yaml
metadata:
  requires: [searchDocs]
run:
  apiResponse:
    success: true
    response:
      passages: "{{ get('searchDocs').results }}"
```

</div>

### `delete`

Removes entries from the collection.

- If `input` is non-empty, deletes rows whose `text` exactly matches `input`.
- Otherwise, deletes **all** rows in the collection.

```yaml
run:
  embedding:
    model: nomic-embed-text
    input: "KDeps is a YAML-based workflow framework."
    collection: docs
    operation: delete
```

**Returns:**

```json
{
  "success": true,
  "operation": "delete",
  "collection": "docs",
  "deleted": 1
}
```

---

## Backends

### Ollama (local, default)

Calls Ollama's `POST /api/embed` endpoint. Requires a running Ollama instance and an embedding model.

```bash
# Pull an embedding model
ollama pull nomic-embed-text
```

```yaml
run:
  embedding:
    model: nomic-embed-text
    backend: ollama           # optional — this is the default
    baseUrl: http://localhost:11434  # optional — default Ollama URL
    input: "Text to embed"
    collection: my_docs
```

Good local embedding models:
- `nomic-embed-text` — 768 dimensions, fast, good quality
- `mxbai-embed-large` — 1024 dimensions, higher quality
- `all-minilm` — 384 dimensions, very fast

### OpenAI

Calls `POST /v1/embeddings`. Works with the official OpenAI API and any compatible endpoint (e.g. LM Studio, vLLM).

<div v-pre>

```yaml
run:
  embedding:
    model: text-embedding-3-small
    backend: openai
    apiKey: "{{ env('OPENAI_API_KEY') }}"
    input: "Text to embed"
    collection: my_docs
```

</div>

Available models: `text-embedding-3-small`, `text-embedding-3-large`, `text-embedding-ada-002`.

### Cohere

Calls `POST /v1/embed`.

<div v-pre>

```yaml
run:
  embedding:
    model: embed-english-v3.0
    backend: cohere
    apiKey: "{{ env('COHERE_API_KEY') }}"
    input: "Text to embed"
    collection: my_docs
```

</div>

Available models: `embed-english-v3.0`, `embed-multilingual-v3.0`.

### HuggingFace Inference API

Calls the HuggingFace feature-extraction pipeline endpoint.

<div v-pre>

```yaml
run:
  embedding:
    model: sentence-transformers/all-MiniLM-L6-v2
    backend: huggingface
    apiKey: "{{ env('HF_API_KEY') }}"   # optional for public models
    input: "Text to embed"
    collection: my_docs
```

</div>

---

## Accessing Results

The embedding resource output is available via `get('<actionId>')`:

<div v-pre>

```yaml
# index resource
metadata:
  actionId: indexDoc
run:
  embedding:
    model: nomic-embed-text
    input: "{{ get('body') }}"
    collection: docs

# downstream resource reads the result
metadata:
  actionId: respond
  requires: [indexDoc]
run:
  apiResponse:
    success: true
    response:
      id: "{{ get('indexDoc').id }}"
```

</div>

---

## Embedding as an Inline Resource

Embedding can run **before** or **after** the main resource action:

<div v-pre>

```yaml
run:
  # Index the document before the main LLM call
  before:
    - embedding:
        model: nomic-embed-text
        input: "{{ get('document') }}"
        collection: docs

  chat:
    model: llama3
    prompt: "Summarise: {{ get('document') }}"

  # Search related context after indexing
  after:
    - embedding:
        model: nomic-embed-text
        input: "{{ get('document') }}"
        collection: docs
        operation: search
        topK: 3
```

</div>

---

## Using Expressions in `input`

The `input` field supports [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  embedding:
    model: nomic-embed-text
    input: "{{ get('scraped_text') }}"
    collection: web_pages
    metadata:
      url: "{{ get('url') }}"
```

</div>

---

## Full RAG Example

A complete Retrieval-Augmented Generation workflow: index documents via one endpoint, query them via another.

```yaml
# workflow.yaml
settings:
  name: rag-demo
  targetActionId: respond
  apiServerMode: true
  apiServer:
    routes:
      - path: /index
        methods: [POST]
      - path: /query
        methods: [POST]
```

<div v-pre>

```yaml
# resources/index.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: indexChunk
  name: Index Document Chunk

run:
  restrictToRoutes: [/index]
  restrictToHttpMethods: [POST]

  embedding:
    model: nomic-embed-text
    input: "{{ get('text') }}"
    collection: knowledge_base
    operation: index
    metadata:
      source: "{{ get('source') }}"

  apiResponse:
    success: true
    response:
      id: "{{ get('indexChunk').id }}"
```

```yaml
# resources/query.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: searchChunks
  name: Search Knowledge Base

run:
  restrictToRoutes: [/query]
  restrictToHttpMethods: [POST]

  embedding:
    model: nomic-embed-text
    input: "{{ get('q') }}"
    collection: knowledge_base
    operation: search
    topK: 5
```

```yaml
# resources/respond.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: respond
  name: Generate Answer
  requires: [searchChunks]

run:
  restrictToRoutes: [/query]
  restrictToHttpMethods: [POST]

  chat:
    model: llama3
    prompt: |
      Answer the question using only the context below.
      Context:
      {{ get('searchChunks').results | map(.text) | join('\n---\n') }}

      Question: {{ get('q') }}

  apiResponse:
    success: true
    response:
      answer: "{{ get('respond') }}"
      sources: "{{ get('searchChunks').results }}"
```

</div>

---

## Online RAG Examples

The examples below mirror the full RAG workflow above, but use cloud embedding backends instead of local Ollama. Swap `backend` and `apiKey` to switch providers — the index/search/respond resource structure stays identical.

### OpenAI Embeddings

Uses [`text-embedding-3-small`](https://platform.openai.com/docs/guides/embeddings) (1536 dimensions). Store the API key in an environment variable and reference it with `env()`.

<div v-pre>

```yaml
# workflow.yaml
settings:
  name: openai-rag
  targetActionId: respond
  apiServerMode: true
  apiServer:
    routes:
      - path: /index
        methods: [POST]
      - path: /query
        methods: [POST]
```

```yaml
# resources/index.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: indexChunk
  name: Index with OpenAI

run:
  restrictToRoutes: [/index]
  restrictToHttpMethods: [POST]

  embedding:
    model: text-embedding-3-small
    backend: openai
    apiKey: "{{ env('OPENAI_API_KEY') }}"
    input: "{{ get('text') }}"
    collection: openai_kb
    operation: index
    metadata:
      source: "{{ get('source') }}"

  apiResponse:
    success: true
    response:
      id: "{{ get('indexChunk').id }}"
```

```yaml
# resources/query.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: searchChunks
  name: Search with OpenAI

run:
  restrictToRoutes: [/query]
  restrictToHttpMethods: [POST]

  embedding:
    model: text-embedding-3-small
    backend: openai
    apiKey: "{{ env('OPENAI_API_KEY') }}"
    input: "{{ get('q') }}"
    collection: openai_kb
    operation: search
    topK: 5
```

```yaml
# resources/respond.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: respond
  name: Generate Answer
  requires: [searchChunks]

run:
  restrictToRoutes: [/query]
  restrictToHttpMethods: [POST]

  chat:
    model: gpt-4o-mini
    baseUrl: https://api.openai.com/v1
    apiKey: "{{ env('OPENAI_API_KEY') }}"
    prompt: |
      Answer using only the context below.
      Context:
      {{ get('searchChunks').results | map(.text) | join('\n---\n') }}

      Question: {{ get('q') }}

  apiResponse:
    success: true
    response:
      answer: "{{ get('respond') }}"
      sources: "{{ get('searchChunks').results }}"
```

</div>

**Run:**

```bash
export OPENAI_API_KEY=sk-...
kdeps run workflow.yaml

# Index a document
curl -X POST http://localhost:16394/index \
  -H 'Content-Type: application/json' \
  -d '{"text": "KDeps is a YAML workflow framework.", "source": "readme"}'

# Query
curl -X POST http://localhost:16394/query \
  -H 'Content-Type: application/json' \
  -d '{"q": "What is KDeps?"}'
```

---

### Cohere Embeddings

Uses [`embed-english-v3.0`](https://docs.cohere.com/docs/embed-api) (1024 dimensions).

<div v-pre>

```yaml
# resources/index.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: indexChunk
  name: Index with Cohere

run:
  restrictToRoutes: [/index]
  restrictToHttpMethods: [POST]

  embedding:
    model: embed-english-v3.0
    backend: cohere
    apiKey: "{{ env('COHERE_API_KEY') }}"
    input: "{{ get('text') }}"
    collection: cohere_kb
    operation: index
    metadata:
      source: "{{ get('source') }}"

  apiResponse:
    success: true
    response:
      id: "{{ get('indexChunk').id }}"
```

```yaml
# resources/query.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: searchChunks
  name: Search with Cohere

run:
  restrictToRoutes: [/query]
  restrictToHttpMethods: [POST]

  embedding:
    model: embed-english-v3.0
    backend: cohere
    apiKey: "{{ env('COHERE_API_KEY') }}"
    input: "{{ get('q') }}"
    collection: cohere_kb
    operation: search
    topK: 5
```

</div>

**Run:**

```bash
export COHERE_API_KEY=...
kdeps run workflow.yaml
```

Available multilingual model: `embed-multilingual-v3.0`.

Available Cohere embedding models:

| Model | Dimensions | Notes |
|-------|-----------|-------|
| `embed-english-v3.0` | 1024 | English, high quality |
| `embed-multilingual-v3.0` | 1024 | 100+ languages |
| `embed-english-light-v3.0` | 384 | Faster, smaller English |
| `embed-multilingual-light-v3.0` | 384 | Faster, smaller multilingual |

---

### HuggingFace Embeddings

Uses the [HuggingFace Inference API](https://huggingface.co/docs/api-inference/tasks/feature-extraction). Public models work without an API key; private models or higher rate limits require one.

<div v-pre>

```yaml
# resources/index.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: indexChunk
  name: Index with HuggingFace

run:
  restrictToRoutes: [/index]
  restrictToHttpMethods: [POST]

  embedding:
    model: sentence-transformers/all-MiniLM-L6-v2
    backend: huggingface
    apiKey: "{{ env('HF_API_KEY') }}"
    input: "{{ get('text') }}"
    collection: hf_kb
    operation: index
    metadata:
      source: "{{ get('source') }}"

  apiResponse:
    success: true
    response:
      id: "{{ get('indexChunk').id }}"
```

```yaml
# resources/query.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: searchChunks
  name: Search with HuggingFace

run:
  restrictToRoutes: [/query]
  restrictToHttpMethods: [POST]

  embedding:
    model: sentence-transformers/all-MiniLM-L6-v2
    backend: huggingface
    apiKey: "{{ env('HF_API_KEY') }}"
    input: "{{ get('q') }}"
    collection: hf_kb
    operation: search
    topK: 5
```

</div>

**Run:**

```bash
export HF_API_KEY=hf_...   # optional for public models
kdeps run workflow.yaml
```

Popular HuggingFace embedding models:

| Model | Dimensions | Notes |
|-------|-----------|-------|
| `sentence-transformers/all-MiniLM-L6-v2` | 384 | Fast, good quality |
| `sentence-transformers/all-mpnet-base-v2` | 768 | Higher quality |
| `BAAI/bge-large-en-v1.5` | 1024 | State-of-the-art English |
| `intfloat/multilingual-e5-large` | 1024 | Multilingual |

---

### Compatible with OpenAI API (LM Studio, vLLM, Ollama)

Any server that exposes an OpenAI-compatible `/v1/embeddings` endpoint can be used with `backend: openai` and a custom `baseUrl`.

<div v-pre>

```yaml
# LM Studio (local)
run:
  embedding:
    model: nomic-embed-text-v1.5
    backend: openai
    baseUrl: http://localhost:1234/v1
    apiKey: lm-studio          # LM Studio accepts any non-empty key
    input: "{{ get('text') }}"
    collection: my_docs

# Ollama via OpenAI-compatible endpoint
run:
  embedding:
    model: nomic-embed-text
    backend: openai
    baseUrl: http://localhost:11434/v1
    apiKey: ollama             # Ollama accepts any non-empty key
    input: "{{ get('text') }}"
    collection: my_docs

# vLLM
run:
  embedding:
    model: BAAI/bge-large-en-v1.5
    backend: openai
    baseUrl: http://my-vllm-host:8000/v1
    apiKey: "{{ env('VLLM_API_KEY') }}"
    input: "{{ get('text') }}"
    collection: my_docs
```

</div>

---

## Storage

Embeddings are persisted in a SQLite database (one file per collection):

- **Default location:** `/tmp/kdeps-embedding/<collection>.db`
- **Custom location:** set `dbPath` to any writable path

The database survives process restarts (as long as the file path is stable). For production use, set `dbPath` to a persistent location outside `/tmp`.

<div v-pre>

```yaml
run:
  embedding:
    model: nomic-embed-text
    input: "{{ get('text') }}"
    collection: production_docs
    dbPath: /var/lib/kdeps/embeddings/production_docs.db
```

</div>

Each collection table has this schema:

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `text` | TEXT | Original input text |
| `embedding` | TEXT | JSON-serialized float64 vector |
| `metadata` | TEXT | JSON-serialized metadata object |
| `created_at` | DATETIME | Row creation timestamp |

---

## Similarity

Search uses **cosine similarity** computed in Go — no external dependencies required. Results are returned sorted by similarity (highest first, range −1 to 1).

For best results, use the same model and backend for both indexing and searching.
