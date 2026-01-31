# Workflow Configuration

The `workflow.yaml` file is the central configuration for your AI agent. It defines metadata, API settings, web server configuration, agent settings, and database connections.

## Basic Structure

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent
  description: Agent description
  version: "1.0.0"
  targetActionId: responseResource

settings:
  apiServerMode: true
  apiServer: { ... }
  webServerMode: false
  webServer: { ... }
  agentSettings: { ... }
  sqlConnections: { ... }
  session: { ... }
```

## Metadata

```yaml
metadata:
  name: my-agent              # Required: Agent name (alphanumeric, hyphens)
  description: My AI agent    # Optional: Human-readable description
  version: "1.0.0"           # Required: Semantic version
  targetActionId: response    # Required: Default resource to execute
  workflows:                  # Optional: Import other workflows
    - "@other-agent"
```

### targetActionId
The `targetActionId` specifies which resource is executed as the main entry point. All routes return the output of this resource.

## API Server Settings

```yaml
settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"      # IP to bind (default: 127.0.0.1)
    portNum: 3000            # Port number (default: 3000)
    trustedProxies:          # Optional: For X-Forwarded-For
      - "192.168.1.0/24"
    routes:
      - path: /api/v1/chat
        methods: [POST]
      - path: /api/v1/query
        methods: [GET, POST]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:8080
        - https://myapp.com
      allowMethods:
        - GET
        - POST
        - OPTIONS
      allowHeaders:
        - Content-Type
        - Authorization
      allowCredentials: true
      maxAge: "24h"
```

### Routes

Each route maps an HTTP path to your workflow:

```yaml
routes:
  - path: /api/v1/users      # URL path
    methods: [GET, POST]     # Allowed HTTP methods
  - path: /api/v1/items/:id  # Path with parameter
    methods: [GET, PUT, DELETE]
```

Supported HTTP methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`

### CORS Configuration

```yaml
cors:
  enableCors: true           # Enable CORS headers
  allowOrigins:              # Allowed origins (* for all)
    - http://localhost:3000
  allowMethods:              # Allowed HTTP methods
    - GET
    - POST
  allowHeaders:              # Allowed request headers
    - Content-Type
    - Authorization
  exposeHeaders:             # Headers exposed to browser
    - X-Request-Id
  allowCredentials: true     # Allow cookies/auth
  maxAge: "24h"             # Preflight cache duration
```

### Trusted Proxies

For applications behind a reverse proxy:

```yaml
trustedProxies:
  - "127.0.0.1"
  - "10.0.0.0/8"
  - "192.168.0.0/16"
```

This allows `get('clientIp')` to return the correct IP from `X-Forwarded-For`.

## Web Server Settings

For serving static files or proxying to web applications:

```yaml
settings:
  webServerMode: true
  webServer:
    hostIp: "127.0.0.1"
    portNum: 8080
    routes:
      # Static file serving
      - path: "/"
        serverType: "static"
        publicPath: "./public"

      # Reverse proxy to app
      - path: "/app"
        serverType: "app"
        appPort: 8501
        command: "streamlit run app.py"
```

### Static File Serving

```yaml
- path: "/dashboard"
  serverType: "static"
  publicPath: "./web/dashboard"  # Relative to workflow directory
```

Serves files from the specified directory. Supports HTML, CSS, JS, images, and other static assets.

### Reverse Proxy

```yaml
- path: "/app"
  serverType: "app"
  publicPath: "./streamlit-app"
  appPort: 8501
  command: "streamlit run app.py"
```

- `appPort`: The port your application runs on
- `command`: Command to start the application
- Supports WebSocket connections

## Agent Settings

```yaml
settings:
  agentSettings:
    timezone: Etc/UTC              # Timezone for the container

    # Python settings
    pythonVersion: "3.12"          # Python version
    pythonPackages:                # pip packages to install
      - pandas
      - numpy
      - requests
    requirementsFile: "requirements.txt"  # Or use a file

    # System packages
    osPackages:                    # OS packages (apt/apk)
      - ffmpeg
      - imagemagick
      - tesseract-ocr
    repositories:                  # Additional apt repositories
      - ppa:alex-p/tesseract-ocr-devel

    # Docker settings
    baseOS: alpine                 # Base OS: alpine, ubuntu, debian

    # LLM settings
    models:                        # Ollama models to include
      - llama3.2:1b
      - llama3.2-vision
    offlineMode: false            # Pre-bake models in image
    ollamaImageTag: "0.13.5"      # Ollama version
    ollamaUrl: http://localhost:11434  # Custom Ollama URL

    # Environment
    env:
      API_KEY: "value"
      DEBUG: "true"
    args:
      BUILD_ARG: ""               # Docker build args
```

