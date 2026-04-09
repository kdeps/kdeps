# search-local-files

Local filesystem search using the **built-in native searchLocal executor** -- walks directories with glob filtering, pure Go, no external dependencies.

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

1. **search** -- `run.searchLocal` walks the given path, filters by glob, and returns files containing the keyword
2. **response** -- returns matching file paths, match count, and the original query

## Config fields

| Field | Description | Default |
|-------|-------------|---------|
| `path` | Directory to search | `/data` |
| `query` | Keyword to search for | (required) |
| `glob` | File pattern filter | all files |
| `limit` | Max results (0 = unlimited) | 20 |

## Structure

```
search-local-files/
├── workflow.yaml
└── resources/
    ├── search.yaml    # run.searchLocal - native filepath.WalkDir
    └── response.yaml  # API response
```
