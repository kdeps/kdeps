# Vector store

The `vector_store` component adds documents to, and runs similarity search against, an external vector database - Qdrant, Chroma, Pinecone, pgvector, and more - for production-scale RAG. The kdeps API tool does the embedding and store work; the component only passes the inputs through.

Works in workflow and agent mode.

```text
component: vector_store
  -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/vector_store/call
    -> embeds documents/queries with embedModel on embedBackend
      -> add_documents: upsert into the provider's collection
      -> similarity_search: query the collection, return the top matches
```

Unlike the [embedding component](/workflow/resources/embedding) (a local SQLite keyword index built into kdeps), `vector_store` talks to a real vector database service.

## Basic usage

```yaml
# resources/index.yaml
actionId: index
name: Index Documents
component:
  name: vector_store
  with:
    provider: qdrant              # store backend; qdrant is the default
    url: http://localhost:6333    # store endpoint or DSN
    collection: docs              # collection/index/table name in the store
    operation: add_documents      # upsert the documents below
    embedModel: text-embedding-3-small   # vectors are generated with this model
    embedBackend: openai          # which provider serves embedModel
    documents:
      - content: "kdeps is a YAML-based AI agent framework."
        metadata: { source: readme }
```

```yaml
# resources/search.yaml
actionId: search
name: Semantic Search
requires: [index]
component:
  name: vector_store
  with:
    provider: qdrant
    url: http://localhost:6333
    collection: docs
    operation: similarity_search  # query the collection
    query: "{{ get('q') }}"       # natural language query
    topK: 5                       # how many matches to return; default 5
    embedModel: text-embedding-3-small
    embedBackend: openai
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
    results: "{{ fromJSON(output('search')).results }}"
```

## Inputs

| Input | Description |
|---|---|
| `operation` | `add_documents` or `similarity_search` (required) |
| `collection` | Collection / index / table name (required) |
| `provider` | Backend (see below); `qdrant` when omitted |
| `url` | Endpoint or DSN for the store |
| `apiKey` | Auth token. For `mongodb`/`mongo`: the database name (default `"kdeps"`). For `opensearch`/`elasticsearch`: `"user:pass"` |
| `documents` | `add_documents`: list of `{ content, metadata }` documents to upsert |
| `query` | `similarity_search`: natural language search query |
| `topK` | `similarity_search`: number of results; default 5 |
| `embedModel` | Model used to vectorize documents and queries, e.g. `text-embedding-3-small` (required, unless `bedrock` or `system`) |
| `embedBackend` | Embedding provider: `openai`, `ollama`, `google`, or `system` (follow `KDEPS_DEFAULT_BACKEND` / the machine config) |
| `embedBaseURL` | Custom base URL for an OpenAI-compatible embedding backend |

## Supported providers

| `provider` | Connection via `url` |
|---|---|
| `qdrant` (default) | `http://localhost:6333` |
| `chroma` | `http://localhost:8000` (default if `url` is empty). `apiKey` is not sent, so authenticated Chroma servers are not supported |
| `pinecone` | `https://<index-host>.svc.<env>.pinecone.io` |
| `azureaisearch` | `https://<service>.search.windows.net`, or `AZURE_AI_SEARCH_ENDPOINT` env var |
| `opensearch` / `elasticsearch` | `http://localhost:9200` |
| `weaviate` | `http://localhost:8080` |
| `pgvector` / `postgres` / `postgresql` / `alloydb` / `cloudsql` | PostgreSQL DSN, e.g. `postgres://user:pass@localhost/db`. AlloyDB and Cloud SQL connect through the same PostgreSQL DSN |
| `mariadb` / `dolt` / `mysql` | MySQL DSN, e.g. `user:pass@tcp(localhost:3306)/dbname` |
| `mongodb` / `mongo` | MongoDB URI, e.g. `mongodb://localhost:27017`. Similarity search uses an Atlas Vector Search index named `vector_index` on the collection, so a self-hosted MongoDB without Atlas Vector Search cannot run `similarity_search` |
| `redis` | Redis URI (default `redis://localhost:6379`). `collection` is the Redis index name |
| `bedrock` | AWS Bedrock Knowledge Base - `collection` is the knowledge base ID; no `url` or `embedModel` needed, embedding happens server-side. Uses the standard AWS SDK credential chain (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`) |

Each store connects when the resource runs. An unreachable `url` fails the resource with a `vectorstore` error instead of failing later on the first call.

## Output

**`add_documents`:**
```json
{ "added": 1, "ids": ["d2f1c0de-..."] }
```

**`similarity_search`:**
```json
{
  "count": 1,
  "results": [
    { "content": "kdeps is a YAML-based AI agent framework.", "metadata": { "source": "readme" } }
  ],
  "json": "[{\"content\":\"kdeps is a YAML-based AI agent framework.\",\"metadata\":{\"source\":\"readme\"}}]"
}
```

## Feeding from the loader component

`load_document` returns documents with `content` and `metadata` fields that map directly onto the `documents` input:

```yaml
# resources/index.yaml
actionId: index
name: Index
requires: [load]
component:
  name: vector_store
  with:
    provider: qdrant
    url: http://localhost:6333
    collection: docs
    operation: add_documents
    embedModel: text-embedding-3-small
    embedBackend: openai
    documents: "{{ fromJSON(output('load')).documents }}"  # load_document's Document list
```

See the [Loader component](/workflow/resources/loader#rag-pipeline-example) for the full load-then-index pipeline.

## See also

- [Loader component](/workflow/resources/loader) - load files/URLs into documents before indexing
- [Embedding component](/workflow/resources/embedding) - local SQLite-backed alternative, no external service required
- [Resources overview](/workflow/resources) - all resource types
