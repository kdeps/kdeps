# embedding-rag

Simple RAG (Retrieval-Augmented Generation) pipeline that indexes documents and searches them through the built-in `embedding` component -- a keyword-indexed SQLite store, pure Go, zero external dependencies.

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

1. **index** -- calls the `embedding` component with `operation: upsert`; the component stores the text in a SQLite keyword index
2. **search** -- calls the `embedding` component with `operation: search`; the component retrieves the top-5 matching documents
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
    ├── index.yaml    # component: embedding, operation: upsert
    ├── search.yaml   # component: embedding, operation: search
    └── response.yaml # API response
```