### Python Configuration

```yaml
pythonVersion: "3.12"        # Python 3.8 - 3.12 supported

# Option 1: List packages
pythonPackages:
  - pandas>=2.0
  - numpy
  - "torch[cuda]"

# Option 2: Use requirements file
requirementsFile: "requirements.txt"

# Option 3: Use pyproject.toml (with uv)
pyprojectFile: "pyproject.toml"
lockFile: "uv.lock"
```

### LLM Models

```yaml
models:
  - llama3.2:1b        # Small, fast
  - llama3.2           # Standard
  - llama3.2-vision    # Vision capable
  - mistral            # Mistral 7B
  - codellama          # Code generation
```

For offline/air-gapped deployments:

```yaml
offlineMode: true      # Models baked into Docker image
models:
  - llama3.2:1b
```

## SQL Connections

Define named database connections:

```yaml
settings:
  sqlConnections:
    analytics:
      connection: "postgres://user:pass@localhost:5432/analytics"
      pool:
        maxConnections: 10
        minConnections: 2
        maxIdleTime: "30s"
        connectionTimeout: "5s"

    inventory:
      connection: "mysql://user:pass@localhost:3306/inventory"
      pool:
        maxConnections: 5
        minConnections: 1

    cache:
      connection: "sqlite:///path/to/cache.db"
```

### Supported Databases

| Database | Connection String Format |
|----------|-------------------------|
| PostgreSQL | `postgres://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| SQLite | `sqlite:///path/to/file.db` |
| SQL Server | `sqlserver://user:pass@host:1433/db` |
| Oracle | `oracle://user:pass@host:1521/service` |

### Connection Pooling

```yaml
pool:
  maxConnections: 10        # Maximum pool size
  minConnections: 2         # Minimum idle connections
  maxIdleTime: "30s"       # Max idle time before close
  connectionTimeout: "5s"   # Connection timeout
```

## Session Configuration

For persistent session storage:

```yaml
settings:
  session:
    type: sqlite              # "sqlite" or "memory"
    path: ".kdeps/sessions.db"  # SQLite file path
    ttl: "30m"               # Session expiration
    cleanupInterval: "5m"    # Cleanup frequency
```

### Session Types

**SQLite (Persistent)**
```yaml
session:
  type: sqlite
  path: ".kdeps/sessions.db"
  ttl: "1h"
```

**Memory (Non-persistent)**
```yaml
session:
  type: memory
  ttl: "30m"
```

### Using Sessions

In resources, use `set()` and `get()` with session storage:

<div v-pre>

```yaml
expr:
  - set('user_id', '123', 'session')    # Store in session
  - set('visits', get('visits', 'session') + 1, 'session')

# Later retrieve
prompt: "User {{ get('user_id', 'session') }} has visited {{ get('visits', 'session') }} times"
```

</div>

## Environment Variables

```yaml
agentSettings:
  env:
    API_KEY: "your-api-key"
    DATABASE_URL: "postgres://..."
    DEBUG: "true"

  args:
    BUILD_SECRET: ""    # Value provided at build time
```

Environment variables are available:
- In Python scripts: `os.environ['API_KEY']`
- In shell commands: `$API_KEY`
- In expressions: `get('API_KEY', 'env')`

## Complete Example

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: production-agent
  description: Full-featured production AI agent
  version: "2.0.0"
  targetActionId: apiResponse

settings:
  apiServerMode: true
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 3000
    trustedProxies:
      - "10.0.0.0/8"
    routes:
      - path: /api/v1/chat
        methods: [POST]
      - path: /api/v1/analyze
        methods: [POST]
      - path: /health
        methods: [GET]
    cors:
      enableCors: true
      allowOrigins:
        - https://myapp.com
      allowCredentials: true

  webServerMode: true
  webServer:
    hostIp: "0.0.0.0"
    portNum: 8080
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./frontend/dist"

  agentSettings:
    timezone: America/New_York
    pythonVersion: "3.12"
    pythonPackages:
      - pandas
      - numpy
    osPackages:
      - ffmpeg
    baseOS: alpine
    models:
      - llama3.2:1b
    offlineMode: true
    env:
      LOG_LEVEL: "info"

  sqlConnections:
    main:
      connection: "postgres://user:pass@db:5432/app"
      pool:
        maxConnections: 20
        minConnections: 5

  session:
    type: sqlite
    path: "/data/sessions.db"
    ttl: "24h"
    cleanupInterval: "1h"
```

## Next Steps

- [Resources Overview](../resources/overview) - Learn about resource types
- [Unified API](../concepts/unified-api) - Master get() and set()
- [Docker Deployment](../deployment/docker) - Build container images
