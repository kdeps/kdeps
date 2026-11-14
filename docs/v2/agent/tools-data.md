# Data tools

`calculator`, the SQLite tools, and the local embedding store are always on. The SQLite tools use `db_path`. If `db_path` is not set, they use `KDEPS_SQL_DB_PATH`. Wolfram, remote RAG, and the rerankers register only when you set their key or URL. They are [declared HTTP tools](/agent/tools#declared-http-tools), and their tool asset fully defines them. Before `sql_query` text runs, kdeps removes its HTML escapes, as for [`bash_exec`](/agent/shell).

Permission modes stay on [Built-in tools](/agent/tools).

## Computation

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `calculator` | (none) | Evaluate an arithmetic expression: numbers, `+ - * / // % **`, parentheses and the `math` module's functions and constants (`sqrt(16)`, `log(8, 2)`, `pi`). A `python:` block in `tools/calculator.yaml`; exponents above 10000 are rejected |
| `wolfram_alpha` | `WOLFRAM_APP_ID` | Wolfram Alpha queries |

## Data and SQL

The three SQLite tools are `python:` blocks in their tool assets (Python's built-in `sqlite3`). Each tool opens the file read-only (`mode=ro`), so a query cannot write to it. First, the tool compares `db_path` with `KDEPS_WORKSPACE_ROOT`. Paths under `~/.kdeps` and `/tmp` are also permitted. `sql_ops` (named connections, any driver) stays in Go.

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `sql_list_tables` | `db_path` or `KDEPS_SQL_DB_PATH` | List the tables in a SQLite file |
| `sql_describe_table` | same | The table's `CREATE TABLE` statement |
| `sql_query` | same | Run one `SELECT` or `WITH` query; tab-separated rows, column names first |

## Embeddings and reranking

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `embedding_vectorize` | (none) | Convert text to embeddings and index it in the local embedding DB |
| `embedding_search` | (none) | Semantic search over the local embedding DB |
| `retrieve_context` | `KDEPS_RAG_BASE_URL` | Retrieve chunks from a remote RAG endpoint (only registered when the URL is set) |
| `cohere_rerank` | `COHERE_API_KEY` | Rerank results using Cohere |
| `voyageai_rerank` | `VOYAGEAI_API_KEY` | Rerank results using VoyageAI |
| `jina_rerank` | `JINA_API_KEY` | Rerank results using Jina |
