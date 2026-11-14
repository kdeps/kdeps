# Process a batch of items in one request

Build an HTTP API that takes a list of items in one request. The API gets data
for each item, transforms each result, and returns an aggregated summary. It
uses `items:` iteration: a resource runs one time for each element in a list.

Without `items:`, a resource runs one time. With `items:` set to a list, the
resource runs one time for each element, and each run makes its own output.
The collected outputs are the result of the resource. This result is a list
that the next resource can iterate over.

**Needs:** Network access (the example calls the public GitHub API).

## Step 1: create the project

```bash
mkdir batch-repos
cd batch-repos
```

## Step 2: define the API route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: batch-repos
  version: "1.0.0"
  targetActionId: response

deploy:
  apiServer:
    portNum: 16395
    routes:
      - path: /process
        methods: [POST]
```

The workflow serves `POST /process`. The request body carries the list of
items.

## Step 3: fetch data for every item

Create `resources/01-fetch.yaml`:

<div v-pre>

```yaml
# resources/01-fetch.yaml
actionId: fetch
name: Fetch repo data
flow:
  items:
    - "{{ input.items }}"           # the "items" array from the request body
component:
  name: http_request
  with:
    method: GET
    url: "https://api.github.com/repos/{{ item.org }}/{{ item.repo }}"
    headers:
      Accept: application/vnd.github.v3+json
    timeout: 10s
```

</div>

Inside an iterating resource, `item` is the current element, so `item.org` and
`item.repo` are its fields. The resource makes one HTTP request for each item.
kdeps collects the responses into a list.

## Step 4: transform each result

Create `resources/02-transform.yaml`:

<div v-pre>

```yaml
# resources/02-transform.yaml
actionId: transform
name: Extract star counts
flow:
  requires: [fetch]
  items:
    - "{{ get('fetch') }}"          # iterate over fetch's collected results
component:
  name: exec
  with:
    command: echo
    args:
      - "{{ default(safe(item, 'data.name'), 'N/A') }} has {{ default(safe(item, 'data.stargazers_count'), 0) }} stars"
```

</div>

`safe(item, 'data.stargazers_count')` reads a nested field without erroring if
it is missing. `default(..., 0)` supplies a fallback.

## Step 5: aggregate and respond

Create `resources/03-response.yaml`:

<div v-pre>

```yaml
# resources/03-response.yaml
actionId: response
name: API response
flow:
  requires: [transform]
apiResponse:
  success: true
  response:
    summary:
      totalItems: "{{ len(get('transform')) }}"
      message: "Processed {{ len(get('transform')) }} repositories"
    results: "{{ get('transform') }}"
```

</div>

`get('transform')` is the full list of per-item outputs. `len()` counts them.

## Step 6: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

In another terminal:

```bash
curl -X POST http://localhost:16395/process \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"org": "kubernetes", "repo": "kubernetes"},
      {"org": "golang", "repo": "go"},
      {"org": "rust-lang", "repo": "rust"}
    ]
  }'
```

Response:

```json
{
  "success": true,
  "data": {
    "summary": { "totalItems": "3", "message": "Processed 3 repositories" },
    "results": [
      "kubernetes has 109000 stars",
      "go has 123000 stars",
      "rust has 97000 stars"
    ]
  }
}
```

## Next steps

- [Items iteration](/workflow/items) - `item.prev()`, `item.next()`, skipping
- [While-loop iteration](/workflow/loop) - iteration with no fixed limit, when you do not know the count
- [HTTP client](/workflow/resources/http-client) - retries, auth, named connections
- [Expression helpers](/workflow/expression-helpers) - `safe()`, `default()`, `json()`
