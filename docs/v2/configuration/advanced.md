# Advanced Configuration

This reference covers security, rate limiting, trusted proxies, resource output caps, and other server-level settings that live in `workflow.yaml` under `settings:`.

## Request Object

The `request` object provides access to HTTP request metadata in expressions.

### Available Properties

| Property | Type | Description |
|----------|------|-------------|
| `request.method` | string | HTTP method (GET, POST, etc.) |
| `request.path` | string | Request path |
| `request.IP` | string | Client IP address |
| `request.ID` | string | Unique request ID |
| `sessionId` | string | Session ID (if sessions enabled) |

### Usage Examples

```yaml
# resources/log-request.yaml
actionId: logRequest
after:
  # Access request metadata
  - set('method', request.method)
  - set('path', request.path)
  - set('clientIp', request.IP)
  - set('requestId', request.ID)
  - set('session', info('sessionId'))

  # Build log entry
  - set('logEntry', json({
      "timestamp": info('ID'),
      "method": get('method'),
      "path": get('path'),
      "ip": get('clientIp'),
      "requestId": get('requestId')
    }))
```

### Request-Based Routing

```yaml
# resources/example.yaml
after:
  # Different behavior based on request method
  - set('isPost', request.method == 'POST')
  - set('isGet', request.method == 'GET')
validations:
  skip:
    - "!get('isPost')"
```

### Logging and Auditing

<div v-pre>

```yaml
# resources/example.yaml
sql:
  connectionName: logs
  queries:
    - query: |
        INSERT INTO audit_log (request_id, method, path, ip, session_id, timestamp)
        VALUES (?, ?, ?, ?, ?, NOW())
      params:
        - "{{ request.ID }}"
        - "{{ request.method }}"
        - "{{ request.path }}"
        - "{{ request.IP }}"
        - "{{ info('sessionId') }}"
```

</div>

## Agent Settings

The `agentSettings` section configures the runtime environment.

### Complete Reference

```yaml
# workflow.yaml
settings:
  agentSettings:
    # Timezone
    timezone: "America/New_York"

    # Python Configuration
    pythonVersion: "3.11"
    pythonPackages:
      - numpy==1.26.0
      - pandas>=2.0.0
      - requests
    requirementsFile: "requirements.txt"
    pyprojectFile: "pyproject.toml"
    lockFile: "uv.lock"

    # System Packages
    packages:
      - ffmpeg
      - imagemagick
    osPackages:
      - libpq-dev
      - libxml2-dev
    repositories:
      - ppa:deadsnakes/ppa

    # Docker Configuration
    baseOS: "ubuntu"  # alpine or ubuntu

    # Environment
    args:
      BUILD_TYPE: production
    env:
      API_KEY: "${API_KEY}"
      DEBUG: "false"
```

### Field Descriptions

#### Python Settings

| Field | Description |
|-------|-------------|
| `pythonVersion` | Python version (e.g., "3.11", "3.12") |
| `pythonPackages` | List of pip packages to install |
| `requirementsFile` | Path to requirements.txt |
| `pyprojectFile` | Path to pyproject.toml (for uv) |
| `lockFile` | Path to uv.lock file |

#### System Packages

| Field | Description |
|-------|-------------|
| `packages` | System packages (installed via apt/apk) |
| `osPackages` | Additional OS-level libraries |
| `repositories` | Additional package repositories |

#### Docker Settings

| Field | Description |
|-------|-------------|
| `baseOS` | Base Docker image OS (`alpine`, `ubuntu`) |
| `installOllama` | Force/suppress Ollama installation in Docker image (default: off - chat resources use the llamafile file backend) |

> LLM model is set per resource in `chat.model`. Backend, base URL, and API keys are configured in `~/.kdeps/config.yaml`. See [LLM Backends](../resources/llm-backends).

#### Environment

| Field | Description |
|-------|-------------|
| `args` | Build-time arguments |
| `env` | Runtime environment variables |

