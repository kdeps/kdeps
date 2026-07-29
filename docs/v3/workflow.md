# Workflow mode

Deterministic DAG. CLI entrypoint: **`kdeps run`**.

```bash
kdeps new my-agent
cd my-agent
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml --dev
```

## Layout

```text
my-agent/
├── workflow.yaml
└── resources/
    ├── llm.yaml
    └── response.yaml
```

`kdeps new <name>` scaffolds this (template default: `api-service`).

## CLI

| Step | Command |
|------|---------|
| Scaffold | `kdeps new <name> [-t template]` |
| Check | `kdeps validate .` |
| Run | `kdeps run workflow.yaml` |
| Reload | `kdeps run workflow.yaml --dev` |
| Port | `kdeps run workflow.yaml --port 16395` |
| File input | `kdeps run workflow.yaml --file ./in.txt` |
| Events | `kdeps run workflow.yaml --events` |
| + REPL | `kdeps run workflow.yaml --interactive` |
| Memory fns | `kdeps run workflow.yaml --memory` |
| Package | `kdeps bundle package .` |
| Image | `kdeps bundle build . --tag …` |

Full flag list: [CLI](/cli).

## workflow.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: chat-api           # alphanumeric + hyphens
  version: "1.0.0"
  targetActionId: response # terminal resource id

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Optional `settings`: `webServer`, `agentSettings`, `sqlConnections`, `session`, TLS (`certFile` / `keyFile` or `letsEncrypt`).

**Secrets stay off-repo** — `~/.kdeps/config.yaml` or env (`KDEPS_API_AUTH_TOKEN`, provider keys).

## Resource file

<div v-pre>

```yaml
actionId: llm
name: chat
requires: []
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
```

</div>

One primary action per resource (`chat`, `httpClient`, `sql`, `python`, `exec`, …). See [Resources](/resources).

Read prior output with `get('actionId')`. Chat text: `get('llm').message.content`.

## Runtime path

```text
kdeps run workflow.yaml
  -> load + resolve graph from targetActionId
  -> open input (api / web / bot / file)
  -> per request: auth gates -> run resources -> apiResponse
```

```text
POST /api/v1/chat + Bearer token
  -> rate limit / body / concurrency
  -> validate resource
  -> chat resource
  -> apiResponse
```

## Ship the same files

```bash
kdeps validate .
kdeps bundle package .
kdeps bundle build . --tag myregistry/agent:latest
kdeps export k8s . -o deploy.yaml
```

See [Deploy](/deploy) · [Security](/security) · [CLI](/cli).
