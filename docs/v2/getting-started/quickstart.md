# Quickstart

Build a working LLM API in under five minutes.

## Prerequisites

Install kdeps:

```bash
# macOS / Linux
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

Install and start Ollama (for local LLM):

```bash
# macOS / Linux
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2:1b
```

## Create a project

```bash
kdeps new my-agent
cd my-agent
```

Or create the structure manually:

```bash
mkdir -p my-agent/resources && cd my-agent
```

## Define your workflow

`workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent
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

## Add an LLM resource

`resources/llm.yaml`:

<div v-pre>

```yaml
# resources/llm.yaml
actionId: llm
name: LLM Chat
validations:
  methods: [POST]
  routes: [/api/v1/chat]
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

## Add a response resource

`resources/response.yaml`:

```yaml
# resources/response.yaml
actionId: response
name: API Response
requires: [llm]
apiResponse:
  success: true
  response:
    # chat output is the raw response object; the reply text is at .message.content
    answer: get('llm').message.content
```

## Run it

When `apiServer` is configured, kdeps requires an API auth token before it starts. Set one for local development (never in `workflow.yaml`):

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml
```

You can also set `api_auth_token` in `~/.kdeps/config.yaml`. See [Security Reference](/reference/security).

Test the API:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is entropy?"}'
```

Expected response:

```json
{
  "success": true,
  "response": {
    "answer": "Entropy is a measure of disorder..."
  }
}
```

## How it works

```d2
direction: down

A: "POST /api/v1/chat\n{\"q\": \"What is entropy?\"}" {shape: oval}
B: "resource: llm\nvalidates get('q') != ''; calls llama3.2:1b"
C: "resource: response\nrequires: [llm]; reads get('llm').message.content"
D: "{\"success\": true, \"response\": {\"answer\": \"...\"}}" {shape: oval}

A -> B
B -> C: "output stored as get('llm')"
C -> D
```

`requires: [llm]` means `response` will not run until `llm` has finished. This two-resource DAG is the simplest workflow mode pipeline.

## Try agent mode

Run the workflow as a tool in an interactive LLM loop. The tool name is `my-agent` (from `metadata.name`). The LLM calls `my-agent`, the full pipeline executes, and the result comes back.

```bash
# Serve the current project directory -- registers one tool named "my-agent"
kdeps serve .

# Point at a folder to expose every workflow inside as separate tools
kdeps serve ./agents/
```

The agent REPL starts. Type a prompt and the LLM calls your workflow tools as needed.

## See Also

- [Modes](/modes/workflow-mode) - Understand workflow and agent modes
- [Workflow Configuration](../configuration/workflow) - Full `workflow.yaml` reference
- [Resources Overview](../resources/overview) - All resource types
- [CLI Reference](/reference/cli/) - All commands and flags
