# Workflow Mode

Workflow mode runs a deterministic DAG pipeline: a request arrives, resources execute in dependency order, and the result is returned. Every run follows the same path for the same input.

Run with:

```bash
kdeps run workflow.yaml
```

## How it works

```d2
direction: down

A: "incoming request\nPOST /api/v1/chat" {shape: oval}
B: "resolve dep graph\nwalks backward from targetActionId"
C: "resource: validate\nfails fast if input invalid"
D: "resource: llm\nreads get('q'); calls the model"
E: "resource: resp\nreads get('llm'); builds the response"
F: HTTP response {shape: oval}

A -> B -> C
C -> D: "output stored as get('validate')"
D -> E: "output stored as get('llm')"
E -> F
```

`requires:` is like an import -- the resource won't run until its dependencies have output. Resources with no shared dependency path run concurrently.

## When to use workflow mode

- You need a deterministic, auditable pipeline.
- You are building a REST API, bot, or file-processing service.
- You want full control over which resources run and in what order.
- You need validation, early-exit, and explicit error handling.

## Comparison with agent mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps [path]`) |
|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Resources | Declared order | Run as part of a whole-workflow tool |
| Session | Single execution | Interactive REPL |

## Minimal example

`workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: chat-api
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

`resources/llm.yaml`:

<div v-pre>

```yaml
# resources/llm.yaml
actionId: llm
validations:
  check:
    - get('q') != ''
  error:
    code: 400
    message: "'q' is required"
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ get('q') }}"
  timeout: 60s
```

</div>

`resources/response.yaml`:

```yaml
# resources/response.yaml
actionId: response
requires: [llm]
apiResponse:
  success: true
  response:
    answer: get('llm')
```

Run:

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml

curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is entropy?"}'
```

`/health` is exempt. `/_kdeps/*` management routes use `KDEPS_MANAGEMENT_TOKEN` instead. See [Security Reference](/reference/security).

## Input sources

Workflow mode supports three input sources configured in `settings`:

```yaml
# API (default) - starts an HTTP server
settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]

# Bot - connects to a chat platform; blocks until SIGINT
# Credentials go in ~/.kdeps/config.yaml bot_connections, not here
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling   # polling = persistent; stateless = one message then exit
      discord: {}              # presence enables the platform

# File - reads one file from disk or stdin, runs once, exits
settings:
  input:
    sources: [file]
    file:
      path: /data/input.txt
```

See [Input Sources](../concepts/input-sources) for full configuration.

## See Also

- [Workflow Configuration](../configuration/workflow) - Full `workflow.yaml` reference
- [Resources Overview](../resources/overview) - Resource types and fields
- [Agent Mode](agent-loop-mode) - Autonomous LLM loop
