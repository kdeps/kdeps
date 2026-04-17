# embedding-rag

Simple RAG (Retrieval-Augmented Generation) pipeline using the **built-in native embedding executor** -- keyword-indexed SQLite store, pure Go, zero external dependencies.

## Usage

```bash
kdeps run examples/embedding-rag/workflow.yaml --dev
```

Index a document:
```bash
curl -X POST http://localhost:16403/index \
  -H "Content-Type: application/json" \
  -d '{"text": "Go is a statically typed, compiled programming language designed at Google.", "collection": "docs"}'
```

Search indexed documents:
```bash
curl -X POST http://localhost:16403/search \
  -H "Content-Type: application/json" \
  -d '{"query": "compiled language", "collection": "docs"}'
```

## How it works

1. **index** -- `run.embedding` with `operation: upsert` stores the text in a SQLite keyword index
2. **search** -- `run.embedding` with `operation: search` retrieves the top-5 matching documents
3. **response** -- returns the results and match count

## Operations

| `operation` | Description |
|-------------|-------------|
| `index` | Add document to index |
| `upsert` | Add or update document |
| `search` | Keyword search, returns ranked results |
| `delete` | Remove document by text |

## Structure

```
embedding-rag/
├── workflow.yaml
└── resources/
    ├── index.yaml    # run.embedding operation: upsert
    ├── search.yaml   # run.embedding operation: search
    └── response.yaml # API response
```
