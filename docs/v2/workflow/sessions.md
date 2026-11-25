# Session configuration

Sessions let resources persist values across multiple HTTP requests from the same caller using `set('key', val, 'session')`. Configure the storage backend in `workflow.yaml` under the top-level `session:` block.

```yaml
# workflow.yaml
session:
  type: sqlite                    # "sqlite" or "memory"
  path: "/var/lib/kdeps/sessions.db"  # optional; default ~/.kdeps/sessions.db. Keep it outside the project folder
  ttl: "30m"                      # Session expiration
  cleanupInterval: "5m"           # Cleanup frequency
```

## Read next

- [Session configuration reference](/reference/workflow/sessions) - configuration, session types, TTL (time to live), using sessions, examples, session ID, cleanup, docker considerations
- [Workflow configuration](/workflow/configuration) - Full settings reference
- [Data access](/workflow/data-access) - get() and set() usage
- [Docker deployment](/deploy/docker) - Production setup
