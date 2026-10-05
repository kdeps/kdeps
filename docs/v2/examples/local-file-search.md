# Search local files

Build an API that searches a directory on disk by
filename pattern and content keyword, using the built-in `searchLocal:`
resource - no index, no external service.

`searchLocal:` walks a directory and returns matching files. With both a
`query` and a `glob` set, a file must match both. For ranked results over a
large folder, add `index: true` (see the resource reference).

**Needs:** A directory with some text files to search.

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

## Next steps

- [searchLocal resource](/workflow/resources/search-local) - `index: true`, fuzzy matching, graph boost
- [Document search tutorial](/examples/rag-search) - semantic search with `embedding:`
- [Code intelligence](/workflow/resources/code-navigation) - symbol search over a source tree
- [Web scraper tutorial](/examples/web-scraper) - fetch remote pages instead
