# kdeps workflow

A kdeps workflow is a deterministic YAML pipeline. A request arrives, resources run in
dependency order, and kdeps returns the result. The same input always has the
same execution path. For this reason, the workflow is auditable and testable, and it can
run safely without supervision. Serve it as an HTTP
API, a bot, a file processor, or an interactive prompt.

```bash
kdeps run ./my-agent/      # run once / serve on :16395
```

A `chat` with `loop:` in this file is an agent. To run several workflows as one system, see [Agencies](/agencies/). To ship the file, see [Deploy](/deploy/).

---

If the workflow has no `chat:` resource, kdeps does not download a model or start an LLM server. `/health` binds immediately. Models download on the first chat request.

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

A resource runs only after everything in its `requires:` has produced output. Resources with no shared dependency path run concurrently.

## One file

The same `workflow.yaml` runs as an HTTP API, a bot, a file processor, or an interactive prompt. A failed check returns 400 before any model call. A `chat` with `loop:` is one step. `maxRounds` stops that step.

## Minimal example

The two-resource chat API, the auth token, and the curl are on the [workflow quickstart](/workflow/quickstart). `/health` is exempt from that token. `/_kdeps/*` management routes use `KDEPS_MANAGEMENT_TOKEN` instead. See [Security reference](/reference/security).

## Input sources

A workflow has four input sources. Set them in `deploy`:

```yaml
# API (default) - starts an HTTP server
deploy:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]

# Bot - connects to a chat platform; blocks until SIGINT
# Credentials go in ~/.kdeps/config.yaml bot_connections, not here
deploy:
  input:
    sources: [bot]
    bot:
      executionType: polling   # polling = persistent; stateless = one message then exit
      discord: {}              # presence enables the platform

# File - reads one file from disk or stdin, runs once, exits
deploy:
  input:
    sources: [file]
    file:
      path: /data/input.txt

# Interactive - each line runs the workflow
deploy:
  input:
    sources: [interactive]
```

See [Input sources](/workflow/input-sources) for full configuration.

## Agent memory (`--memory`)

A workflow can use the agent memory notes. Pass `--memory` to enable:

```bash
kdeps run workflow.yaml --memory
kdeps exec my-agent --memory
```

When enabled, four expression functions become available in resource YAML:

| Function | Description |
|---|---|
| `memory_save(key, value)` | Save a key-value pair to persistent memory |
| `memory_search(query)` | Search memory entries by content (returns JSON array) |
| `memory_list()` | List all memory keys |
| `memory_delete(key)` | Delete a memory entry by key |

Memory persists across workflow runs as notes in the knowledge vault's `Memory/` folder (default `~/.kdeps/notes/Memory/`). Use it to carry state between invocations:

<div v-pre>

```yaml
# resources/llm.yaml
actionId: llm
before:
  - memory_save('last_query', get('q'))
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: |
      Previous context: {{ memory_search('user preference') }}
      Answer: {{ get('q') }}
```

</div>

```yaml
# resources/response.yaml
actionId: response
flow:
  requires: [llm]
before:
  - memory_save('last_response', get('llm').message.content)
apiResponse:
  success: true
  response:
    answer: get('llm').message.content
```

kdeps automatically links memory entries into a relationship graph. The graph shows the chain from the prompt to the tool calls and to the results.

## See also

- [Workflow configuration](/workflow/configuration) - Full `workflow.yaml` reference
- [Resources overview](/workflow/resources) - Resource types and fields
- [Agent loop](/workflow/agent-loop/) - a chat with `loop:`
