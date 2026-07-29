# Chapter 14: Workflow Configuration

`workflow.yaml` is the entry point for every kdeps workflow. It declares the workflow's identity, the HTTP server configuration, runtime settings, database connections, and agent behavior. This chapter is the full reference for every field in `workflow.yaml`.

## Top-Level Structure

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent
  version: "1.0.0"
  description: "What this workflow does"
  targetActionId: respond        # actionId of the terminal resource

settings:
  apiServer:                     # HTTP server configuration
    # ...
  webServer:                     # optional: static files / subprocess proxy
    # ...
  agentSettings:                 # Python, OS packages, env vars, model config
    # ...
  sqlConnections:                # named database connections
    # ...
  session:                       # session storage configuration
    # ...
```

## metadata

```yaml
metadata:
  name: my-agent            # required; alphanumeric + hyphens; becomes tool name in agent mode
  version: "1.0.0"          # semantic version string
  description: "..."        # optional; shown in logs and as tool description in agent mode
  targetActionId: respond   # required; actionId of the terminal apiResponse resource
```

**`name`** must be unique within an agency when multiple agents are used together. It becomes the tool name the LLM sees in agent mode and the identifier used in `agent:` resource calls.

**`targetActionId`** identifies which resource builds the HTTP response. Only resources in the dependency chain of this resource are executed. Resources unreachable from `targetActionId` are ignored.

## apiServer

Configures the HTTP server:

```yaml
settings:
  # TLS: certFile and keyFile are top-level under settings, not under apiServer
  certFile: "/certs/server.crt"
  keyFile: "/certs/server.key"

  apiServer:
    hostIp: "127.0.0.1"      # bind address; use "0.0.0.0" for all interfaces
    portNum: 16395             # TCP port
    routes:                   # list of accepted paths and methods
      - path: /api/v1/chat
        methods: [POST]
      - path: /api/v1/status
        methods: [GET]

    # Rate limiting (optional)
    rateLimit:
      requestsPerMinute: 60
      burst: 10

    # Trusted proxies (optional)
    trustedProxies:
      - "10.0.0.0/8"
      - "172.16.0.0/12"

    # CORS (optional) — covered in Chapter 15
    cors:
      allowOrigins: ["*"]

    # Max incoming request body size
    maxBodyBytes: 10485760    # 10 MB

# Auth token: set api_auth_token in ~/.kdeps/config.yaml or KDEPS_API_AUTH_TOKEN env var
```

### Routes

Each route specifies a path and the HTTP methods it accepts:

```yaml
routes:
  - path: /api/v1/chat
    methods: [POST]
  - path: /api/v1/search
    methods: [GET, POST]
  - path: /api/v1/admin
    methods: [GET, POST, PUT, DELETE]
