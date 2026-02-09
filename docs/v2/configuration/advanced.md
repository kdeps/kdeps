# Advanced Configuration

This guide covers advanced configuration options for KDeps workflows.

## Workflow Imports

Workflows can import resources and configurations from other workflows using the `workflows` field.

### Syntax

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: responseResource
  workflows:
    - "@base-agent"
    - "@shared-utils"
```

### How It Works

1. **Resource Inheritance**: Resources from imported workflows are available to your workflow
2. **Configuration Merging**: Settings are merged with local settings taking precedence
3. **Dependency Resolution**: Imported workflows are loaded in order

### Example: Shared Authentication

**base-workflow.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: base-agent
  version: "1.0.0"
  targetActionId: authCheck
settings:
  apiServerMode: true
```

**base-workflow/resources/auth.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: authCheck
  name: Authentication Check
run:
  expr:
    - set('token', get('Authorization'))
    - set('isValid', get('token') != '')
  skipCondition: "!get('isValid')"
```

**my-workflow.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: responseResource
  workflows:
    - "@base-agent"  # Import the base workflow
settings:
  apiServerMode: true
```

Your workflow now includes the `authCheck` resource automatically.

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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: logRequest
run:
  expr:
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
run:
  expr:
    # Different behavior based on request method
    - set('isPost', request.method == 'POST')
    - set('isGet', request.method == 'GET')
  skipCondition: "!get('isPost')"
```

### Logging and Auditing

<div v-pre>

```yaml
run:
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

    # LLM Configuration
    models:
      - llama3.2:1b
      - nomic-embed-text
    offlineMode: false
    ollamaImageTag: "0.3.0"
    ollamaUrl: "http://ollama:11434"

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

#### LLM Settings

| Field | Description |
|-------|-------------|
| `models` | Models to download/use |
| `offlineMode` | Run without internet access |
| `ollamaUrl` | Custom Ollama server URL |

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
run:
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
  apiServerMode: true
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
  webServerMode: true
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
  apiServerMode: true
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

## See Also

- [Workflow Configuration](workflow.md) - Basic workflow configuration
- [Session & Storage](session.md) - Session persistence
- [CORS](cors.md) - Cross-origin configuration
- [Docker Deployment](../deployment/docker.md) - Deployment options
