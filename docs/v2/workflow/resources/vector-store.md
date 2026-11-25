# Vector store

The `vector_store` component adds documents to, and runs similarity search against, an external vector database - Qdrant, Chroma, Pinecone, pgvector, and more - for production-scale RAG. The kdeps API tool does the embedding and store work. The component only passes the inputs through.

```text
component: vector_store
  -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/vector_store/call
    -> embeds documents/queries with embedModel on embedBackend
      -> add_documents: upsert into the provider's collection
      -> similarity_search: query the collection, return the top matches
```

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

## Read next

- [Vector store reference](/reference/resources/vector-store) - basic usage, inputs, supported providers, output, feeding from the loader component
- [Loader component](/workflow/resources/loader) - load files/URLs into documents before you index them
- [Embedding component](/workflow/resources/embedding) - local SQLite-backed alternative, no external service required
- [Resources overview](/workflow/resources) - all resource types
