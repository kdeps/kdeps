# Build a document search API (RAG)

Build an HTTP API with two endpoints: one indexes a
document, the other searches the indexed documents and returns the closest
matches. It uses the built-in `embedding:` resource, which stores text in a
local SQLite index - no external vector database, no API key.

This is the retrieval half of RAG. The `embedding:` resource keeps a keyword
index in SQLite and ranks results by match - enough for a working search API
with zero setup.

## Step 1: create the project

```bash
mkdir doc-search
cd doc-search
mkdir resources
```

## Step 2: define two routes

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: doc-search
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    portNum: 16403
    routes:
      - path: /index
        methods: [POST]
      - path: /search
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
```

## Step 3: index a document

Create `resources/index.yaml`:

<div v-pre>

```yaml
# resources/index.yaml
actionId: indexDoc
name: Index document
validations:
  methods: [POST]
  routes: [/index]              # this resource only runs for POST /index
  check:
    - get('text') != ''        # reject an empty body
  error:
    code: 400
    message: "text is required"
embedding:
  operation: upsert            # add or update the document in the index
  text: "{{ get('text') }}"
  collection: "{{ get('collection', 'docs') }}"   # 'docs' if not supplied
```

</div>

`get('collection', 'docs')` reads the `collection` field from the request, or
falls back to `"docs"`.

## Step 4: search the index

Create `resources/search.yaml`:

<div v-pre>

```yaml
# resources/search.yaml
actionId: searchDocs
name: Search documents
validations:
  methods: [POST]
  routes: [/search]            # this resource only runs for POST /search
  check:
    - get('query') != ''
  error:
    code: 400
    message: "query is required"
embedding:
  operation: search            # return ranked matches
  text: "{{ get('query') }}"
  collection: "{{ get('collection', 'docs') }}"
  limit: 5
```

</div>

## Step 5: return the results

Create `resources/response.yaml`:

<div v-pre>

```yaml
# resources/response.yaml
actionId: response
name: API response
requires: [indexDoc, searchDocs]
apiResponse:
  success: true
  response:
    results: "{{ output('searchDocs').results }}"
    count: "{{ output('searchDocs').count }}"
```

</div>

Only one of `indexDoc` / `searchDocs` runs per request - the other is skipped
by its `routes` validation. `output('searchDocs')` is empty on an `/index`
call, which is fine.

## Step 6: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run . --dev
```

Index a few documents:

```bash
curl -X POST http://localhost:16403/index \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"text": "Go is a statically typed, compiled language designed at Google."}'

curl -X POST http://localhost:16403/index \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"text": "Rust is a systems language focused on memory safety without a garbage collector."}'
```

Search:

```bash
curl -X POST http://localhost:16403/search \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "compiled language"}'
```

Response:

```json
{
  "success": true,
  "data": {
    "count": 1,
    "results": [
      { "text": "Go is a statically typed, compiled language designed at Google.", "score": 0.82 }
    ]
  }
}
```

## Next steps

- [Embedding resource](/workflow/resources/embedding) - collections, delete, batch upsert
- [Vector store](/workflow/resources/vector-store) - persistent vector storage
- [searchLocal](/workflow/resources/search-local) - TF-IDF search over files on disk
- [Validation and control flow](/workflow/validation) - route and method scoping
