# Chapter 4: Workflow Mode

Workflow mode is where kdeps earns the "production-ready" label. You run a workflow with `kdeps run`, the framework starts an HTTP server, and every incoming request is processed through a deterministic DAG. Inputs are validated before any resource executes. Resources run in dependency order. The same input produces the same processing path every time.

This chapter covers how workflow mode works, how to design effective DAG pipelines, how validation fits into the execution flow, and patterns for structuring real workflows.

## Starting a Workflow

```bash
$ kdeps run workflow.yaml
```

Or point at a directory containing `workflow.yaml`:

```bash
$ kdeps run ./my-agent/
```

kdeps reads `workflow.yaml`, discovers all YAML files in `resources/`, builds the DAG, and starts the HTTP server.

```
kdeps: loading workflow: my-agent v1.0.0
kdeps: discovered 5 resources: [fetch, parse, enrich, combine, respond]
kdeps: DAG validated — no cycles detected
kdeps: starting server on 127.0.0.1:16395
kdeps: ready
```

## Request Lifecycle

When a request arrives:

```
1. Request received: POST /api/v1/chat
2. Route matching: does this path+method match any configured route?
3. Resource filtering: which resources have matching validations.routes/methods?
4. Validation: check expressions evaluated; fail fast if any are false
5. DAG execution: resources run in dependency order
6. Terminal resource: targetActionId resource executes last
7. Response: apiResponse builds and returns HTTP response body
```

Steps 4 through 6 happen per-request. Steps 1 through 3 happen in the server layer before any resource logic runs.

If validation fails in step 4, the entire pipeline stops. No downstream resources execute. The error defined in `validations.error` is returned to the caller.

## Declaring Dependencies with `requires:`

`requires:` is how you express "this resource depends on that one." kdeps builds the execution order from these declarations alone.

```yaml
# resources/fetch-data.yaml
actionId: fetchData
httpClient:
  method: GET
  url: "https://api.example.com/data"
```

```yaml
# resources/analyze.yaml
actionId: analyze
requires: [fetchData]       # fetchData must complete before analyze runs
chat:
  model: llama3.2:1b
  prompt: "Analyze this data: &#123;&#123; get('fetchData') &#125;&#125;"
```

```yaml
# resources/store.yaml
actionId: store
requires: [analyze]         # analyze must complete before store runs
sql:
  connectionName: main
  query: "INSERT INTO analyses (result) VALUES ($1)"
  params:
    - get('analyze')
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [store]           # store must complete (write confirmed) before responding
apiResponse:
  success: true
  response:
    result: get('analyze')
```

The DAG:

```
fetchData → analyze → store → respond
```

Every step waits for its predecessors. If `fetchData` fails (network error, non-200 response), `analyze` never runs, `store` never runs, and the caller gets an error response immediately.

## Parallel Execution

When multiple resources declare the same upstream dependency, they run concurrently:

```yaml
# resources/analyze.yaml
actionId: analyze
requires: [fetchData]

# resources/summarize.yaml
actionId: summarize
requires: [fetchData]

# resources/respond.yaml
actionId: respond
requires: [analyze, summarize]
```

```
fetchData
 ├── analyze ─┐
 └── summarize┘
              ↓
           respond
```

`analyze` and `summarize` both start as soon as `fetchData` completes. `respond` waits for both. You did not write any concurrency management code. You declared the dependencies; kdeps handles the scheduling.

## Validation

Validations gate resource execution. A resource with `validations:` checks those conditions before executing. If any `check:` expression is false, the resource stops and the `error:` block is returned.

```yaml
# resources/create-user.yaml
actionId: createUser
validations:
  methods: [POST]
  routes: [/api/v1/users]
  check:
    - get('email') != ''
    - get('email') matches '^[^@]+@[^@]+\\.[^@]+$'
    - len(get('password')) >= 8
  error:
    code: 422
    message: "validation failed: email must be valid and password at least 8 chars"
```

**`validations.methods`** — list of HTTP methods this resource responds to. Requests with a different method skip this resource silently.

**`validations.routes`** — list of route patterns this resource responds to. Requests to other paths skip this resource silently.

**`validations.check`** — list of boolean expressions. All must be true for execution to continue. If any is false, execution stops and `validations.error` is returned.

**`validations.skip`** — list of boolean expressions. If any is true, the resource is silently skipped (not an error — just not executed). Useful for optional processing steps.

```yaml
validations:
  skip:
    - get('cached') != ''    # skip re-computation if we already have a cached result
```

## The before: and after: Blocks

Every resource can run expression statements before and after its action:

```yaml
# resources/process.yaml
actionId: process

before:
  - set('query', lower(trim(get('q'))))        # normalize input
  - set('limit', int(get('limit')) or 10)      # default limit to 10

chat:
  model: llama3.2:1b
  prompt: "Answer in &#123;&#123; get('limit') &#125;&#125; words: &#123;&#123; get('query') &#125;&#125;"

after:
  - set('word_count', len(split(get('process'), ' ')))
  - set('response_length', len(get('process')))
```

`before:` runs before the action. It is the right place to normalize and derive inputs. `after:` runs after the action produces output. It is the right place to compute derived values from the output.

