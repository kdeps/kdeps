# Workflow Mode

Workflow mode runs a deterministic DAG pipeline: a request arrives, resources execute in dependency order, and the result is returned. Every run follows the same path for the same input.

Run with:

```bash
kdeps run workflow.yaml
```

## How it works

```
incoming request  (POST /api/v1/chat)
        |
        v
+-------------------------+
|  resolve dep graph      |  <- walks backward from targetActionId
|  targetActionId: resp   |
+-------------------------+
        |
        v
+-------------------------+
|  resource: validate     |  <- runs first; fails fast if input invalid
+-------------------------+
        |  output stored as get('validate')
        v
+-------------------------+
|  resource: llm          |  <- reads get('q'); calls the model
+-------------------------+
        |  output stored as get('llm')
        v
+-------------------------+
|  resource: resp         |  <- reads get('llm'); builds the response
+-------------------------+
        |
        v
     HTTP response
```

`requires:` is like an import -- the resource won't run until its dependencies have output. Resources with no shared dependency path run concurrently.

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
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling   # polling = persistent; stateless = one message then exit
      discord:
        botToken: "${DISCORD_BOT_TOKEN}"

# File - reads one file from disk or stdin, runs once, exits
settings:
  input:
    sources: [file]
    file:
      path: /data/input.txt
```

See [Input Sources](../concepts/input-sources) for full configuration.

## See also

- [Workflow Configuration](../configuration/workflow) - Full `workflow.yaml` reference
- [Resources Overview](../resources/overview) - Resource types and fields
- [Agent Mode](agent-mode) - Autonomous LLM loop