## SQL Connections

SQL connection strings (DSNs) live in `~/.kdeps/config.yaml` - never in `workflow.yaml`, which is version-controlled. Pool configuration lives in `workflow.yaml`.

### Configuration

`~/.kdeps/config.yaml` - credentials (machine-local, never committed):

```yaml
sql_connections:
  primary:
    connection: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
  analytics:
    connection: "mysql://analyst:pass@analytics-db:3306/analytics"
  cache:
    connection: "sqlite://./cache.db"
```

`workflow.yaml` - pool config only (no credentials):

```yaml
settings:
  sqlConnections:
    primary:
      pool:
        maxConnections: 25
        minConnections: 5
        maxIdleTime: "30m"
        connectionTimeout: "10s"
    analytics:
      pool:
        maxConnections: 10
        minConnections: 2
        maxIdleTime: "15m"
        connectionTimeout: "5s"
```

### Pool Configuration

| Field | Default (when `pool:` is omitted entirely) | Description |
|-------|---------|-------------|
| `maxConnections` | 10 | Maximum pool size |
| `minConnections` | 2 | Minimum idle connections |
| `maxIdleTime` | 5m | Max time before idle connection is closed |
| `connectionTimeout` | (none -- no connection lifetime limit) | Connection acquisition timeout |

These defaults (`pkg/config/defaults.yml`) apply only when `pool:` is left out of `sqlConnections.<name>` entirely. If `pool:` is present but a sub-field is omitted or empty, that sub-field is **not** backfilled with the default above -- it falls through to the Go `database/sql` driver's own zero-value behavior (effectively unbounded for `maxConnections`/`connectionTimeout`, no idle connections kept for `minConnections`). Set every field you care about explicitly once you add a `pool:` block.

### Using Named Connections

<div v-pre>

```yaml
# resources/example.yaml
sql:
  connectionName: primary  # Reference by name -- must match key in sql_connections in ~/.kdeps/config.yaml
  queries:
    - query: "SELECT * FROM users WHERE id = ?"
      params:
        - "{{ get('userId') }}"
```

</div>

## Trusted Proxies

Configure trusted proxies for accurate client IP detection behind load balancers. kdeps ignores `X-Forwarded-For` and `X-Real-IP` unless the direct peer matches an entry in this list.

### API Server

```yaml
# workflow.yaml
settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    trustedProxies:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
```

### Web Server

```yaml
# workflow.yaml
settings:
  webServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    trustedProxies:
      - "127.0.0.1"
      - "10.0.0.1"
```

## Environment Variable Expansion

Use environment variables in configuration values.

### Syntax

```yaml
# workflow.yaml
settings:
  agentSettings:
    env:
      # Direct reference
      API_KEY: "${API_KEY}"

      # With default value
      LOG_LEVEL: "${LOG_LEVEL:-info}"

      # Combined
      DATABASE_URL: "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:5432/${DB_NAME}"
```

### In SQL Connections

```yaml
# ~/.kdeps/config.yaml
sql_connections:
  primary:
    connection: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/${POSTGRES_DB}"
```

## Multiple Route Definitions

Define multiple routes with different methods and paths.

```yaml
# workflow.yaml
settings:
  apiServer:
    portNum: 16395
    routes:
      # Chat endpoint
      - path: /api/v1/chat
        methods: [POST]

      # Search endpoint
      - path: /api/v1/search
        methods: [GET, POST]

      # CRUD operations
      - path: /api/v1/users
        methods: [GET, POST]
      - path: /api/v1/users/:id
        methods: [GET, PUT, DELETE]

      # Health check
      - path: /health
        methods: [GET]
```

## Security

