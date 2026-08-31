# Run extra resources inline

*Applies to workflow mode.*

## Overview

In this tutorial you attach `exec:`, `python:`, and `sql:` actions directly to
one resource's `before:` and `after:` blocks, instead of creating a separate
file for each. The inline actions run as part of the main resource.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic Python and SQL

By the end you will be able to:

- Run a full action inside `before:` (setup)
- Run full actions inside `after:` (post-processing)
- Decide when an inline action is clearer than a separate resource

## Background

`before:` and `after:` usually hold bare expressions. They can also hold whole
resource actions - `chat:`, `httpClient:`, `sql:`, `python:`, `exec:` - as list
items. Use inline actions for one-off setup or teardown that only this resource
needs; use a separate resource when other resources also depend on the result.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the project

```bash
mkdir inline-demo
cd inline-demo
mkdir resources

sqlite3 results.db "CREATE TABLE runs (data TEXT, at TEXT);"
```

## Step 2: define the route and connection

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: inline-demo
  version: "1.0.0"
  targetActionId: main

settings:
  apiServer:
    portNum: 16396
    routes:
      - path: /api/v1/process
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
  sqlConnections:
    results:
      connection: "sqlite:///./results.db"
```

## Step 3: the main resource with inline actions

Create `resources/main.yaml`:

<div v-pre>

```yaml
# resources/main.yaml
actionId: main
name: Process with inline resources
validations:
  methods: [POST]
  routes: [/api/v1/process]
  required:
    - data
  rules:
    - field: data
      type: string
      minLength: 1
      message: "data is required"

before:
  # setup: a shell command that only this resource needs
  - exec:
      command: "echo 'preparing'"
      timeout: 5s

# main action
chat:
  model: llama3.2:1b
  role: user
  prompt: "Rewrite this more formally: {{ get('data') }}"
  timeout: 30s

after:
  # persist the input
  - sql:
      connectionName: results
      query: "INSERT INTO runs (data, at) VALUES ($1, datetime('now'))"
      params:
        - "{{ get('data') }}"
  # post-process
  - python:
      script: |
        import json
        print(json.dumps({"post": "done"}))

onError:
  action: continue
  fallback:
    error: "processing failed"

apiResponse:
  success: true
  response:
    formal: "{{ get('main').message.content }}"
```

</div>

## Step 4: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl -X POST http://localhost:16396/api/v1/process \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": "hey can you send me that file"}'
```

The `before:` exec runs, then the chat, then the `after:` SQL insert and Python
script - all within the `main` resource.

## Summary

You attached full actions to one resource:

- `exec:` in `before:` for setup
- `sql:` and `python:` in `after:` for persistence and post-processing
- Kept them inline because nothing else depends on their output

## Next steps

- [Inline resources](/concepts/inline-resources) - all supported types, ordering
- [Expression blocks](/reference/expr-blocks) - `before:` / `after:` in detail
- [Error handling (onError)](/concepts/error-handling) - fallback for the whole resource
- [SQL resource](/resources/sql) - standalone SQL resources
