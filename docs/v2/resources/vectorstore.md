# Vector Store Resource

The `vectorStore:` resource adds documents to, and runs similarity search against, an external vector database. Unlike `embedding:` (SQLite keyword/vector index built into kdeps), `vectorStore:` talks to a real vector database service -- Qdrant, Chroma, Pinecone, pgvector, and more -- for production-scale RAG.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode).

## Basic Usage

```yaml
# resources/index.yaml
actionId: index
name: Index Documents
vectorStore:
  provider: qdrant
  url: http://localhost:6333
  collection: docs
  operation: add_documents
  documents:
    - content: "kdeps is a YAML-based AI agent framework."
      metadata: { source: readme }
  embedModel: text-embedding-3-small
```

```yaml
# resources/search.yaml
actionId: search
name: Semantic Search
requires: [index]
vectorStore:
  provider: qdrant
  url: http://localhost:6333
  collection: docs
  operation: similarity_search
  query: "{{ get('q') }}"
  topK: 5
  embedModel: text-embedding-3-small
```

## Supported Providers

| `provider` | Connection via `url` |
|---|---|
| `qdrant` (default) | `http://localhost:6333` |
| `chroma` | `http://localhost:8000` (default if `url` is empty) |
| `pinecone` | `https://<index-host>.svc.<env>.pinecone.io` |
| `azureaisearch` | `https://<service>.search.windows.net`, or `AZURE_AI_SEARCH_ENDPOINT` env var |
| `opensearch` / `elasticsearch` | `http://localhost:9200` |
| `weaviate` | `http://localhost:8080` |
| `pgvector` / `postgres` / `postgresql` / `alloydb` / `cloudsql` | PostgreSQL DSN, e.g. `postgres://user:pass@localhost/db` |
| `mariadb` / `dolt` / `mysql` | MySQL DSN, e.g. `user:pass@tcp(localhost:3306)/dbname` |
| `mongodb` / `mongo` | MongoDB URI, e.g. `mongodb://localhost:27017` |
| `redis` | Redis URI, e.g. `redis://localhost:6379` (default if `url` is empty). `collection` is the Redis index name |
| `bedrock` | AWS Bedrock Knowledge Base -- `collection` is the knowledge base ID; no `url` or `embedModel` needed, embedding happens server-side. Uses the standard AWS SDK credential chain (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`) |

## Configuration Options

| Option | Operation | Description |
|---|---|---|
| `provider` | all | Vector store backend (see table above) |
| `url` | all | Endpoint or DSN for the store |
| `collection` | all | Collection / index / table name (required) |
| `apiKey` | all | Auth token. For `mongodb`/`mongo`, used as the database name instead (default `"kdeps"`). For `opensearch`/`elasticsearch`, format `"user:pass"` |
| `operation` | all | `add_documents` or `similarity_search` (required) |
| `documents` | add_documents | List of `{ content, metadata }` documents to upsert |
| `query` | similarity_search | Natural language search query |
| `topK` | similarity_search | Number of results to return (default: 5) |
| `embedModel` | all | Embedding model used to vectorize documents/queries, e.g. `text-embedding-3-small` (required) |
| `embedBackend` | all | Embedding provider: `openai`, `ollama`, `google` |
| `embedBaseURL` | all | Custom base URL for an OpenAI-compatible embedding backend |

## Output

**`add_documents`:**
```json
{ "success": true, "upserted": 1 }
```

**`similarity_search`:**
```json
{
  "success": true,
  "results": [
    { "content": "kdeps is a YAML-based AI agent framework.", "metadata": { "source": "readme" }, "score": 0.91 }
  ]
}
```

## Feeding from a Loader

`loader:`'s output documents map directly onto `vectorStore:`'s `documents:` field -- see the [Loader Resource](loader#rag-pipeline-example) for the full load-then-index pipeline.

## See also

- [Loader Resource](loader) -- load files/URLs into documents before indexing
- [Embedding Resource](embedding) -- local SQLite-backed alternative, no external service required
- [Resources Overview](overview) -- all resource types
