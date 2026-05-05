# Quickstart

## Prerequisites

```bash
# Install via script (Mac/Linux)
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## Create a project

```bash
kdeps new my-agent
cd my-agent
```

Or manually create the structure:

```bash
mkdir my-agent && cd my-agent && mkdir resources
```

## Define your workflow

`workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: responseResource
settings:
  apiServerMode: true
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: llmResource
run:
  validations:
    methods: [POST]
    routes: [/api/v1/chat]
    check:
      - get('q') != ''
    error:
      code: 400
      message: Query parameter 'q' is required
  chat:
    role: user
    prompt: "{{ get('q') }}"
    timeoutDuration: 60s
```

</div>

## Add a response resource

`resources/response.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: responseResource
  requires: [llmResource]
run:
  apiResponse:
    success: true
    response:
      data: get('llmResource')
```

## Run it

```bash
kdeps run workflow.yaml
```

Test the API:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"q": "What is AI?"}'
```

## How it works

1. Request arrives at `/api/v1/chat` with `{"q": "..."}`
2. Validation checks `q` is not empty
3. LLM resource sends the prompt to the model
4. Response resource formats the output
5. API responds with the result

## Next steps

- [CLI Reference](/reference/cli-reference)
- [Workflow Configuration](../configuration/workflow)
- [LLM Resource](../resources/llm)
