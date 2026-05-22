# Advanced Configuration

This reference covers security, rate limiting, trusted proxies, resource output caps, and other server-level settings that live in `workflow.yaml` under `settings:`.

## Request Object

The `request` object provides access to HTTP request metadata in expressions.

### Available Properties

| Property | Type | Description |
|----------|------|-------------|
| `request.method` | string | HTTP method (GET, POST, etc.) |
| `request.path` | string | Request path |
| `request.ip` | string | Client IP address |
| `request.id` | string | Unique request ID |
| `request.sessionId` | string | Session ID (if sessions enabled) |

### Usage Examples

```yaml
actionId: logRequest
after:
  # Access request metadata
  - set('method', request.method)
  - set('path', request.path)
  - set('clientIp', request.ip)
  - set('requestId', request.id)
  - set('session', request.sessionId)

  # Build log entry
  - set('logEntry', json({
      "timestamp": info('request.id'),
      "method": get('method'),
      "path": get('path'),
      "ip": get('clientIp'),
      "requestId": get('requestId')
    }))
```

### Request-Based Routing

```yaml
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
sql:
  connection: logs
  queries:
    - query: |
        INSERT INTO audit_log (request_id, method, path, ip, session_id, timestamp)
        VALUES (?, ?, ?, ?, ?, NOW())
      params:
        - "{{ request.id }}"
        - "{{ request.method }}"
        - "{{ request.path }}"
        - "{{ request.ip }}"
        - "{{ request.sessionId }}"
```

</div>

## Agent Settings

The `agentSettings` section configures the runtime environment.

### Complete Reference

```yaml
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
    baseOS: "ubuntu"  # alpine, ubuntu, debian

    # Docker/Ollama Configuration
    ollamaImageTag: "0.3.0"

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
| `baseOS` | Base Docker image OS |
| `ollamaImageTag` | Ollama Docker image version |

#### Docker Settings (extended)

| Field | Description |
|-------|-------------|
| `ollamaImageTag` | Ollama Docker image version |
| `installOllama` | Force/suppress Ollama installation in image |

> LLM model is set per resource in `chat.model`. Backend, base URL, and API keys are configured in `~/.kdeps/config.yaml`. See [LLM Backends](../resources/llm-backends).

#### Environment

| Field | Description |
|-------|-------------|
| `args` | Build-time arguments |
| `env` | Runtime environment variables |

## SQL Connections

Define named database connections for reuse across resources.

### Configuration

```yaml
settings:
  sqlConnections:
    primary:
      connection: "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
      pool:
        maxConnections: 25
        minConnections: 5
        maxIdleTime: "30m"
        connectionTimeout: "10s"

    analytics:
      connection: "mysql://analyst:pass@analytics-db:3306/analytics"
      pool:
        maxConnections: 10
        minConnections: 2
        maxIdleTime: "15m"
        connectionTimeout: "5s"

    cache:
      connection: "sqlite://./cache.db"
```

### Pool Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `maxConnections` | 25 | Maximum pool size |
| `minConnections` | 5 | Minimum idle connections |
| `maxIdleTime` | 30m | Max time before idle connection is closed |
| `connectionTimeout` | 10s | Connection acquisition timeout |

### Using Named Connections

<div v-pre>

```yaml
sql:
  connection: primary  # Reference by name
  queries:
    - query: "SELECT * FROM users WHERE id = ?"
      params:
        - "{{ get('userId') }}"
```

</div>

## Trusted Proxies

Configure trusted proxies for accurate client IP detection behind load balancers.

### API Server

```yaml
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
settings:
  sqlConnections:
    primary:
      connection: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/${POSTGRES_DB}"
```

## Multiple Route Definitions

Define multiple routes with different methods and paths.

```yaml
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

### Authentication

Protect the API server with a shared secret. When `auth.token` is set, every request must include it via `Authorization: Bearer <token>` or `X-Api-Key: <token>`. The `/health` endpoint is always exempt.

```yaml
settings:
  apiServer:
    auth:
      token: "${API_TOKEN}"
```

Omit `auth` (or leave `token` empty) to disable authentication entirely.

### Rate Limiting

Limit requests per client IP using a token-bucket algorithm. `requestsPerMinute` is the sustained rate; `burst` is the number of requests allowed above that rate in a single burst. Clients that exceed the limit receive a `429` response with a `Retry-After: 60` header.

```yaml
settings:
  apiServer:
    rateLimit:
      requestsPerMinute: 60
      burst: 10
```

### Body Size Limit

Cap the size of incoming request bodies. Requests that exceed `maxBodyBytes` receive a `413` response. This limit does not apply to `multipart/form-data` uploads, which are managed separately by the upload middleware.

```yaml
settings:
  apiServer:
    maxBodyBytes: 1048576   # 1 MiB
```

### TLS

Enable HTTPS by pointing `certFile` and `keyFile` at a PEM certificate and private key. These fields belong in `settings`, not in `apiServer`.

```yaml
settings:
  certFile: "/etc/certs/server.crt"
  keyFile:  "/etc/certs/server.key"
  apiServer:
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

### Concurrent Request Limit

Cap the number of simultaneous in-flight requests the server handles. When the limit is reached, new requests receive a `503 Service Unavailable` response immediately rather than queuing. Omit or set to `0` to disable.

```yaml
settings:
  apiServer:
    maxConcurrent: 50
```

### Resource Output Caps

Four environment variables limit how many bytes executor resources return to the workflow engine. Set them in the container environment or in `agentSettings.env`.

| Variable | Applies to |
|---|---|
| `KDEPS_EXEC_MAX_OUTPUT_BYTES` | Shell / exec resource stdout |
| `KDEPS_HTTP_MAX_RESPONSE_BYTES` | HTTP resource response body |
| `KDEPS_CHAT_MAX_OUTPUT_BYTES` | LLM chat response content |
| `KDEPS_PYTHON_MAX_OUTPUT_BYTES` | Python resource stdout |

```yaml
settings:
  agentSettings:
    env:
      KDEPS_EXEC_MAX_OUTPUT_BYTES: "524288"    # 512 KiB
      KDEPS_HTTP_MAX_RESPONSE_BYTES: "1048576" # 1 MiB
      KDEPS_CHAT_MAX_OUTPUT_BYTES: "1048576"   # 1 MiB
      KDEPS_PYTHON_MAX_OUTPUT_BYTES: "524288"  # 512 KiB
```

## See Also

- [Workflow Configuration](workflow.md) - Basic workflow configuration
- [Session & Storage](session.md) - Session persistence
- [CORS](cors.md) - Cross-origin configuration
- [Docker Deployment](../deployment/docker.md) - Deployment options
