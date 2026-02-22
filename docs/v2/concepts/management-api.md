# Management API

The built-in management API lets you update a running kdeps server's workflow without rebuilding or redeploying the container. Every kdeps server exposes four endpoints under `/_kdeps/` alongside your normal agent routes.

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/_kdeps/status` | — | Workflow name, version, description, resource count |
| `PUT` | `/_kdeps/workflow` | ✓ | Write a single workflow YAML, clear stale `resources/`, hot-reload |
| `PUT` | `/_kdeps/package` | ✓ | Extract a full `.kdeps` tar.gz archive (preserves `data/`, `scripts/`, etc.), hot-reload |
| `POST` | `/_kdeps/reload` | ✓ | Reload the workflow from the current on-disk file |

## Authentication

Write endpoints (`PUT` and `POST`) require a **bearer token**. Set the token on the server by exporting `KDEPS_MANAGEMENT_TOKEN`:

```bash
export KDEPS_MANAGEMENT_TOKEN=mysecret
kdeps run workflow.yaml
```

Clients send the token in the `Authorization` header:

```
Authorization: Bearer mysecret
```

| Token state | Response |
|-------------|----------|
| `KDEPS_MANAGEMENT_TOKEN` unset | `503 Service Unavailable` |
| Token wrong or header missing | `401 Unauthorized` |
| Token correct | Handler runs |

`GET /_kdeps/status` is unauthenticated and always returns `200 OK`.

## Size Limits

| Endpoint | Limit | Over-limit response |
|----------|-------|---------------------|
| `PUT /_kdeps/workflow` | 5 MB | `413 Payload Too Large` |
| `PUT /_kdeps/package` | 200 MB | `413 Payload Too Large` |

Oversized uploads are rejected before any data is written to disk.

## Status Response

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

The `workflow` field is omitted when no workflow is loaded.

## Using `kdeps push`

The `kdeps push` command is the recommended way to call the management API. See the [`kdeps push` reference](../getting-started/cli-reference#kdeps-push) for details.

```bash
# Push a workflow directory
kdeps push ./my-agent http://container:16395

# Push a packaged .kdeps archive
kdeps push myagent-2.0.0.kdeps http://container:16395

# Explicit token (overrides KDEPS_MANAGEMENT_TOKEN)
kdeps push --token mysecret myagent-2.0.0.kdeps http://container:16395
```

## Direct curl Examples

```bash
# Check status (no auth required)
curl http://localhost:16395/_kdeps/status

# Push a workflow YAML
curl -X PUT \
  -H "Authorization: Bearer $KDEPS_MANAGEMENT_TOKEN" \
  -H "Content-Type: application/yaml" \
  --data-binary @workflow.yaml \
  http://localhost:16395/_kdeps/workflow

# Push a .kdeps package archive
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

## Restart Persistence

When a new workflow is pushed, kdeps writes it to the same path that was given at startup (or `/app/workflow.yaml` inside Docker). The workflow path is never changed after push — only the file contents are updated. On the next restart, kdeps reads the updated file automatically.

After a YAML push, any stale `.yaml`/`.yml` files in the `resources/` sibling directory are removed. This prevents duplicate resource loading because `kdeps push` inlines all resources into a single `workflow.yaml`.

Package pushes extract the full archive in-place — `resources/`, `data/`, and `scripts/` are all replaced or added.

## Security

- Path-traversal entries in `.kdeps` archives are rejected with `422 Unprocessable Entity`.
- Per-file decompression cap of 500 MB guards against zip-bomb payloads.
- Response bodies read by `kdeps push` are capped at 1 MB.
