# Config

Two layers: **workflow files** (what runs) and **machine config** (secrets and defaults).

## workflow.yaml settings

| Block | Purpose |
|-------|---------|
| `apiServer` | Host, port, routes, rate limits, trusted proxies |
| `webServer` | Static files or reverse-proxy target |
| `agentSettings` | Runtime packages, env, Python deps for packaged runs |
| `sqlConnections` | Named DB DSNs (prefer secrets via env) |
| `session` | Cross-request key-value store |
| `certFile` / `keyFile` | Static TLS |
| `letsEncrypt` | ACME TLS for a public domain |

Example API surface:

```yaml
settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
      - path: /api/v1/public
        methods: [GET]
        public: true   # no bearer token
```

## ~/.kdeps/config.yaml

Machine-local. Do not commit.

```yaml
api_auth_token: "your-secret"

llm:
  backend: openai          # or file, ollama, gguf, …
  base_url: http://host:8000/v1
  # openai_api_key: "..."

# Optional per-agent overrides (metadata.name)
agents:
  my-agent:
    llm:
      backend: openai
      openai_api_key: sk-...
```

Named connections for email, HTTP auth, and search providers also live here.

## Environment overrides

| Variable | Role |
|----------|------|
| `KDEPS_API_AUTH_TOKEN` | API bearer / X-Api-Key value |
| `KDEPS_DEFAULT_BACKEND` | LLM backend id |
| `KDEPS_LLM_BASE_URL` | OpenAI-compat base URL (…/v1) |
| `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, … | Provider keys |

## Auth rule of thumb

- Token for `apiServer` routes (unless `public: true`)
- Token in env or `~/.kdeps/config.yaml`, never in `workflow.yaml`
- Management endpoints under `/_kdeps/*` use a separate management token when enabled

See [Security](/security) and [TLS](/tls).
