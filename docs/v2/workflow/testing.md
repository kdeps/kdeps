# Evals: test a workflow

`kdeps test` runs the `evals:` cases declared in `workflow.yaml` against the workflow's own router, in-process, with no port bound.

```text
workflow.yaml (evals:) -> kdeps test -> in-process router -> engine -> assertions + judges
```

The request path is the one a live client gets: the same middleware, bearer auth, route matching, executors, and `apiResponse` formatting. Only the network listener is not used. As a result, a case that passes here also passes against `kdeps run`.

## Declare the cases

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: echo
  version: "1.0.0"
  targetActionId: response

deploy:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/echo
        methods: [POST]
  runtime:
    timezone: "UTC"

resources:
  - actionId: response
    name: Response
    apiResponse:
      success: true
      response:
        ok: true

evals:
  - name: health
    request:
      path: /health            # method defaults to GET; path is required
    assert:
      status: 200
      body:
        jsonPath:
          - path: $.status
            equals: ok

  - name: echo
    timeout: 10s               # per-case hard stop; default 30s
    request:
      method: POST
      path: /api/v1/echo
      query:
        dryRun: "1"            # appended as ?dryRun=1
      headers:
        X-Trace: local         # sent as given
      body:
        q: hi                  # encoded as JSON
    assert:
      headers:
        Content-Type: application/json
      body:
        contains: '"success":true'
        jsonPath:
          - path: $.data.ok
            equals: true
```

Run it:

```bash
kdeps test .                   # or: kdeps test workflow.yaml
```

```text
Running 2 eval(s) for echo

  PASS  health (200, 0s)
  PASS  echo (200, 3ms)

2 passed, 0 failed
```

`kdeps test --fail-fast` stops at the first failing case and exits non-zero. A case that fails prints its status and the first 400 bytes of the response body.

## Assertions

Every field under `assert:` is optional. A case with no `assert:` only checks that the run completes.

| Field | Passes when |
|---|---|
| `status` | The response status equals the value |
| `headers` | Each named header contains the expected substring |
| `body.contains` | The raw body contains the string |
| `body.equals` | The raw body equals the string exactly |
| `body.jsonPath` | Each entry resolves and matches its check |

A `jsonPath` entry needs exactly one of:

| Check | Passes when |
|---|---|
| `equals` | The resolved value equals the value (numbers, booleans, and strings compare by value) |
| `contains` | The resolved value, as text, contains the string |
| `exists` | A `true`/`false` expectation that the path resolves |

The path subset is `$`, `.key`, and `[n]` - for example `$.data.items[0].name`. `kdeps validate` rejects wildcards, filters, and quoted keys.

## Score with judges

A judge scores the response body. Graders run in the process. `judgeModel` reviews a judge with `criteria`. `repeat` runs the case several times and `threshold` is the pass rate the runs need, for answers that vary from run to run.

```yaml
evals:
  - name: answers-politely
    request:
      method: POST
      path: /api/v1/ask
      body:
        q: "Where is my order?"
    judges:
      - name: mentions-order       # graders: no model call
        graders:
          - kind: contains         # exact | contains | not_contains | regex | json | expr
            pattern: order
      - name: polite               # criteria: reviewed by judgeModel
        criteria: "The reply is polite and does not blame the customer."
    judgeModel:
      name: llama3.2:1b            # required when any judge has criteria
      backend: ollama
    repeat: 5                      # run the case 5 times (default 1)
    threshold: 0.8                 # pass when 4 of 5 runs pass (default 1: every run)
```

```text
  PASS  answers-politely (200, 2.1s, 5/5 runs)
  FAIL  answers-politely: passed 3/5 runs, threshold 0.8: judge polite: blames the customer
```

The judges and graders are the ones a chat uses as runtime [guardrails](/workflow/resources/llm). A failed review fails the run.

## Auth

If a case sets no `Authorization` or `X-API-Key` header, kdeps sends it with `KDEPS_API_AUTH_TOKEN` as a bearer token. A case that sets its own credential keeps it. `kdeps test` applies the same security gate as `kdeps run`: a workflow with an authenticated `apiServer` refuses to run without a token.

## What runs before the cases

The same setup as `kdeps run`:

1. kdeps renders the `.j2` templates in the project directory.
2. kdeps validates the workflow against the schema and the test definitions.
3. kdeps prepares `runtime.env` and the Python environment.
4. If a chat resource needs a local LLM backend and none runs, kdeps starts one.

Each case then executes the full DAG through the engine, so a test exercises the real resources, not fixtures.

## Scope

`kdeps test` runs workflow `evals:`. Agencies and components do not define `evals:`; test the entry-point agent's workflow directly.

## See also

- [`kdeps test`](/reference/cli-dev#kdeps-test) - flags and exit behavior
- [Validation and control](/workflow/validation) - per-resource request gates
- [Expressions](/workflow/expressions) - expression syntax used inside resources
