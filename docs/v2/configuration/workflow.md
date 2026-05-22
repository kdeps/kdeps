# Workflow Configuration

`workflow.yaml` is the entry point for a kdeps workflow. It declares metadata, the HTTP server or input source, agent settings, and SQL connections. Resources live in separate files under `resources/`.

## How the pieces fit together

```mermaid
flowchart TD
    A(["incoming request"]) --> B
    B["apiServer<br/><small>HTTP server: TLS, auth, rate limit, routes</small>"] --> C
    C["resources/<br/><small>DAG pipeline: chat, sql, http, python, exec</small>"] --> D
    D["apiResponse<br/><small>terminal resource; builds the HTTP response</small>"] --> E
    E(["HTTP response"])

    B2["webServer &#40;optional&#41;<br/><small>serves static files or proxies subprocess</small>"]
    B3["agentSettings<br/><small>Python packages, OS packages, env vars</small>"]
    B4["sqlConnections<br/><small>named DB connections for sql: resources</small>"]
    B5["session<br/><small>persists set&#40;'key', val, 'session'&#41; across requests</small>"]
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
    auth:
      token: "${API_TOKEN}"    # require Bearer or X-Api-Key header; omit to disable
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

Named connections are declared here and referenced by name in `sql:` resources. The name `analytics` below becomes `connectionName: analytics` in any SQL resource.

```yaml
# workflow.yaml
settings:
  sqlConnections:
    analytics:
      connection: "postgres://user:pass@localhost:5432/analytics"
      pool:
        maxConnections: 10    # max open connections in the pool
        minConnections: 2     # min idle connections kept alive
    cache:
      connection: "sqlite:///path/to/cache.db"
```

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

- [Global Config](/configuration/advanced) - Backend, defaults, and agent profiles
- [Resources Overview](/resources/overview) - Resource types and fields
- [Agencies](/concepts/agency) - Multi-agent orchestration
