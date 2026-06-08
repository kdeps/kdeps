# Workflow Configuration

`workflow.yaml` is the entry point for a kdeps workflow. It declares metadata, the HTTP server or input source, agent settings, and SQL connections. Resources live in separate files under `resources/`. Credentials (SMTP, IMAP, HTTP auth, search API keys) live in `~/.kdeps/config.yaml` -- not here.

## How the pieces fit together

```d2
direction: down

A: incoming request {shape: oval}
B: "apiServer\nHTTP server: TLS, auth, rate limit, routes"
C: "resources/\nDAG pipeline: chat, sql, http, python, exec"
D: "apiResponse\nterminal resource; builds the HTTP response"
E: HTTP response {shape: oval}

A -> B -> C -> D -> E

settings: workflow.yaml settings {
  B2: "webServer\nstatic files or subprocess proxy"
  B3: "agentSettings\nPython, OS packages, env vars"
  B4: "sqlConnections\nnamed DB connections"
  B5: "session\ncross-request key-value store"
}

global: "~/.kdeps/config.yaml" {
  G1: "smtp_connections / imap_connections\nemail send + receive"
  G2: "http_connections\nHTTP auth + proxy"
  G3: "search_connections\nweb search API keys"
}

settings.B3 -> C: configures runtime
settings.B4 -> C: provides connections
settings.B5 -> C: provides session store
settings.B2 -> B: runs alongside
global.G1 -> C: provides connections
global.G2 -> C: provides connections
global.G3 -> C: provides connections
```

## Basic structure

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent           # required; must be alphanumeric + hyphens
  description: My agent    # optional
  version: "1.0.0"         # required; semantic version
  targetActionId: response # required; the resource whose output becomes the HTTP response

settings:
  apiServer: { ... }       # HTTP REST server settings
  webServer: { ... }       # static file or app proxy settings
  agentSettings: { ... }   # runtime environment (Python, OS packages, Ollama)
  sqlConnections: { ... }  # named database connections
  session: { ... }         # session persistence settings
```

Credentials and named connections (SMTP, IMAP, HTTP auth, search API keys) belong in `~/.kdeps/config.yaml`, not `workflow.yaml`. See [Global Config](/configuration/advanced) for the full reference.

## Metadata and config profiles

`metadata.name` maps to a per-agent profile in `~/.kdeps/config.yaml`. When the workflow runs, kdeps merges that profile on top of global config -- only the fields you specify override; everything else inherits.

```yaml
# ~/.kdeps/config.yaml
agents:
  my-agent:           # matches metadata.name: my-agent in workflow.yaml
    llm:
      backend: openai
      openai_api_key: sk-...
    defaults:
      timezone: America/New_York
```

In an [agency](/reference/glossary#agency), each agent resolves its own profile independently. Without a matching profile, the global config is used unchanged. On startup, kdeps warns about profiles that don't match any installed workflow name (non-fatal).

## API Server

`apiServer` starts an HTTP server. TLS certificate paths go in `settings` (not under `apiServer`).

```yaml
# workflow.yaml
settings:
  certFile: "/etc/certs/server.crt"   # TLS certificate PEM -- omit for plain HTTP
  keyFile:  "/etc/certs/server.key"   # TLS private key PEM
  apiServer:
    hostIp: "127.0.0.1"        # bind address (default: 127.0.0.1)
    portNum: 16395              # port (default: 16395)
    trustedProxies:
      - "10.0.0.0/8"           # IPs/CIDRs whose X-Forwarded-For header is trusted
    routes:
      - path: /api/v1/chat
        methods: [POST]        # GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
    cors:
      allowOrigins:
        - http://localhost:16395
    # auth token (required): KDEPS_API_AUTH_TOKEN env var or api_auth_token in ~/.kdeps/config.yaml
    rateLimit:
      requestsPerMinute: 60    # sustained per-IP rate
      burst: 10                # burst allowance above the sustained rate
    maxBodyBytes: 1048576      # 1 MB -- excludes multipart file uploads
    maxConcurrent: 50          # max in-flight requests; 0 = unlimited
```

See [Security](advanced.md#security) for the full security reference.

## Web Server

`webServer` serves static files or proxies to a running app process. Use it alongside `apiServer` to serve a frontend next to your API.

```yaml
# workflow.yaml
settings:
  webServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    trustedProxies:
      - "10.0.0.0/8"         # honor X-Forwarded-For from load balancer peers
    rateLimit:
      requestsPerMinute: 120   # optional; applies even without apiServer
      burst: 20
    maxBodyBytes: 1048576      # 1 MB -- excludes multipart file uploads
    maxConcurrent: 50          # max in-flight requests; 0 = unlimited
    routes:
      - path: "/"
        serverType: "static"   # serve files from publicPath
        publicPath: "./public"
      - path: "/app"
        serverType: "app"      # proxy to a subprocess on appPort
        appPort: 8501
        command: "streamlit run app.py"
```

## Agent Settings

`agentSettings` controls the runtime environment. These settings affect Docker image builds and local execution.

```yaml
# workflow.yaml
settings:
  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
    pythonPackages:
      - pandas
      - requests
    osPackages:
      - ffmpeg
    baseOS: alpine               # alpine (default), ubuntu, debian
    installOllama: true          # install Ollama in Docker image
    env:
      API_KEY: "value"           # environment variables available to all resources
```

Model selection goes in `chat.model` inside each resource file. Backend and API keys go in `~/.kdeps/config.yaml`. See [LLM Backends](../resources/llm-backends) for routing.

## SQL Connections

Named SQL connections are split across two files: the connection string (DSN) lives in `~/.kdeps/config.yaml` (machine-local, never committed), and pool config lives in `workflow.yaml`.

`~/.kdeps/config.yaml` - DSN (credentials stay here):

```yaml
sql_connections:
  analytics:
    connection: "postgres://user:pass@localhost:5432/analytics"
  cache:
    connection: "sqlite:///path/to/cache.db"
```

`workflow.yaml` - pool config (no credentials here):

```yaml
settings:
  sqlConnections:
    analytics:
      pool:
        maxConnections: 10    # max open connections in the pool
        minConnections: 2     # min idle connections kept alive
    cache:
      pool:
        maxConnections: 5
```

Resources reference a connection by name: `connectionName: analytics`. The name must match the key in `sql_connections` in `~/.kdeps/config.yaml`.

Supported: Postgres, MySQL, SQLite, Oracle, SQL Server, and any `database/sql` driver.

## Session

Session storage persists values set with `set('key', val, 'session')` across requests from the same caller.

```yaml
# workflow.yaml
settings:
  session:
    ttl: "30m"     # how long a session lives without activity
    type: sqlite   # storage backend
```

## See Also

- [Global Config](/configuration/advanced) - Backend, defaults, named connections, and agent profiles
- [Resources Overview](/resources/overview) - Resource types and fields
- [Agencies](/concepts/agency) - Multi-agent orchestration
