# Data tools

`calculator` and the local embedding store are always on. SQL tools register when a database path is set. Wolfram, remote RAG, and the rerankers register only when their key or URL is set. `sql_query` text is HTML-unescaped before it runs, same as [`bash_exec`](/agent/shell).

Permission modes stay on [Built-in tools](/agent/tools).

## Computation

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `calculator` | (none) | Evaluate math expressions |
| `wolfram_alpha` | `WOLFRAM_APP_ID` | Wolfram Alpha queries |

## Data and SQL

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `sql_list_tables` | `KDEPS_SQL_DB_PATH` or connection config | List tables in a database |
| `sql_describe_table` | same | Describe a table's columns and types |
| `sql_query` | same | Execute a SELECT query |

## Embeddings and reranking

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `embedding_vectorize` | (none) | Convert text to embeddings and index it in the local embedding DB |
| `embedding_search` | (none) | Semantic search over the local embedding DB |
| `retrieve_context` | `KDEPS_RAG_BASE_URL` | Retrieve chunks from a remote RAG endpoint (only registered when the URL is set) |
| `cohere_rerank` | `COHERE_API_KEY` | Rerank results using Cohere |
| `voyageai_rerank` | `VOYAGEAI_API_KEY` | Rerank results using VoyageAI |
| `jina_rerank` | `JINA_API_KEY` | Rerank results using Jina |
