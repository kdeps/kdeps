# Loader resource

The `loader:` resource reads a file, URL, or directory into structured `Document` objects - plain text plus metadata - and optionally splits them into chunks. It is the ingestion step of a RAG pipeline: load, then feed the output into `vectorStore:` or `embedding:` for indexing.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In agent mode, the same loader is available as the `load_document` built-in tool.

## Basic usage

```yaml
# resources/load.yaml
actionId: load
name: Load Report
loader:
  type: pdf
  source: /data/quarterly-report.pdf
```

```yaml
# resources/loadAndChunk.yaml
actionId: loadAndChunk
name: Load and Chunk for RAG
loader:
  type: text
  source: /data/handbook.txt
  chunkSize: 500
  chunkOverlap: 50
  chunkSplitter: recursive
```

## Configuration options

| Option | Description |
|---|---|
| `type` | Loader to use: `text` (default), `pdf`, `html`, `csv`, `directory` |
| `source` | File path, URL (for `html`), or directory path (required) |
| `columns` | CSV only - optional column filter (empty = all columns) |
| `password` | PDF only - decryption password for encrypted PDFs |
| `chunkSize` | When set (> 0), splits each loaded document into chunks of this size |
| `chunkOverlap` | Characters of overlap between consecutive chunks |
| `chunkSplitter` | Splitting strategy: `recursive` (default), `token`, or `markdown` |

## Loader types

| `type` | Reads |
|---|---|
| `text` | Plain text file |
| `pdf` | PDF file, page text extracted |
| `html` | A URL - fetches and extracts readable text |
| `csv` | CSV file, one Document per row (or per `columns` filter) |
| `directory` | Every file in a directory, one Document per file |

## Output

```json
{
  "documents": [
    { "content": "...", "metadata": { "source": "/data/handbook.txt", "chunk": 0 } }
  ],
  "count": 3
}
```

`documents` is directly usable as the `documents:` input to a `vectorStore:` `add_documents` operation - each `content`/`metadata` pair maps onto a [`VectorStoreDocument`](vectorstore).

## RAG pipeline example

```yaml
# resources/load.yaml
actionId: load
loader:
  type: pdf
  source: "{{ get('filePath') }}"
  chunkSize: 500
  chunkOverlap: 50
```

```yaml
# resources/index.yaml
actionId: index
requires: [load]
vectorStore:
  operation: add_documents
  provider: qdrant
  url: http://localhost:6333
  collection: docs
  documents: "{{ get('load').documents }}"
  embedModel: text-embedding-3-small
```

## See also

- [Vector Store Resource](vectorstore) - index and search the loaded documents
- [Embedding Resource](embedding) - lightweight keyword-based alternative for local, API-key-free RAG
- [Resources Overview](overview) - all resource types
