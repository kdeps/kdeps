# Workflow Configuration

`workflow.yaml` is the central configuration for your agent — it defines metadata, API settings, agent settings, and resource execution order.

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
  name: my-agent           # Required: Agent name (alphanumeric, hyphens)
  description: My agent    # Optional
  version: "1.0.0"         # Required: Semantic version
  targetActionId: response # Required: Entry point resource actionId
```

### Agent Name and Config Profiles

The `metadata.name` field maps to the corresponding config profile in `~/.kdeps/config.yaml`. When a workflow runs, kdeps looks for `agents.<name>` and merges its values on top of the global config.

```yaml
# ~/.kdeps/config.yaml
agents:
  my-agent:           # matches metadata.name: my-agent
    llm:
      backend: openai
      openai_api_key: sk-...
    defaults:
      timezone: America/New_York
```

Only the fields you specify in the profile override global values — everything else inherits. In an agency, each agent resolves its own profile independently. Components inherit the calling agent's profile. Without a matching profile, the global config is used unchanged.

**Validation:** On startup, kdeps validates the config file and prints warnings to stderr. Agent profiles that don't match any installed workflow `metadata.name` are flagged. Empty agent profiles (no non-zero fields) are also reported. These warnings are non-fatal — the config loads and runs as usual.

## API Server

```yaml
settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"       # Bind address (default: 127.0.0.1)
    portNum: 16395             # Port (default: 16395)
    trustedProxies:
      - "10.0.0.0/8"
    routes:
      - path: /api/v1/chat
        methods: [POST]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:16395
```

Supported methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, `HEAD`.

## Web Server

For serving static files or proxying to web apps:

```yaml
settings:
  webServerMode: true
  webServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./public"
      - path: "/app"
        serverType: "app"
        appPort: 8501
        command: "streamlit run app.py"
```

## Agent Settings

```yaml
settings:
  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
    pythonPackages:
      - pandas
      - requests
    osPackages:
      - ffmpeg
    baseOS: alpine               # alpine, ubuntu, debian
    installOllama: true
    env:
      API_KEY: "value"
```

### LLM Model

Set per resource in `run.chat.model`. Backend and API keys are in `~/.kdeps/config.yaml`.

```yaml
run:
  chat:
    model: llama3.2:1b
    role: user
    prompt: "{{ get('q') }}"
```

Set `model: router` to use the LLM router. See [LLM Backends](../resources/llm-backends).

## SQL Connections

Named database connections — available to any SQL resource:

```yaml
settings:
  sqlConnections:
    analytics:
      connection: "postgres://user:pass@localhost:5432/analytics"
      pool:
        maxConnections: 10
        minConnections: 2
    cache:
      connection: "sqlite:///path/to/cache.db"
```

Supported: Postgres, MySQL, SQLite, Oracle, SQL Server, and any `database/sql` driver.

## Session

```yaml
settings:
  session:
    enabled: true
    expiresIn: "30m"
    namespace: "my-agent"
```