Auth, rate limiting, TLS (static certs or Let's Encrypt custom domains), body size cap, concurrency limits, and resource output caps. See [Security Reference](/reference/security) for the full documentation.

## Global Defaults (`defaults`, `resource_defaults`)

`~/.kdeps/config.yaml` can set two kinds of global default, applied whenever
a workflow doesn't set the equivalent field itself. Edit it with
[`kdeps edit`](/reference/cli/dev#kdeps-edit); [`kdeps doctor`](/reference/cli/dev#kdeps-doctor)
checks it (along with Ollama/Python/installed agents) before you hit run.

```yaml
# ~/.kdeps/config.yaml
defaults:
  timezone: "America/New_York"   # sets TZ env var for every agent
  python_version: "3.12"         # sets KDEPS_PYTHON_VERSION
  offline_mode: false            # sets KDEPS_OFFLINE_MODE=true when enabled

resource_defaults:               # applied to every resource of that type
  chat:
    timeout: 60s                 # hard stop per LLM call
    context_length: 4096
    streaming: false
    max_output_bytes: 1048576
  http:
    timeout: 30s
    follow_redirects: true
    retry_max_attempts: 3
    retry_backoff: 1s
    max_response_bytes: 10485760
  python:
    timeout: 60s
    max_output_bytes: 1048576
  exec:
    timeout: 30s
    max_output_bytes: 1048576
  sql:
    timeout: 30s
    max_rows: 1000
  onError:
    action: fail                 # fail | continue | retry
    max_retries: 3
    retry_delay: 1s
```

`defaults` mirrors a workflow's `agentSettings` (timezone, Python version);
`resource_defaults` mirrors the fields a resource sets on itself (a `chat:`
resource's own `timeout:`, for example) -- either can still be overridden
per-resource in `workflow.yaml`, which always wins over the global default.

Config is validated on load. Warnings go to stderr for unknown keys, missing
API keys, invalid durations, and agent profiles under `agents:` that don't
match any installed workflow's `metadata.name`.

## Agent Identity

An agent can have a configured identity — name, email, mailing address, and named accounts for services it authenticates with. Like SMTP/IMAP/bot credentials, it lives in `~/.kdeps/config.yaml`, never in `workflow.yaml`, and follows the same [per-agent profile](workflow.md#metadata-and-config-profiles) merge: set it globally, override it under `agents.<name>`, or both.

```yaml
# ~/.kdeps/config.yaml
identity:                        # global default identity
  name: "kdeps bot"
  email: bot@example.com

agents:
  sales-bot:                     # matches metadata.name: sales-bot
    identity:
      name: "Sales Bot"
      email: sales-bot@example.com
      address: "123 Example St, Springfield"
      accounts:                  # named credentials for tools that log into a service
        crm:
          username: "${CRM_USER}"
          password: "${CRM_PASS}"
          url: https://crm.example.com
```

Identity is `name`/`email`/`address` (attribution — who the agent is) plus `accounts` (credentials — what the agent can log into). They're used in three places:

- **Git commits.** In agent-loop mode, the `Co-Authored-By` trailer the agent adds to its own commits uses `"name <email>"` when configured, instead of the default `kdeps (<model>) <noreply@kdeps.com>`.
- **Outbound email.** An [`email:` resource](/resources/email) with no `from:` set defaults to `identity.email`.
- **`identity_get` tool.** Agent-loop mode registers this tool so the model can answer "who are you" or sign its own output — it returns `name`/`email`/`address` only. `accounts` (and its passwords) are never exposed to the LLM through this or any tool; a model that can read a credential can leak it in its own output.

`accounts` are consumed by resources that need to authenticate as the agent. An `httpClient` resource with `accountName: crm` and no `connectionName` resolves Basic Auth from `identity.accounts.crm` — `connectionName` (a full [named HTTP connection](/resources/http-client)) always takes priority when both are set.

## See Also

- [Security Reference](/reference/security) - Auth, rate limiting, TLS, concurrency, output caps
- [Workflow Configuration](workflow.md) - Basic workflow configuration
- [Session & Storage](session.md) - Session persistence
- [CORS](cors.md) - Cross-origin configuration
- [Docker Deployment](../deployment/docker.md) - Deployment options
