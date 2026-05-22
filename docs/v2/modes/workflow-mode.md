# Workflow Mode

Workflow mode is the default execution mode in kdeps. You define a set of resources with explicit dependencies, and kdeps resolves the execution order as a DAG. The workflow runs in response to an incoming request, processes it through the resource graph, and returns a structured response.

Run with:

```bash
kdeps run workflow.yaml
```

## How it works

1. A request arrives at the configured input source (API, bot, or file).
2. kdeps resolves the dependency graph from `targetActionId` backward.
3. Resources execute in dependency order. Resources with no common dependency path execute concurrently.
4. Each resource reads prior outputs via `get('actionId')` and produces its own output.
5. The resource named in `metadata.targetActionId` defines the response.

## When to use workflow mode

- You need a deterministic, auditable pipeline.
- You are building a REST API, bot, or file-processing service.
- You want full control over which resources run and in what order.
- You need validation, early-exit, and explicit error handling.

## Comparison with agent mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps serve`) |
|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Resources | Declared order | Called as tools on demand |
| Session | Single execution | Interactive REPL |

## Minimal example

`workflow.yaml`:

```yaml
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
actionId: response
requires: [llm]
apiResponse:
  success: true
  response:
    answer: get('llm')
```

Run:

```bash
kdeps run workflow.yaml

curl -X POST http://localhost:16395/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"q": "What is entropy?"}'
```

## Input sources

Workflow mode supports three input sources. Configure them in `settings`:

- `api` - HTTP REST server (default)
- `bot` - Discord, Slack, Telegram, WhatsApp
- `file` - Read from stdin, env var, or file path

See [Input Sources](../concepts/input-sources) for full configuration.

## See also

- [Workflow Configuration](../configuration/workflow) - Full `workflow.yaml` reference
- [Resources Overview](../resources/overview) - Resource types and fields
- [Agent Mode](agent-mode) - Autonomous LLM loop
