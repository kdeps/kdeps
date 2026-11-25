# Vector store

The `vector_store` component adds documents to, and runs similarity search against, an external vector database, such as Qdrant, Chroma, Pinecone or pgvector.

```text
documents -> embed -> vector database -> similarity search -> top matches
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
    documents:
      - content: "kdeps is a YAML-based AI agent framework."
        metadata: { source: readme }
```

## Read next

- [Vector store reference](/reference/resources/vector-store) - every key, option and detail
- [Loader component](/workflow/resources/loader) - load files/URLs into documents before you index them
- [Embedding component](/workflow/resources/embedding) - local SQLite-backed alternative, no external service required
