# Workflow configuration

`workflow.yaml` is the entry point for a kdeps workflow. It declares metadata, the HTTP server or input source, agent settings, and SQL connections. Resources live in separate files under `resources/`. Credentials (HTTP auth, search API keys, bot tokens) live in `~/.kdeps/config.yaml` - not here.

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent           # required; must be alphanumeric + hyphens
  description: My agent    # optional
  version: "1.0.0"         # required; semantic version
  targetActionId: response # required; the resource whose output becomes the HTTP response

deploy:
  apiServer: { ... }       # HTTP REST server settings
  webServer: { ... }       # static file or app proxy settings
  runtime: { ... }         # runtime environment (Python, OS packages, LLM backend)
session: { ... }           # per-session key-value store: set(k, v, 'session')
repl: { ... }              # look of the interactive prompt (prompt, sessionId)
```

## Read next

- [Workflow configuration reference](/reference/workflow/configuration) - how the pieces fit together, basic structure, metadata and config profiles, interactive prompt, API server, web server, runtime, SQL connections
- [Global config](/reference/advanced-config) - Backend, defaults, named connections, and agent profiles
- [Resources overview](/workflow/resources) - Resource types and fields
- [Agencies](/agencies/) - Multi-agent orchestration