Both blocks are lists of bare expressions executed sequentially. Unlike expression interpolation in strings (`&#123;&#123; &#125;&#125;`), these are not wrapped in double braces — they are evaluated directly.

## Designing Effective DAGs

A few practical patterns for DAG design:

**Pattern: validate early, fail fast.** Put all input validation on the first resource in each request path. If validation fails, no downstream resources run.

**Pattern: one resource, one concern.** Keep resources focused. A resource that fetches data should not also transform it. Separation makes resources independently testable and reusable.

**Pattern: name resources by what they produce.** `actionId: userRecord` or `actionId: enrichedData` communicates intent. Downstream resources use `get('userRecord')` — the name reads naturally.

**Pattern: use `before:` for derivation, `after:` for reduction.** Normalize and prepare inputs in `before:`. Compute summaries and downstream-useful values from outputs in `after:`.

**Pattern: handle the "cache check" with `validations.skip`.** If you have expensive resources (LLM calls, external API calls) that should be skipped when a cached result exists, add a `skip:` check at the top of that resource.

## A Real-World Example: Document Q&A Pipeline

Here is a workflow that takes a document URL and a question, fetches the document, and answers the question using its content:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: doc-qa
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/ask
        methods: [POST]
```

```yaml
# resources/validate.yaml
actionId: validate
validations:
  methods: [POST]
  routes: [/api/v1/ask]
  check:
    - get('url') != ''
    - get('question') != ''
  error:
    code: 400
    message: "both 'url' and 'question' are required"
exec:
  command: "echo 'validated'"    # no-op; just a validation gate
```

```yaml
# resources/fetch-doc.yaml
actionId: fetchDoc
requires: [validate]
scraper:
  url: "&#123;&#123; get('url') &#125;&#125;"
  timeout: 30
```

```yaml
# resources/answer.yaml
actionId: answer
requires: [fetchDoc]
chat:
  model: llama3.2:1b
  prompt: |
    Using only the following document content, answer the question.
    If the answer is not in the document, say so.

    Document:
    &#123;&#123; get('fetchDoc') &#125;&#125;

    Question: &#123;&#123; get('question') &#125;&#125;
  timeout: 120s
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [answer]
apiResponse:
  success: true
  response:
    question: get('question')
    source_url: get('url')
    answer: get('answer')
```

The DAG:

```
validate → fetchDoc → answer → respond
```

Test it (set `KDEPS_API_AUTH_TOKEN` first — see Chapter 2):

```bash
$ curl -X POST http://localhost:16395/api/v1/ask \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/article.html",
    "question": "What is the main conclusion?"
  }'
```

This pipeline is deterministic, testable, and deployable. Every step is explicit. Every dependency is declared. Validation fires before any external call is made.

## Multiple Routes on One Workflow

A workflow can serve multiple routes, each triggering different resource subgraphs:

```yaml
# workflow.yaml
settings:
  apiServer:
    routes:
      - path: /api/v1/ask
        methods: [POST]
      - path: /api/v1/summarize
        methods: [POST]
```

Resources gate themselves with `validations.routes`:

```yaml
# resources/answer.yaml
actionId: answer
validations:
  routes: [/api/v1/ask]
  # ...

# resources/summarize.yaml
actionId: summarize
validations:
  routes: [/api/v1/summarize]
  # ...
```

Resources that do not match the incoming route are silently skipped. The `targetActionId` resource must handle all routes or be conditional itself.

## What Workflow Mode Is Not

Workflow mode does not support polling loops, background jobs, or streaming responses (in the base configuration). It is a request-response model: one request in, one response out, DAG executes in between.

For background processing, wrap kdeps as a service and trigger it from a queue or scheduler. For streaming, Chapter 20 covers the WebServer mode. For long-running autonomous tasks, agent mode (next chapter) is the right tool.

The constraint is a feature. Deterministic request-response pipelines are easy to test, easy to monitor, and easy to reason about under load. Most AI workloads fit this model if you design them well.

X> ## Exercise
X>
X> Build a three-resource workflow that accepts a product name via `POST /api/v1/describe` and returns a short marketing description.
X>
X> Requirements:
X>
X> 1. A `validate` resource that checks the `name` field is present, non-empty, and under 100 characters. Return a `400` error with a clear message if validation fails.
X> 2. An `llm` resource that requires `validate` and uses a prompt like: `"Write a 2-sentence marketing description for: &#123;&#123; get('name') &#125;&#125;"`.
X> 3. A `respond` resource that requires `llm` and returns `{ "success": true, "description": "..." }`.
X>
X> Test both the happy path and the error path:
X> ```bash
X> # Happy path
X> curl -X POST localhost:16395/api/v1/describe \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"name":"noise-cancelling headphones"}'
X>
X> # Error path — empty name
X> curl -X POST localhost:16395/api/v1/describe \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"name":""}'
X> ```
X>
X> Confirm the error path returns HTTP 400 before `kdeps run --instrument` confirms the `llm` resource was never evaluated.
X>
X> **Stretch goal:** Add a fourth resource `cache` that uses `validations.skip` to skip the LLM call if a session key for the same product name already exists, returning the cached result instead.
