# RAG resources

Three resources that build a retrieval-augmented generation (RAG) pipeline.
Each has its own reference page.

Works in workflow and agent mode.

```text
loader (load + chunk)  ->  embedding component OR vectorStore (index)  ->  embedding component OR vectorStore (search)  ->  chat (answer with context)
```

| Resource | Role | Reference |
| :--- | :--- | :--- |
| `loader:` | Read files, URLs, or a directory into text chunks | [Loader](/workflow/resources/loader) |
| `component: embedding` | Local SQLite keyword store - index and search, no API key | [Embedding](/workflow/resources/embedding) |
| `vectorStore:` | External vector database - Qdrant, Chroma, pgvector, ... | [Vector store](/workflow/resources/vector-store) |

## embedding or vector store

| | `component: embedding` | `vectorStore:` |
| :--- | :--- | :--- |
| Storage | Local SQLite file | External vector DB service |
| Matching | Keyword (LIKE) | Vector similarity |
| Setup | None | Run and configure the DB |
| API key | Not needed | Needed for the embedding model |
| Best for | On-prem, small corpora | Production scale, semantic ranking |

## See also

- [Document search tutorial](/examples/rag-search) - `component: embedding` upsert and search
- [Scraper component](/workflow/resources/scraper) - fetch content to index
- [Search resources](/workflow/resources/search) - keyword file and web search
- [LLM resource](/workflow/resources/llm) - answer with the retrieved context
- [Resources overview](/workflow/resources) - all resource types
