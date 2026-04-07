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
    portNum: 16395            # Port number (default: 16395)
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
        - http://localhost:16395
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
    - http://localhost:16395
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
    portNum: 16395
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
    models:                        # Ollama models to include (also enforced as allowlist)
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

### Model Allowlist Enforcement

When `agentSettings.models` is set, it acts as an **allowlist**: any resource or component that requests a model not in the list is automatically overridden with `models[0]` (the first listed model), and a red error is logged.

```yaml
agentSettings:
  models:
    - llama3.3:latest   # ← Only this model is permitted at runtime
```

If a resource specifies `model: llama3.2:1b` but the allowlist only contains `llama3.3:latest`, the request will use `llama3.3:latest` and log:
```
model not in workflow allowlist — overriding with first allowlisted model
```

This ensures a consistent, auditable model selection across all resources in a workflow.

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
    portNum: 16395
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
    portNum: 16395
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

---

## Input Configuration

The `settings.input` block specifies how the workflow receives input. Workflows can accept input from one or more sources simultaneously: HTTP API requests, audio hardware, video hardware, and telephony devices.

### Sources (required)

`sources` is a required array of one or more source types:

| Source | Description |
|--------|-------------|
| `api` | HTTP API requests (default) |
| `audio` | Audio hardware device (microphone, line-in) |
| `video` | Video hardware device (camera) |
| `telephony` | Phone or SIP device (local hardware or cloud provider) |

HTTP only (default):
```yaml
settings:
  input:
    sources: [api]
```

Microphone only:
```yaml
settings:
  input:
    sources: [audio]
```

Audio and video simultaneously:
```yaml
settings:
  input:
    sources: [audio, video]
```

API and microphone together:
```yaml
settings:
  input:
    sources: [api, audio]
```

### Audio Source

Required when `audio` is in `sources`:

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0          # ALSA device (Linux), e.g. "default", "hw:0,0"
                              # macOS: "Built-in Microphone"
                              # Windows: "Microphone (Realtek)"
```

### Video Source

Required when `video` is in `sources`:

```yaml
settings:
  input:
    sources: [video]
    video:
      device: /dev/video0     # V4L2 device (Linux)
                              # macOS: "FaceTime HD Camera"
                              # Windows: "USB Video Device"
```

### Telephony Source

Required when `telephony` is in `sources`:

```yaml
settings:
  input:
    sources: [telephony]
    telephony:
      type: local             # local | online
      device: /dev/ttyUSB0    # Required when type is "local"
      provider: twilio        # Required when type is "online"
```

### Activation (Wake Phrase)

Optional. When configured, the workflow only triggers when the wake phrase is detected. Works with non-API sources.

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0
    activation:
      phrase: "hey kdeps"     # Required: wake phrase to listen for
      mode: offline           # online | offline
      sensitivity: 0.9        # 0.0–1.0 (1.0 = exact match)
      chunkSeconds: 3         # Audio probe duration in seconds
      online:
        provider: deepgram
        apiKey: dg-...
      offline:
        engine: faster-whisper
        model: small
```

### Transcriber (Speech-to-Text)

Optional. Converts audio/video/telephony input to text (or saves raw media). Not used when all sources are `api`.

```yaml
settings:
  input:
    sources: [audio]
    audio:
      device: hw:0,0
    transcriber:
      mode: offline           # online | offline
      output: text            # text (transcript) | media (raw file)
      language: en-US         # BCP-47 language code
      online:
        provider: openai-whisper
        apiKey: sk-...
      offline:
        engine: faster-whisper
        model: small
```

After transcription, access results via:
- `inputTranscript` — the text transcript
- `inputMedia` — path to the saved media file
- `get("inputTranscript")` — unified API equivalent

### Multi-Source Example

```yaml
settings:
  input:
    sources: [audio, video]
    audio:
      device: hw:0,0
    video:
      device: /dev/video0
    activation:
      phrase: "hey kdeps"
      mode: offline
      offline:
        engine: faster-whisper
        model: small
    transcriber:
      mode: offline
      output: text
      offline:
        engine: faster-whisper
        model: small
```

See the [Input Sources guide](../concepts/input-sources) for complete examples, and the [TTS resource](../resources/tts) for the speech output side of voice workflows.

## Self-Tests

An optional `tests:` block lets you define HTTP-level smoke tests that run against the live server when you pass `--self-test` or `--self-test-only` to `kdeps run`.

### Scaffold tests automatically

```bash
# Generates tests from your resources and appends them to workflow.yaml
kdeps run workflow.yaml --write-tests
```

`--write-tests` inspects every resource in your workflow and creates one or more test cases per resource - validation resources get both a valid and an invalid test, LLM resources get a prompt-body test, HTTP client resources with static URLs are tested directly, and so on. A `GET /health -> 200` smoke test is always included.

### Manual tests

```yaml
tests:
  - name: "health check"
    request:
      method: GET
      path: /health
    assert:
      status: 200

  - name: "chat endpoint"
    request:
      method: POST
      path: /api/v1/chat
      body:
        message: "hello"
    assert:
      status: 200
      body:
        jsonPath:
          - path: "$.reply"
            exists: true

  - name: "validation rejects missing fields"
    request:
      method: POST
      path: /api/v1/apply
      body: {}
    assert:
      status: 400
    timeout: "10s"
```

### Test case fields

| Field | Description |
|-------|-------------|
| `name` | Human-readable label (required) |
| `request.method` | HTTP method - GET, POST, PUT, DELETE, PATCH (default: GET) |
| `request.path` | Request path (required) |
| `request.headers` | Optional map of request headers |
| `request.body` | Request body, JSON-encoded when sent |
| `request.query` | Optional map of URL query parameters |
| `assert.status` | Expected HTTP status code (0 = no check) |
| `assert.headers` | Expected response header substrings |
| `assert.body.contains` | Raw body must contain this substring |
| `assert.body.equals` | Raw body must exactly equal this string |
| `assert.body.jsonPath` | List of JSONPath assertions (see below) |
| `timeout` | Per-test timeout, e.g. `"30s"` (default: 30s) |

### JSONPath assertions

Each entry under `assert.body.jsonPath` supports one of:

```yaml
assert:
  body:
    jsonPath:
      - path: "$.success"       # JSONPath expression
        equals: true            # value must equal this (type-aware)
      - path: "$.data.name"
        contains: "Alice"       # string value must contain this
      - path: "$.items[0]"
        exists: true            # key/index must exist (any value)
```

### Running tests

```bash
# Start server and run tests once, keep server running
kdeps run workflow.yaml --self-test

# CI/CD: run tests then exit (non-zero on any failure)
kdeps run workflow.yaml --self-test-only
```

When no `tests:` block is present, both flags auto-generate smoke tests from routes and resources at runtime without modifying the file.
