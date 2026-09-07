# RAG resources

Three resources that build a retrieval-augmented generation (RAG) pipeline.
Each has its own reference page.

*Applies to both workflow mode and agent mode.*

```text
loader (load + chunk)  ->  embedding OR vectorStore (index)  ->  embedding OR vectorStore (search)  ->  chat (answer with context)
```

| Resource | Role | Reference |
| :--- | :--- | :--- |
| `loader:` | Read files, URLs, or a directory into text chunks | [Loader](/workflow/resources/loader) |
| `embedding:` | Local SQLite keyword store - index and search, no API key | [Embedding](/workflow/resources/embedding) |
| `vectorStore:` | External vector database - Qdrant, Chroma, pgvector, ... | [Vector store](/workflow/resources/vector-store) |

## embedding or vector store

| | `embedding:` | `vectorStore:` |
| :--- | :--- | :--- |
| Storage | Local SQLite file | External vector DB service |
| Matching | Keyword (LIKE) | Vector similarity |
| Setup | None | Run and configure the DB |
| API key | Not needed | Needed for the embedding model |
| Best for | On-prem, small corpora | Production scale, semantic ranking |

## See also

- [Document search tutorial](/examples/rag-search) - `embedding:` upsert and search
- [Scraper resource](/workflow/resources/scraper) - fetch content to index
- [Search resources](/workflow/resources/search) - keyword file and web search
- [LLM resource](/workflow/resources/llm) - answer with the retrieved context
- [Resources overview](/workflow/resources) - all resource types
