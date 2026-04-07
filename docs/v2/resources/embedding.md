# Embedding (Vector DB) Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install embedding`
>
> Usage: `run: { component: { name: embedding, with: { text: "...", apiKey: "...", model: "text-embedding-ada-002" } } }`

The Embedding component converts text into vector embeddings using the OpenAI Embeddings API.
It is the building block for Retrieval-Augmented Generation (RAG) pipelines.

> **Note**: The component generates embeddings via OpenAI only. For local embedding (Ollama, Cohere, HuggingFace) or for index/search/delete vector store operations, use a Python resource directly.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `text` | string | yes | — | Text to embed |
| `apiKey` | string | yes | — | OpenAI API key |
| `model` | string | no | `text-embedding-ada-002` | Embedding model: `text-embedding-ada-002`, `text-embedding-3-small`, `text-embedding-3-large` |

## Using the Embedding Component

```yaml
run:
  component:
    name: embedding
    with:
      text: "The quick brown fox jumps over the lazy dog."
      apiKey: "sk-..."
      model: "text-embedding-3-small"
```

Access the result via `output('<callerActionId>')`. The result includes the embedding vector.

---

## Result Map

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` if embedding was generated. |
| `embedding` | []float | The embedding vector. |
| `model` | string | Model used. |
| `dimensions` | int | Vector dimensions. |

---

## Expression Support

The `text` field supports [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: embedding
    with:
      text: "{{ get('document_text') }}"
      apiKey: "{{ env('OPENAI_API_KEY') }}"
      model: text-embedding-3-small
```

</div>

---

## Full Example: RAG Pipeline

This pipeline embeds a document chunk at index time, then uses the vector in a query.

<div v-pre>

```yaml
# Step 1: Embed a document chunk
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: embedChunk
    name: Embed Document Chunk

  run:
    component:
      name: embedding
      with:
        text: "{{ get('chunk_text') }}"
        apiKey: "{{ env('OPENAI_API_KEY') }}"
        model: text-embedding-3-small

# Step 2: Store the embedding in a SQL table (using the sql resource)
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: storeChunk
    name: Store Chunk
    requires: [embedChunk]

  run:
    sql:
      dsn: "sqlite:///data/kb.db"
      query: |
        INSERT INTO chunks (text, embedding)
        VALUES ('{{ get("chunk_text") }}', '{{ output("embedChunk").embedding | toJSON }}')

# Step 3: Return confirmation
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: confirmIndex
    name: Confirm Indexed
    requires: [storeChunk]

  run:
    apiResponse:
      success: true
      response:
        indexed: true
        dimensions: "{{ output('embedChunk').dimensions }}"
```

</div>

---

## Next Steps

- [SQL Resource](sql) - Store and query embeddings in SQLite or Postgres
- [LLM Resource](llm) - Generate answers from retrieved context
- [Scraper Resource](scraper) - Extract text from documents before embedding
