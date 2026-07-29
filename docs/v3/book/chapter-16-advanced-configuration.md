# Chapter 16: Advanced Configuration

This chapter covers the server-level settings that most developers reach for once they move from "it works" to "it works under load with real security requirements": rate limiting, trusted proxies, TLS, authentication, output caps, persistent memory, and the request object.

## Persistent Memory Configuration

Persistent memory stores key-value facts across requests, sessions, and restarts. In agent mode, it is enabled by default — no YAML configuration needed. The store lives at `~/.kdeps/memory/<encoded-cwd>/memory.jsonl` with per-project isolation.

For workflow mode, configure it under `settings.memory`:

```yaml
settings:
  memory:
    path: "/data/memory.jsonl"            # file path for persistent storage
    maxEntries: 10000                     # max entries before compaction triggers
    compactionThreshold: 5000              # compact when entries exceed this count
    compactionTarget: 2000                 # target entry count after compaction
```

| Field | Default | Description |
|---|---|---|
| `path` | `~/.kdeps/memory/<cwd>/memory.jsonl` | File path for the persistent memory store |
| `maxEntries` | `10000` | Maximum entries before compaction triggers |
| `compactionThreshold` | `5000` | Entry count that triggers compaction |
| `compactionTarget` | `2000` | Target entry count after compaction completes |

If `path` is not set in workflow mode, memory uses the agent-mode default path. For production, always set an explicit file path.

### Auto-Extraction (Agent Mode)

Memory is fully automatic in agent mode — no explicit `memory_save` calls required:

1. **Explicit markers** — `[MEMORY: key] value` on its own line in any LLM response
2. **Action sentences** — "Added X", "Fixed Y" captured as `last_action`
3. **File references** — edited/modified file paths captured as `last_files`
4. **Tool results** — every write/exec/search tool call saved as a `tool:<name>:<key>` entry (capped at 20 per tool type)

Entries are auto-classified into 14 types (`prompt`, `purpose`, `progress`, `tool_result`, `result`, `status`, etc.) and auto-linked into a dependency graph showing `prompt -> progress -> tool_result -> result -> status`.

### Session Persistence

The agent's LLM config (model, backend, base URL) is saved to `session:config` on startup and after every `/model` switch. On the next run, it is restored automatically. Working directory saved on start and resume.

See Chapter 15 for the full persistent memory reference including `set(..., 'memory')`, `memory_search`, `memory_delete`, `memory_list`, and the memory graph.

## Rate Limiting

Protect your agent from runaway clients and accidental abuse:

```yaml
settings:
  apiServer:
    rateLimit:
      requestsPerMinute: 60      # max requests per minute per client IP
      burst: 10                  # burst allowance above the per-minute rate
```

The rate limiter uses a token bucket algorithm. `requestsPerMinute: 60` is one request per second on average. `burst: 10` allows a client to send up to 10 requests in quick succession before being throttled.

When a client exceeds the rate limit, kdeps returns:

```json
HTTP/1.1 429 Too Many Requests
Retry-After: 5

{"success": false, "error": {"code": 429, "message": "rate limit exceeded"&#125;&#125;
```

### Per-Route Rate Limiting

```yaml
routes:
  - path: /api/v1/chat
    methods: [POST]
    rateLimit:
      requestsPerMinute: 10    # expensive LLM endpoint: stricter limit
  - path: /api/v1/status
    methods: [GET]
    rateLimit:
      requestsPerMinute: 300   # status endpoint: relaxed limit
```

Route-level limits override the global limit for that specific route.

## Trusted Proxies

When kdeps runs behind a reverse proxy (Nginx, Traefik, AWS ALB), the client IP that kdeps sees is the proxy's IP, not the actual client's. This breaks IP-based rate limiting and logging.

```yaml
settings:
  apiServer:
    trustedProxies:
      - "10.0.0.0/8"          # RFC 1918 private ranges
      - "172.16.0.0/12"
      - "192.168.0.0/16"
      - "127.0.0.1"           # loopback
```

