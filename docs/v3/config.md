# Config

Two layers: **workflow files** (behavior) and **machine config** (secrets / defaults).

## workflow.yaml settings

| Block | Purpose |
|-------|---------|
| `input` | `sources: [api\|bot\|file]` — [Inputs](/inputs) |
| `apiServer` | Host, port, routes, rate limits, CORS, trusted proxies |
| `webServer` | Static + optional frontend proxy — [Web server](/webserver) |
| `agentSettings` | Python version, packages, env for packaged runs |
| `sqlConnections` | Named DB DSNs |
| `session` | Cross-request store |
| `certFile` / `keyFile` / `letsEncrypt` | TLS — [TLS](/tls) |

### Session

```yaml
settings:
  session:
    type: sqlite                 # or memory
    path: "/data/sessions.db"
    ttl: "30m"
    cleanupInterval: "5m"
```

```yaml
before:
  - set('history', get('history') or [], 'session')
  - set('turn', int(get('turn') or 0) + 1, 'session')
```

Client sends / receives `X-Session-ID`.

### CORS (sketch)

```yaml
settings:
  apiServer:
    cors:
      allowOrigins: ["https://app.example.com"]
      allowMethods: [GET, POST]
      allowHeaders: [Authorization, Content-Type]
```

### Routes

API routes on `apiServer.routes`. Per-resource gates: `validations.methods` / `routes` / `public: true` on a route for open access.

## ~/.kdeps/config.yaml

Do not commit.

```yaml
api_auth_token: "…"

llm:
  backend: file                  # file | gguf | ollama | openai | …
  # base_url: http://host:8000/v1
  # openai_api_key: "…"

bot_connections: { }             # platform tokens — [Inputs](/inputs)
# smtp_connections / imap_connections / http_connections / search_connections

agents:
  my-agent:                      # matches metadata.name
    llm:
      backend: openai
```

## Environment

| Variable | Role |
|----------|------|
| `KDEPS_API_AUTH_TOKEN` | API bearer |
| `KDEPS_DEFAULT_BACKEND` | LLM backend |
| `KDEPS_LLM_BASE_URL` | OpenAI-compat `/v1` |
| `KDEPS_PERMISSION_MODE` | Agent tool permissions |
| `KDEPS_LEAN_MODE` / `KDEPS_AGENT_PRESET` | Agent tool surface |
| Provider keys | `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, … |

[Security](/security) · [LLM](/llm) · [Coding agent](/agent).
