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
actionId: llm
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
actionId: response
requires: [llm]
apiResponse:
  success: true
  response:
    answer: get('llm')
```

## Run it

```bash
kdeps run workflow.yaml
```

Test the API:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
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

1. A POST request arrives at `/api/v1/chat` with `{"q": "..."}`.
2. kdeps validates that `q` is not empty.
3. The `llm` resource sends the prompt to `llama3.2:1b`.
4. The `response` resource depends on `llm` and formats the output.
5. The API returns the result.

The two resources form a simple DAG: `llm` -> `response`. This is workflow mode.

## Try agent mode

Run the same workflow as an interactive LLM agent that can call your resources as tools:

```bash
kdeps serve workflow.yaml
```

The agent REPL starts. Type a prompt and the LLM calls your resources as needed.

## Next steps

- [Modes](/modes/workflow-mode) - Understand workflow and agent modes
- [Workflow Configuration](../configuration/workflow) - Full `workflow.yaml` reference
- [Resources Overview](../resources/overview) - All resource types
- [CLI Reference](../reference/cli-reference) - All commands and flags