When a request comes from a trusted proxy, kdeps reads the real client IP from the `X-Forwarded-For` or `X-Real-IP` header. Rate limiting, logging, and `request.IP` in expressions all use the real client IP. Without `trustedProxies`, kdeps uses `RemoteAddr` only — forwarded headers from untrusted peers are ignored.

When both `apiServer` and `webServer` are configured, entries from both `trustedProxies` blocks are merged for rate limiting and request IP context.

W> Only add IPs/ranges to `trustedProxies` that you control. Trusting an external IP means that IP can forge client addresses.

In Kubernetes, add your cluster's pod CIDR and your ingress controller's external IP:

```yaml
trustedProxies:
  - "10.0.0.0/8"              # pod CIDR
  - "172.20.0.0/16"           # service CIDR
```

## TLS

Enable HTTPS by setting `certFile` and `keyFile` directly under `settings:` — not under `settings.apiServer:`:

```yaml
# workflow.yaml
settings:
  certFile: "/etc/certs/server.crt"
  keyFile:  "/etc/certs/server.key"
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

The server listens on HTTPS only when both `certFile` and `keyFile` are set and the files exist. Omit them entirely for HTTP (e.g., when TLS is handled by a reverse proxy or ingress controller).

For development with a self-signed certificate:

```bash
# Generate a self-signed cert (development only)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost"
```

```yaml
settings:
  certFile: "./cert.pem"
  keyFile:  "./key.pem"
