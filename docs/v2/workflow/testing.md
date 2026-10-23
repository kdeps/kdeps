# Test a workflow

`kdeps test` runs the `tests:` cases declared in `workflow.yaml` against the workflow's own router, in-process, with no port bound.

```text
workflow.yaml (tests:) -> kdeps test -> in-process router -> engine -> assertions
```

The request path is the one a live client gets: the same middleware, bearer auth, route matching, executors, and `apiResponse` formatting. Only the network listener is skipped, so a case that passes here passes against `kdeps run`.

## Declare the cases

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: echo
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/echo
        methods: [POST]
  agentSettings:
    timezone: "UTC"

resources:
  - actionId: response
    name: Response
    apiResponse:
      success: true
      response:
        ok: true

tests:
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
Running 2 test(s) for echo

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

The path subset is `$`, `.key`, and `[n]` - for example `$.data.items[0].name`. Wildcards, filters, and quoted keys are rejected by `kdeps validate`.

## Auth

A case that sets no `Authorization` or `X-API-Key` header is sent with `KDEPS_API_AUTH_TOKEN` as a bearer token. A case that sets its own credential keeps it. `kdeps test` applies the same security gate as `kdeps run`: a workflow with an authenticated `apiServer` refuses to run without a token.

## What runs before the cases

The same setup as `kdeps run`:

1. `.j2` templates in the project directory are rendered.
2. The workflow is validated against the schema and the test definitions.
3. `agentSettings.env` and the Python environment are prepared.
4. A local LLM backend is started when a chat resource needs one and none is running.

Each case then executes the full DAG through the engine, so a test exercises the real resources, not fixtures.

## Scope

`kdeps test` runs workflow `tests:`. Agencies and components do not define `tests:`; test the entry-point agent's workflow directly.

## See also

- [`kdeps test`](/reference/cli-dev#kdeps-test) - flags and exit behavior
- [Validation and control](/workflow/validation) - per-resource request gates
- [Expressions](/workflow/expressions) - expression syntax used inside resources
