# Process a batch of items in one request

*Applies to workflow mode.*

## Overview

In this tutorial you build an HTTP API that takes a list of items in one
request, fetches data for each item, transforms each result, and returns an
aggregated summary. It uses `items:` iteration - a resource runs once per
element in a list.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- How to send a POST request with `curl`

By the end you will be able to:

- Iterate a resource over a list with `items:`
- Read the current element with `item` (or `get('current')`)
- Iterate a second resource over the first resource's results
- Aggregate iteration output with `len()`

## Background

Without `items:`, a resource runs once. With `items:` set to a list, the
resource runs once per element, and each run produces its own output. The
collected outputs become the resource's result - a list the next resource can
iterate over in turn.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.
- Network access (the example calls the public GitHub API).

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

settings:
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
items:
  - "{{ input.items }}"           # the "items" array from the request body
httpClient:
  method: GET
  url: "https://api.github.com/repos/{{ item.org }}/{{ item.repo }}"
  headers:
    Accept: application/vnd.github.v3+json
  timeout: 10s
```

</div>

Inside an iterating resource, `item` is the current element, so `item.org` and
`item.repo` are its fields. The resource makes one HTTP request per item; the
responses are collected into a list.

## Step 4: transform each result

Create `resources/02-transform.yaml`:

<div v-pre>

```yaml
# resources/02-transform.yaml
actionId: transform
name: Extract star counts
requires: [fetch]
items:
  - "{{ get('fetch') }}"          # iterate over fetch's collected results
exec:
  command: echo
  args:
    - "{{ default(safe(item, 'data.name'), 'N/A') }} has {{ default(safe(item, 'data.stargazers_count'), 0) }} stars"
```

</div>

`safe(item, 'data.stargazers_count')` reads a nested field without erroring if
it is missing; `default(..., 0)` supplies a fallback.

## Step 5: aggregate and respond

Create `resources/03-response.yaml`:

<div v-pre>

```yaml
# resources/03-response.yaml
actionId: response
name: API response
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

## Summary

You built an API that:

- Iterates `fetch` over the request's `items` array with `items:`
- Iterates `transform` over `fetch`'s collected results
- Reads nested fields safely with `safe()` and `default()`
- Aggregates the outputs with `len()` in the response

## Next steps

- [Items iteration](/concepts/items) - `item.prev()`, `item.next()`, skipping
- [While-loop iteration](/concepts/loop) - unbounded iteration when the count is not known
- [HTTP client](/resources/http-client) - retries, auth, named connections
- [Expression helpers](/concepts/expression-helpers) - `safe()`, `default()`, `json()`
