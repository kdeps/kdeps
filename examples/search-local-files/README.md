# search-local-files

Local filesystem search through the `search_local` component, which calls the kdeps API's `search_local` tool. The tool walks directories in pure Go with glob filtering - no external dependencies.

## Usage

```bash
kdeps run examples/search-local-files/workflow.yaml --dev
```

Search files by keyword:
```bash
curl -X POST http://localhost:16404/search \
  -H "Content-Type: application/json" \
  -d '{"query": "error handling", "path": "/var/log", "glob": "*.log"}'
```

Search Go source files:
```bash
curl -X POST http://localhost:16404/search \
  -H "Content-Type: application/json" \
  -d '{"query": "interface", "path": "/app/src", "glob": "*.go"}'
```

## How it works

1. **search** - the `search_local` component POSTs to the kdeps API's `search_local` tool, which walks the given path, filters by glob, and returns files containing the keyword
2. **response** - returns matching file paths, match count, and the original query

## Component inputs

| Input | Description | Default |
|-------|-------------|---------|
| `path` | Directory to search | `/data` |
| `query` | Keyword to search for | (required) |
| `glob` | File pattern filter | all files |
| `limit` | Max results | 20 |

## Structure

```
search-local-files/
├── workflow.yaml
└── resources/
    ├── search.yaml    # component: search_local -> kdeps API search_local tool
    └── response.yaml  # API response
```