```

In production, prefer TLS termination at the proxy layer (Nginx, Traefik, AWS ALB) and run kdeps on HTTP internally. This avoids certificate rotation complexity inside the container.

## Authentication

The API server auth token lives in `~/.kdeps/config.yaml`, not in `workflow.yaml`:

```yaml
# ~/.kdeps/config.yaml
api_auth_token: "${API_TOKEN}"
```

Or via environment variable:

```bash
export KDEPS_API_AUTH_TOKEN="your-secret-token"
```

When `apiServer` is configured, the token is required — kdeps refuses to start without it. All workflow routes must include `Authorization: Bearer <token>` or `X-Api-Key: <token>`. Requests without a valid token receive 401. The `/health` endpoint is always exempt. `/_kdeps/*` management routes are exempt from API auth — they authenticate separately with `KDEPS_MANAGEMENT_TOKEN` (Chapter 21).

## Security Headers

Every `apiServer` and `webServer` response includes defensive HTTP headers:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains` (HTTPS only)

No configuration required — these apply automatically.

## Body Size and Concurrency Limits

### maxBodyBytes

Cap the size of incoming request bodies. Requests exceeding this limit receive `413 Payload Too Large` before any resource runs:

```yaml
settings:
  apiServer:
    maxBodyBytes: 1048576    # 1 MiB incoming body limit
```

This does not apply to `multipart/form-data` file uploads — uploaded files are streamed separately.

### maxConcurrent

Cap the number of simultaneous in-flight requests. When the limit is reached, new requests receive `503 Service Unavailable` immediately rather than queuing:

```yaml
settings:
  apiServer:
    maxConcurrent: 50    # max 50 requests processed simultaneously
```

Set this to prevent memory exhaustion when many LLM requests arrive at once. Each in-flight request holds its full DAG state in memory. A reasonable starting point is `maxConcurrent = (available_RAM_GB * 200)` for lightweight LLM workflows.

## Request Body Size Preflight

The agent loop checks the estimated request body size before every LLM API call. Providers with strict payload caps (e.g. DashScope at 6 MB) silently reject oversized requests -- the preflight catches this before the call is made, returning an actionable error message rather than a cryptic 400.

**This check runs automatically** in both the streaming tool-rounds loop and the engine-based `Loop.Run()` path. No code changes needed -- configure your backend and the preflight runs.

### Per-Provider Limits

| Backend | Limit |
|---------|-------|
| DashScope (kimi) | 6 MB |
| xAI (Grok) | 50 MB |
| OpenAI | 100 MB |
| All others | No known limit |

### Error Message

If the estimated size exceeds the provider's limit, the call is blocked before sending:

```
request body size ~5.5 MB exceeds dashscope limit of 6.0 MB
(reduce context window, trim conversation history, or switch models)
```

### Startup Warnings

At startup, kdeps warns about backends with limits under 10 MB so users of constrained providers know what to expect:

```
WARN  backend "dashscope" has a 6.0 MB request body limit --
      reduce MaxHistoryTokens if you hit payload errors
```

## Resource Output Caps

### HTTP Response Body Cap

Prevent oversized HTTP responses from being stored in the data store:

```yaml
settings:
  apiServer:
    maxResponseBytes: 10485760    # 10 MB max HTTP response body returned to callers
```

### Per-Resource Executor Output Caps

Each executor type has an environment variable that limits how many bytes it returns to the workflow engine. Set these in `agentSettings.envVars`:

```yaml
settings:
  agentSettings:
    envVars:
      KDEPS_EXEC_MAX_OUTPUT_BYTES: "524288"     # exec: stdout cap (512 KiB)
      KDEPS_HTTP_MAX_RESPONSE_BYTES: "1048576"  # httpClient: response body cap (1 MiB)
      KDEPS_CHAT_MAX_OUTPUT_BYTES: "1048576"    # chat: LLM response cap (1 MiB)
      KDEPS_PYTHON_MAX_OUTPUT_BYTES: "524288"   # python: stdout cap (512 KiB)
```

Output exceeding the cap is truncated before being stored. This prevents a single runaway resource from exhausting memory in a multi-resource pipeline.

For fine-grained control per resource, use `after:` with a slice expression:

```yaml
after:
  - set('truncated', get('llm')[0:5000])    # cap at 5000 chars
```

## The Request Object

The `request` object gives expressions access to the full HTTP request — method, path, headers, query params, body, files, and client metadata:

### Properties

| Property | Type | Description |
|---|---|---|
| `request.method` | string | HTTP method: `"POST"`, `"GET"`, etc. |
| `request.path` | string | Request URL path: `"/api/v1/chat"` |
| `request.IP` | string | Client IP (real IP when behind trusted proxy) |
| `request.ID` | string | Unique request ID (same as `info('ID')`) |
| `request.headers` | object | All request headers as a map |
| `request.query` | object | URL query parameters as a map |
| `request.body` | object | Parsed request body as a map |

```yaml
after:
  - set('auth', request.headers["Authorization"])
  - set('page', request.query["page"])
  - set('q', request.body["q"])
  - set('ip', request.IP)
```

### File Methods

When the request contains uploaded files (`multipart/form-data`), use these methods to access them:

**Per-file access** (by form field name):

```yaml
request.file('document')      # content of the uploaded file named "document"
request.filepath('document')  # path of the uploaded file
request.filetype('document')  # MIME type of the uploaded file
```

**Multi-file access:**

```yaml
request.files()               # array of all uploaded file paths
request.filetypes()           # array of MIME types for all uploaded files
request.filecount()           # total number of uploaded files (integer)
request.filesByType('image/*') # array of paths matching a MIME type pattern
```

**Convenience methods:**

```yaml
request.header('Authorization')   # single header value (case-insensitive)
request.params('page')            # single query parameter value
request.data()                    # entire request body as object (same as request.body)
```

Example — processing an uploaded document:

```yaml
before:
  - set('content', request.file('document'))
  - set('filename', request.filepath('document'))
  - set('mime', request.filetype('document'))

validations:
  check:
    - get('content') != ''
    - get('mime') in ['text/plain', 'application/pdf', 'text/html']
  error:
    code: 400
    message: "document field required; must be text, PDF, or HTML"
```

Example — handling multiple image uploads:

```yaml
before:
  - set('images', request.filesByType('image/*'))

validations:
  check:
    - request.filecount() > 0
    - request.filecount() <= 10
  error:
    code: 400
    message: "1–10 images required"

# resources/analyze-images.yaml
items: "&#123;&#123; get('images') &#125;&#125;"
chat:
  model: llama3.2-vision
  prompt: "Describe this image."
  files:
    - "&#123;&#123; get('current') &#125;&#125;"
```

### Practical Uses

**Log request metadata:**

```yaml
after:
  - set('log_entry', json({
      "request_id": request.ID,
      "ip": request.IP,
      "path": request.path,
      "method": request.method,
      "user_agent": request.headers["User-Agent"]
    }))
```

**Conditional processing based on client IP:**

```yaml
validations:
  check:
    - request.IP != '192.168.1.100'
  error:
    code: 403
    message: "access denied"
```

**Read a specific query parameter:**

```yaml
before:
  - set('page', int(request.query["page"]) or 1)
  - set('limit', min(int(request.query["limit"]) or 20, 100))
```

## Health Endpoints

Add a health check endpoint that deployment systems (Kubernetes, load balancers) can use:

```yaml
# workflow.yaml
settings:
  apiServer:
    routes:
      - path: /health
        methods: [GET]
      - path: /api/v1/chat
        methods: [POST]
```

```yaml
# resources/health.yaml
actionId: health
validations:
  routes: [/health]
  methods: [GET]
exec:
  command: "echo '{\"status\":\"ok\"}'"

# resources/health-respond.yaml
actionId: healthRespond
requires: [health]
validations:
  routes: [/health]
apiResponse:
  success: true
  statusCode: 200
  response:
    status: ok
    version: "1.0.0"
    timestamp: info('timestamp')
```

For Kubernetes you do not need to write probes by hand: the API server already serves a built-in `GET /health` endpoint (exempt from auth), and `kdeps export k8s` generates readiness and liveness probes against it automatically (Chapter 18). The custom health route above is for when you want richer status output — version, dependency checks, model availability — than the built-in endpoint provides.

## Per-Agent Config Profiles

The global `~/.kdeps/config.yaml` applies to all workflows on a machine. For cases where different workflows need different LLM backends or defaults — for example, one agent uses Ollama locally and another uses OpenAI — you can create per-agent profiles under the `agents:` key, keyed by the workflow's `metadata.name`:

```yaml
# ~/.kdeps/config.yaml

# Global defaults (apply to all workflows without a matching profile)
llm:
  backend: ollama
  ollama_host: http://localhost:11434

# Per-agent overrides (keyed by metadata.name in workflow.yaml)
agents:
  gpt-agent:
    llm:
      backend: openai
      openai_api_key: "${OPENAI_API_KEY}"
    defaults:
      timezone: America/New_York

  claude-agent:
    llm:
      backend: anthropic
      anthropic_api_key: "${ANTHROPIC_API_KEY}"
```

When `kdeps run` starts a workflow named `gpt-agent`, it merges the `agents.gpt-agent` profile over the global config. Workflows without a matching profile use the global config unchanged.

In an agency, each agent resolves its own profile independently. On startup, kdeps warns (non-fatally) about profiles that don't match any installed workflow name.

This is the right pattern when:
- You want different agents on the same machine to use different LLM providers
- You need different API keys per workflow without embedding them in `workflow.yaml`
- You are developing multiple agents locally and want each to pick up its own backend config

## Provider-Specific Chat Options

Some LLM providers expose capabilities that don't map to the standard sampling parameters. These fields go in the `chat:` resource, alongside `model:` and `prompt:`.

### Google AI and Vertex AI

**Vertex AI** -- To route requests through Google Cloud's Vertex AI endpoint instead of the standard `generativelanguage.googleapis.com` API, set the GCP project and region in the chat resource:

```yaml
chat:
  model: gemini-1.5-pro
  googleCloudProject: my-gcp-project   # GCP project ID
  googleCloudLocation: us-central1     # Vertex AI region (e.g. us-central1, europe-west4)
  prompt: "&#123;&#123; get('q') &#125;&#125;"
```

Vertex AI uses Application Default Credentials. Run `gcloud auth application-default login` on the host, or set `GOOGLE_APPLICATION_CREDENTIALS` to a service account key file.

**Safety thresholds** -- Tune how aggressively the Gemini safety filters block content:

```yaml
chat:
  model: gemini-1.5-pro
  googleHarmThreshold: 1   # 1=block-none, 2=block-few, 3=block-some, 4=block-most
```

**Cached content** -- Reference a pre-created `CachedContent` resource to avoid re-sending large context on every call:

```yaml
chat:
  model: gemini-1.5-pro
  googleCachedContent: "cachedContents/my-doc-cache-v1"
```

### Anthropic Extended Output and Beta Features

**Extended output** -- Increases the maximum output token limit to 128K for long-form generation:

```yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicExtendedOutput: true
  maxTokens: 32000
```

**Beta headers** -- Pass Anthropic beta feature headers directly when you need features not yet covered by a dedicated field:

```yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicBetaHeaders:
    - "interleaved-thinking-2025-05-14"
```

**Prompt caching** -- Reduce cost and latency for large, reused context:

```yaml
chat:
  model: claude-sonnet-4-20250514
  promptCaching: true
```

For full details on Anthropic caching, see Chapter 6.

### Ollama Native Options

When using the Ollama backend, these fields bypass the OpenAI-compatibility layer and talk directly to Ollama's native API:

```yaml
chat:
  model: qwen3:14b
  ollamaThink: true         # enable extended thinking for supported models
  ollamaKeepAlive: 5m       # evict the model from VRAM after this idle duration; "0" = unload immediately
  ollamaPullModel: true     # pull the model if not already present
  ollamaPullTimeout: 15m    # max wait for a pull before failing
```

These fields are silently ignored when the backend is not Ollama.

## Production Security Checklist

Before exposing a kdeps agent to production traffic:

- [ ] `hostIp: "0.0.0.0"` only if you need external access; use `"127.0.0.1"` + reverse proxy otherwise
- [ ] `api_auth_token` set in `~/.kdeps/config.yaml` or `KDEPS_API_AUTH_TOKEN` env var, rotated from a secret manager
- [ ] `KDEPS_MANAGEMENT_TOKEN` set in production if the management API is reachable; use a different value than the API token
- [ ] `rateLimit:` set to prevent abuse; stricter on expensive LLM endpoints
- [ ] `trustedProxies:` configured if running behind a reverse proxy
- [ ] `cors.allowOrigins:` lists specific origins; not `["*"]` for authenticated APIs
- [ ] `tls:` configured, or TLS handled by the proxy layer
- [ ] `maxBodyBytes:` set to prevent oversized incoming requests
- [ ] No credentials in `workflow.yaml`; all sensitive values via `${ENV_VAR}` substitution
- [ ] `validations:` on every resource that accepts user input
- [ ] SQL queries use parameterized form, never string interpolation of user input
- [ ] `exec:` commands do not interpolate unvalidated user input into shell strings

X> ## Exercise
X>
X> Harden the chatbot from Chapter 2 against production conditions using the settings from this chapter.
X>
X> Apply all of the following and test each with curl:
X>
X> 1. **Rate limiting** — set `requestsPerMinute: 5` and `burst: 2`. Send 8 rapid requests and confirm the 6th receives `429 Too Many Requests`.
X> 2. **Authentication** — run `export KDEPS_API_AUTH_TOKEN=testtoken` (or set `api_auth_token: "${API_TOKEN}"` in `~/.kdeps/config.yaml` with `API_TOKEN=testtoken` in your shell). Confirm that a request without the header gets `401`, and one with `Authorization: Bearer testtoken` succeeds.
X> 3. **Max body size** — set `maxBodyBytes: 512`. Send a request body larger than 512 bytes and confirm `413 Payload Too Large`.
X> 4. **Per-agent profile** — add a `~/.kdeps/config.yaml` with an `agents:` entry for your chatbot's `metadata.name`. Set a different `defaults.timezone` in the profile and confirm the startup log shows the profile was loaded.
X>
X> After all four are working, go through the Production Security Checklist at the end of the chapter and verify your workflow satisfies every item.
X>
X> **Stretch goal:** Generate a self-signed certificate with `openssl` and configure TLS. Test that `curl https://localhost:16395/api/v1/chat -k -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"` works and `http://` is refused.
