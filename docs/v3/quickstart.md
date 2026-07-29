# Quickstart

Code path: **scaffold → validate → run → call**.

## Workflow API

```bash
kdeps new my-agent
cd my-agent
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml --dev
```

In another terminal:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q":"hello"}'
```

### What `kdeps new` wrote

```text
my-agent/
├── workflow.yaml
└── resources/
    └── …
```

Edit `workflow.yaml` (`metadata.targetActionId`, `settings.apiServer.routes`) and resources under `resources/`. Each resource is one step; chain with `requires:` and `get('id')`.

### Minimal hand-written shape

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

### Useful run flags

```bash
kdeps run workflow.yaml --port 16395
kdeps run workflow.yaml --events          # NDJSON on stderr
kdeps run workflow.yaml --interactive     # workflow + REPL
kdeps run workflow.yaml --file ./doc.txt  # file input source
```

## Agent REPL (optional)

```bash
kdeps                 # local model REPL
kdeps ./my-agent/     # workflows become tools
```

## Next

- [CLI](/cli) — command + flag map  
- [Workflow mode](/workflow) — graph, layout, ship  
- [Resources](/resources) · [Config](/config)
