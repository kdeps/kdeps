# Workflow Configuration

`workflow.yaml` is the entry point for a kdeps workflow. It declares metadata, the HTTP server or input source, agent settings, SQL connections, and IMAP connections. Resources live in separate files under `resources/`.

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
  B6: "smtpConnections / imapConnections\nemail send + receive"
  B7: "httpConnections\nHTTP auth + proxy"
  B8: "searchConnections\nweb search API keys"
  B5: "session\ncross-request key-value store"
}

settings.B3 -> C: configures runtime
settings.B4 -> C: provides connections
settings.B6 -> C: provides connections
settings.B7 -> C: provides connections
settings.B8 -> C: provides connections
settings.B5 -> C: provides session store
settings.B2 -> B: runs alongside
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
  sqlConnections: { ... }    # named database connections
  smtpConnections: { ... }   # named SMTP connections for email send
  imapConnections: { ... }   # named IMAP connections for email read/search/modify
  httpConnections: { ... }   # named HTTP auth + proxy (for httpClient resources)
  searchConnections: { ... } # named API keys for web search (brave/bing/tavily)
  session: { ... }           # session persistence settings
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

## SMTP Connections

Named SMTP connections are declared here and referenced by name in `email:` resources that use `action: send`.

```yaml
# workflow.yaml
settings:
  smtpConnections:
    default:
      host: "${SMTP_HOST}"      # e.g. smtp.gmail.com
      port: 587                 # 465 for implicit TLS, 587 for STARTTLS
      username: "${SMTP_USER}"
      password: "${SMTP_PASS}"
      tls: false                # false = STARTTLS on port 587, true = implicit TLS on port 465
```

Reference by name in a resource:

```yaml
email:
  action: send
  smtpConnection: default   # references settings.smtpConnections.default
  from: "reports@example.com"
  to: ["alice@example.com"]
  subject: "Report"
  body: "..."
```

See [Email Resource](/resources/email) for full field reference.

## IMAP Connections

Named IMAP connections are declared here and referenced by name in `email:` resources that use `action: read`, `search`, or `modify`.

```yaml
# workflow.yaml
settings:
  imapConnections:
    inbox:
      host: "${IMAP_HOST}"      # e.g. imap.gmail.com
      port: 993                 # 993 for TLS, 143 for plain
      username: "${IMAP_USER}"
      password: "${IMAP_PASS}"
      tls: true
```

Reference by name in a resource:

```yaml
email:
  action: read
  imapConnection: inbox   # references settings.imapConnections.inbox
  mailbox: "INBOX"
  limit: 20
```

See [Email Resource](/resources/email) for full field reference.

## HTTP Connections

Named HTTP connections hold auth credentials and proxy settings for `httpClient:` resources. This keeps API keys and passwords out of resource files.

```yaml
# workflow.yaml
settings:
  httpConnections:
    stripe:
      auth:
        type: bearer             # basic | bearer | api_key | oauth2
        token: "${STRIPE_KEY}"
    internal:
      auth:
        type: basic
        username: "${API_USER}"
        password: "${API_PASS}"
    via-proxy:
      proxy: "http://${PROXY_HOST}:${PROXY_PORT}"
```

Reference by name in a resource:

```yaml
httpClient:
  method: POST
  url: "https://api.stripe.com/v1/charges"
  connectionName: stripe   # references settings.httpConnections.stripe
```

See [HTTP Client Resource](/resources/http-client) for full field reference.

## Search Connections

Named search connections hold API keys for paid web search providers (Brave, Bing, Tavily). DuckDuckGo requires no connection.

```yaml
# workflow.yaml
settings:
  searchConnections:
    brave:
      apiKey: "${BRAVE_API_KEY}"
    tavily:
      apiKey: "${TAVILY_API_KEY}"
```

Reference by name in a resource:

```yaml
searchWeb:
  query: "{{ get('q') }}"
  provider: brave
  connectionName: brave   # references settings.searchConnections.brave
```

See [Search Resource](/resources/search) for full field reference.

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
