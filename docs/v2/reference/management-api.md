# Management API

The management API updates a running kdeps server's workflow. You do not rebuild or redeploy the container. Every kdeps server exposes six endpoints under `/_kdeps/` alongside your normal agent routes. The [kdeps API](/reference/glossary#kdeps-api) is a different listener.

Workflow mode only.

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/_kdeps/status` | ✓ | Workflow name, version, description, resource count |
| `GET` | `/_kdeps/openapi` | ✓ | OpenAPI spec generated from the loaded workflow's routes |
| `GET` | `/_kdeps/schema` | ✓ | JSON Schema generated from the loaded workflow |
| `PUT` | `/_kdeps/workflow` | ✓ | Write a single workflow YAML, clear stale `resources/`, hot-reload |
| `PUT` | `/_kdeps/package` | ✓ | Extract a full `.kdeps` tar.gz archive (preserves `data/`, `scripts/`, etc.), hot-reload |
| `POST` | `/_kdeps/reload` | ✓ | Reload the workflow from the current on-disk file |

## Authentication

All management endpoints require a bearer token separate from workflow API auth. When you configure `apiServer`, the server also needs `KDEPS_API_AUTH_TOKEN` (or `api_auth_token` in `~/.kdeps/config.yaml`) before it starts.

```bash
export KDEPS_API_AUTH_TOKEN=api-secret
export KDEPS_MANAGEMENT_TOKEN=mysecret
kdeps run workflow.yaml
```

Workflow routes (`/api/*` and the other workflow routes) use `KDEPS_API_AUTH_TOKEN`. Management routes (`/_kdeps/*`) use `KDEPS_MANAGEMENT_TOKEN`.

Clients send the token in the `Authorization` header:

```
Authorization: Bearer mysecret
```

| Token state | Response |
|-------------|----------|
| `KDEPS_MANAGEMENT_TOKEN` unset | `503 Service Unavailable` |
| Token wrong or header missing | `401 Unauthorized` |
| Token correct | Handler runs |

## Size limits

| Endpoint | Limit | Over-limit response |
|----------|-------|---------------------|
| `PUT /_kdeps/workflow` | 5 MB | `413 Payload Too Large` |
| `PUT /_kdeps/package` | 200 MB | `413 Payload Too Large` |

kdeps rejects uploads that are too large before it writes data to disk.

## Status response

```json
{
  "status": "ok",
  "workflow": {
    "name": "my-agent",
    "version": "2.0.0",
    "description": "My AI agent",
    "targetActionId": "respond",
    "resources": 3
  }
}
```

If no workflow is loaded, the response has no `workflow` field.

## Examples

```bash
# Check status
curl -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
  http://localhost:16395/_kdeps/status

# Push a workflow YAML
curl -X PUT \
  -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
  -H "Content-Type: application/yaml" \
  --data-binary @workflow.yaml \
  http://localhost:16395/_kdeps/workflow

# Push a .kdeps bundle package archive
curl -X PUT \
  -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @myagent-2.0.0.kdeps \
  http://localhost:16395/_kdeps/package

# Reload from current on-disk file
curl -X POST \
  -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
  http://localhost:16395/_kdeps/reload
```

## Restart persistence

When you push a new workflow, kdeps writes it to the same path that you gave at startup (or `/app/workflow.yaml` in Docker). A push never changes the workflow path. It changes only the file contents. On the next restart, kdeps reads the updated file automatically.

After a YAML push, kdeps removes old `.yaml`/`.yml` files from the `resources/` directory next to it. The management API puts all resources into one `workflow.yaml`, so this prevents a second load of the same resources.

Package pushes extract the full archive in-place - `resources/`, `data/`, and `scripts/` are all replaced or added.

## Security

- kdeps rejects path-traversal entries in `.kdeps` archives with `422 Unprocessable Entity`.
- Per-file decompression cap of 500 MB guards against zip-bomb payloads.
- The maximum size of a response body is 1 MB.

## See also

- [Configuration reference](/workflow/configuration) - Full workflow.yaml settings
- [Deployment guide](/deploy/) - CI/CD pipeline walkthrough
- [Security](/reference/advanced-config) - Auth, rate limiting, TLS