```

Resources use `validations.routes` and `validations.methods` to filter which route they respond to. The server accepts all configured routes and dispatches to the resource DAG.

### TLS

Static PEM certificates (paths are **top-level** under `settings`, not under `apiServer`):

```yaml
settings:
  certFile: "/etc/ssl/server.crt"
  keyFile: "/etc/ssl/server.key"
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
```

Alternatively, terminate TLS at a reverse proxy (Nginx, Caddy, Traefik) or Kubernetes Ingress and keep kdeps on plain HTTP internally.

### Let's Encrypt (custom domain)

Automatic certificates for a public hostname via ACME — no PEM files to mount:

```yaml
settings:
  letsEncrypt:
    domain: api.example.com          # required
    domains:                         # optional SANs
      - www.example.com
    email: ops@example.com
    cacheDir: /var/lib/kdeps/letsencrypt   # default ~/.kdeps/letsencrypt
    staging: false                   # true while testing (untrusted CA)
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
```

**Requirements:** DNS A/AAAA for every hostname → this host; open **TCP 80** (HTTP-01) and **TCP 443** (HTTPS). Persist `cacheDir` across restarts so renewals keep working.

**Priority:** if both `certFile`/`keyFile` and `letsEncrypt` are set, static PEM wins. The same settings apply to `webServer` when enabled.

Use `staging: true` first to avoid production rate limits, then switch to production with a clean or separate cache directory.

Docs: `docs/v2/deployment/tls-https.md` and Security reference TLS section.


### Authentication

The API server auth token lives in `~/.kdeps/config.yaml`, not in `workflow.yaml`:

```yaml
# ~/.kdeps/config.yaml
api_auth_token: "${API_TOKEN}"
```

Or set the environment variable `KDEPS_API_AUTH_TOKEN`. When `apiServer` is configured, the token is required — kdeps refuses to start without it. Requests without a valid `Authorization: Bearer <token>` or `X-Api-Key: <token>` header receive 401. The `/health` endpoint is always exempt. `/_kdeps/*` management routes use `KDEPS_MANAGEMENT_TOKEN` instead (Chapter 21).

## agentSettings

Configures the runtime environment for resource execution:

```yaml
settings:
  agentSettings:
    # Python packages installed before first Python resource runs
    pythonPackages:
      - pandas==2.1.0
      - requests
      - beautifulsoup4
      - numpy
    
    # OS-level packages (requires appropriate base image in Docker)
    osPackages:
      - ffmpeg
      - poppler-utils
      - tesseract-ocr
    
    # Environment variables available to exec: and python: resources
    envVars:
      LOG_LEVEL: "info"
      API_BASE_URL: "https://api.internal.example.com"
      OPENAI_API_KEY: "${OPENAI_API_KEY}"    # passed from shell environment
      DATABASE_URL: "${DATABASE_URL}"
    
    # Timezone for the agent runtime (IANA timezone name)
    timezone: Etc/UTC
    
    # Default LLM model (overridden per resource)
    defaultModel: llama3.2:1b
    
    # Default LLM timeout (overridden per resource)
    defaultTimeout: 60s
    
    # Base OS for Docker image (default: alpine)
    baseOS: alpine            # alpine | ubuntu | debian
    
    # Python version (default: system Python; specify to pin)
    pythonVersion: "3.12"
    
    # Ollama opt-in (default backend is llamafile - no server install)
    installOllama: true       # bake the ollama server into Docker/ISO builds
    env:
      KDEPS_DEFAULT_BACKEND: ollama   # route chat resources to ollama at runtime
    models:                   # models to pull at startup (Ollama only)
      - llama3.2:1b
      - llama3.2:7b
      - nomic-embed-text
```

**`timezone`** — sets the timezone for the agent runtime. Accepts any IANA timezone name (`America/New_York`, `Europe/Amsterdam`, `Asia/Tokyo`). Defaults to `Etc/UTC`. This affects timestamp formatting and any time-aware logic in Python resources.

**`installOllama`** — when `true`, kdeps bakes the Ollama server into Docker and ISO builds. When omitted, no LLM server is installed: chat resources run on the default file backend, and the referenced llamafiles are pre-baked into the image instead. Pair `installOllama: true` with `env: {KDEPS_DEFAULT_BACKEND: ollama}` so the runtime routes chat calls to Ollama.

**`models:`** — list of Ollama models to pull at startup. kdeps runs `ollama pull <model>` for each entry before serving requests. This ensures your container or binary has the required models ready before the first request arrives.

**`pythonPackages`** — installed via pip at startup. Version pinning (`pandas==2.1.0`) is recommended for reproducibility. Unpinned packages use the latest version at the time of installation.

**`osPackages`** — installed via the system package manager. Useful for workflows that call CLI tools via `exec:`. When packaging as Docker, these become `RUN apt-get install` instructions.

**`envVars`** — made available to `exec:` commands and Python scripts via the process environment. The `${VAR}` syntax passes through from the host environment without embedding the actual value in the workflow file.

**`defaultModel`** — the model used by `chat:` resources that do not specify their own `model:` field. Useful for ensuring all LLM calls in a workflow use the same model unless explicitly overridden.

**`baseOS`** — base Linux distribution for the generated Docker image. Options: `alpine` (default, smallest image), `ubuntu`, `debian`. Switch to `ubuntu` or `debian` when your `osPackages` require apt-managed libraries not available in Alpine's apk repository.

**`pythonVersion`** — Python version to install in the Docker image (e.g., `"3.12"`). When omitted, kdeps uses the system Python bundled with the base image. Pin this when your `pythonPackages` require a specific Python version for compatibility.

Three additional keys — `replicas`, `resources`, and `networkPolicy` — only affect `kdeps export k8s` output and are covered in Chapter 18.

## sqlConnections

SQL connection strings (DSNs) live in `~/.kdeps/config.yaml`, not in `workflow.yaml`. Pool settings go in `workflow.yaml`.

`~/.kdeps/config.yaml`:

```yaml
sql_connections:
  main:
    connection: "${DATABASE_URL}"
  readonly:
    connection: "${DATABASE_READONLY_URL}"
  cache:
    connection: "sqlite:///data/cache.db"
  legacy:
    connection: "${MYSQL_URL}"
```

`workflow.yaml` (pool settings only):

```yaml
settings:
  sqlConnections:
    main:
      pool:
        maxConnections: 10
        minConnections: 5
        maxIdleTime: "1h"
    readonly:
      pool:
        maxConnections: 20
```

Connection names must be unique and are referenced in `sql:` resources via `connectionName:`. Supported databases: PostgreSQL, MySQL, SQLite, SQL Server, Oracle.

**URL formats:**
- PostgreSQL: `postgresql://user:pass@host:5432/dbname?sslmode=require`
- MySQL: `mysql://user:pass@tcp(host:3306)/dbname`
- SQLite: (use `path:` instead of `url:`)

## A Production-Ready workflow.yaml

Putting it all together:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: document-processor
  version: "2.1.0"
  description: "Processes and indexes uploaded documents. Supports PDF, TXT, HTML."
  targetActionId: respond

settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 8080
    routes:
      - path: /api/v1/process
        methods: [POST]
      - path: /api/v1/search
        methods: [GET, POST]
      - path: /health
        methods: [GET]
    # auth token: set api_auth_token in ~/.kdeps/config.yaml or KDEPS_API_AUTH_TOKEN env var
    rateLimit:
      requestsPerMinute: 30
      burst: 5
    cors:
      allowOrigins:
        - "https://app.example.com"
      allowMethods: [POST, GET, OPTIONS]
      allowCredentials: true
    trustedProxies:
      - "10.0.0.0/8"
  
  agentSettings:
    timezone: Etc/UTC
    pythonPackages:
      - pdfplumber==0.10.3
      - beautifulsoup4
    osPackages:
      - poppler-utils
    envVars:
      LOG_LEVEL: "info"
      STORAGE_BUCKET: "${STORAGE_BUCKET}"
    defaultModel: llama3.2:7b
    defaultTimeout: 120s
  
  sqlConnections:
    main:
      pool:
        maxConnections: 10
    cache:
      pool:
        maxConnections: 5
  # DSNs go in ~/.kdeps/config.yaml:
  # sql_connections:
  #   main: { connection: "${DATABASE_URL}" }
  #   cache: { connection: "sqlite:///data/doc-cache.db" }
  
  session:
    type: sqlite
    path: "/data/sessions.db"
    ttl: "4h"
    cleanupInterval: "30m"
```

This configuration supports 30 authenticated requests per minute, processes documents with Python and system tools, connects to a PostgreSQL database and a local SQLite cache, and maintains session state for multi-turn interactions.

## Environment Variable Best Practices

Always use `${VAR}` in `workflow.yaml` to reference sensitive values. Never commit actual credentials.

For local development, create a `.env` file and source it before running kdeps:

```bash
$ source .env && kdeps run workflow.yaml
```

Or use `direnv` to automatically load `.env` in the project directory.

For Docker/Kubernetes deployment, inject environment variables via Docker `--env-file` or Kubernetes `Secret` and `ConfigMap`. The workflow file itself never changes between environments.

This separation — workflow definition in git, credentials in environment — is what makes the same `workflow.yaml` work identically from a developer laptop to a production Kubernetes cluster.

X> ## Exercise
X>
X> Take the chatbot from Chapter 2 and configure it for three distinct deployment environments — local dev, staging, and production — without changing `workflow.yaml`.
X>
X> 1. Add a `sqlConnections` block with a connection named `analytics` that reads its URL from `${DATABASE_URL}`.
X> 2. Add an `agentSettings` block that pins `pythonVersion: "3.11"`, sets `timezone: Europe/Amsterdam`, and adds `pandas` and `requests` to `pythonPackages`.
X> 3. Add `baseOS: ubuntu` (since pandas needs glibc libraries not available on Alpine).
X> 4. Set `defaultModel: llama3.2:1b` and `defaultTimeout: 120s`.
X>
X> Then verify the separation of concerns:
X> - Run locally with `DATABASE_URL=sqlite:///local.db kdeps run workflow.yaml`
X> - Confirm `kdeps validate workflow.yaml` passes
X> - Check that none of the database credentials appear literally in `workflow.yaml` — only `${DATABASE_URL}` references
X>
X> **Stretch goal:** Create a `~/.kdeps/config.yaml` with a per-agent profile for your chatbot's `metadata.name`. Set a different `llm.backend` in the profile and confirm that the profile overrides the global config on startup.
