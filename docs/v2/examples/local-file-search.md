# Search local files

*Applies to workflow mode.*

## Overview

In this short tutorial you build an API that searches a directory on disk by
filename pattern and content keyword, using the built-in `searchLocal:`
resource - no index, no external service.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Glob patterns

By the end you will be able to:

- Search a directory with `searchLocal:`
- Combine a filename `glob` with a content `query`
- Read the result shape (`results`, `count`)

## Background

`searchLocal:` walks a directory and returns matching files. With both a
`query` and a `glob` set, a file must match both. For ranked results over a
large folder, add `index: true` (see the resource reference).

## Before you start

- kdeps installed (`kdeps --version`).
- A directory with some text files to search.

## Step 1: create the project

```bash
mkdir file-search
cd file-search
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: file-search
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    portNum: 16404
    routes:
      - path: /search
        methods: [POST]
```

## Step 3: the search resource

Create `resources/search.yaml`:

<div v-pre>

```yaml
# resources/search.yaml
actionId: search
name: Search
validations:
  methods: [POST]
  routes: [/search]
  check:
    - get('query') != ''
  error:
    code: 400
    message: "query is required"
searchLocal:
  path: "{{ get('path', '/data') }}"      # ?path=... or default
  query: "{{ get('query') }}"             # keyword in file contents
  glob: "{{ get('glob', '') }}"           # optional filename pattern
  limit: 20
```

</div>

## Step 4: return the results

Create `resources/response.yaml`:

<div v-pre>

```yaml
# resources/response.yaml
actionId: response
name: Response
requires: [search]
apiResponse:
  success: true
  response:
    results: "{{ output('search').results }}"
    count: "{{ output('search').count }}"
    query: "{{ get('query') }}"
```

</div>

## Step 5: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl -X POST http://localhost:16404/search \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path": "./docs", "query": "invoice", "glob": "*.md"}'
```

Response:

```json
{
  "success": true,
  "data": {
    "results": [{ "path": "./docs/2024-invoice.md", "name": "2024-invoice.md", "size": 812 }],
    "count": 1,
    "query": "invoice"
  }
}
```

## Summary

You built a search API that:

- Walks a directory with `searchLocal:`
- Requires both a `glob` and a `query` match when both are set
- Returns `results` and `count`

## Next steps

- [searchLocal resource](/resources/search/searchlocal) - `index: true`, fuzzy matching, graph boost
- [Document search tutorial](/examples/rag-search) - semantic search with `embedding:`
- [Code intelligence](/resources/code-intelligence/navigation) - symbol search over a source tree
- [Web scraper tutorial](/examples/web-scraper) - fetch remote pages instead
