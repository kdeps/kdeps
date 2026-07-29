# Workflow mode

Deterministic pipelines. Each resource declares dependencies with `requires:`. kdeps builds a graph, runs steps in order, and returns a response.

```bash
kdeps run workflow.yaml
kdeps run workflow.yaml --dev   # reload on file change
```

## Layout

```text
my-agent/
├── workflow.yaml
└── resources/
    ├── llm.yaml
    └── response.yaml
```

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

Optional under `settings`: `webServer`, `agentSettings`, `sqlConnections`, `session`, TLS (`certFile` / `keyFile` or `letsEncrypt`).

**Credentials stay out of the repo.** Put tokens and connection secrets in `~/.kdeps/config.yaml` or env vars.

## A resource file

<div v-pre>

```yaml
actionId: llm              # unique id; used by requires: and get()
name: chat
requires: []               # ids that must finish first
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

Each resource has **one primary action** (`chat`, `httpClient`, `sql`, `python`, `exec`, …). See [Resources](/resources).

Read another step's output with `get('actionId')`. The chat reply text is usually `get('llm').message.content`.

## Request flow

```text
POST /api/v1/chat
  -> auth / rate limit / body checks
  -> walk graph from targetActionId
  -> run each resource
  -> apiResponse builds the HTTP body
```

## Input sources

Workflows can listen on:

- **API** — `settings.apiServer` (REST)
- **Web** — `settings.webServer` (static or proxy)
- **Bot / file** — platform or file-driven input (see examples in the repo)

## Validate before run

```bash
kdeps validate workflow.yaml
kdeps doctor
```

Next: [Resources](/resources) · [Config](/config) · [Deploy](/deploy).
