# Quickstart

Two paths. Pick one.

## Agent in 30 seconds

```bash
kdeps
```

Talk to a local model (llamafile by default). No workflow files required.

Give the agent tools by pointing at a project:

```bash
kdeps ./my-agent/
```

Every workflow under that path becomes a callable tool. Type `/help` inside the REPL for slash commands.

## Workflow API in a few minutes

### 1. Create a project

```bash
kdeps new my-agent
cd my-agent
```

### 2. Define the workflow

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

### 3. Add an LLM step

`resources/llm.yaml`:

<div v-pre>

```yaml
actionId: llm
name: chat
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

### 4. Return the response

`resources/response.yaml`:

```yaml
actionId: response
name: api
requires: [llm]
apiResponse:
  success: true
  data:
    answer: get('llm').message.content
```

### 5. Run

API mode needs a token (never commit it):

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml
```

Call it:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q":"hello"}'
```

Next: [Two modes](/modes) · [Workflow mode](/workflow) · [Agent mode](/agent).
