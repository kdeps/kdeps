# Security Reference

Every request passes through a chain of gates before reaching the workflow DAG. Each gate can reject the request with a specific status code.

```d2
direction: down

A: incoming request {shape: oval}
B: "auth check\n401 if Bearer / X-Api-Key wrong or missing"
C: "rate limit\n429 if over requestsPerMinute + burst"
D: "body size check\n413 if body exceeds maxBodyBytes"
E: "concurrency cap\n503 if over maxConcurrent in-flight requests"
F: workflow DAG {shape: oval}

A -> B -> C -> D -> E -> F
```

## Authentication

Protect the API server with a shared secret. When `auth.token` is set, every request must include it via `Authorization: Bearer <token>` or `X-Api-Key: <token>`. The `/health` endpoint is always exempt.

```yaml
# workflow.yaml
settings:
  apiServer:
    auth:
      token: "${API_TOKEN}"
```

Omit `auth` (or leave `token` empty) to disable authentication entirely.

## Rate Limiting

Limit requests per client IP using a token-bucket algorithm. `requestsPerMinute` is the sustained rate; `burst` is the number of requests allowed above that rate in a single burst. Clients that exceed the limit receive a `429` response with a `Retry-After: 60` header.

```yaml
# workflow.yaml
settings:
  apiServer:
    rateLimit:
      requestsPerMinute: 60
      burst: 10
```

## Body Size Limit

Cap the size of incoming request bodies. Requests that exceed `maxBodyBytes` receive a `413` response. This limit does not apply to `multipart/form-data` uploads.

```yaml
# workflow.yaml
settings:
  apiServer:
    maxBodyBytes: 1048576   # 1 MiB
```

## TLS

Enable HTTPS by pointing `certFile` and `keyFile` at a PEM certificate and private key. These fields belong in `settings`, not in `apiServer`.

```yaml
# workflow.yaml
settings:
  certFile: "/etc/certs/server.crt"
  keyFile:  "/etc/certs/server.key"
  apiServer:
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

## Concurrent Request Limit

Cap the number of simultaneous in-flight requests. When the limit is reached, new requests receive `503 Service Unavailable` immediately rather than queuing. Omit or set to `0` to disable.

```yaml
# workflow.yaml
settings:
  apiServer:
    maxConcurrent: 50
```

## Resource Output Caps

Four environment variables limit how many bytes executor resources return to the workflow engine. Set them in `agentSettings.env`.

| Variable | Applies to |
|---|---|
| `KDEPS_EXEC_MAX_OUTPUT_BYTES` | Shell / exec resource stdout |
| `KDEPS_HTTP_MAX_RESPONSE_BYTES` | HTTP resource response body |
| `KDEPS_CHAT_MAX_OUTPUT_BYTES` | LLM chat response content |
| `KDEPS_PYTHON_MAX_OUTPUT_BYTES` | Python resource stdout |

```yaml
# workflow.yaml
settings:
  agentSettings:
    env:
      KDEPS_EXEC_MAX_OUTPUT_BYTES: "524288"    # 512 KiB
      KDEPS_HTTP_MAX_RESPONSE_BYTES: "1048576" # 1 MiB
      KDEPS_CHAT_MAX_OUTPUT_BYTES: "1048576"   # 1 MiB
      KDEPS_PYTHON_MAX_OUTPUT_BYTES: "524288"  # 512 KiB
```

## See Also

- [Advanced Configuration](/configuration/advanced) - Request object, agent settings, SQL connections, trusted proxies
- [Docker Reference](/reference/docker-reference) - Container security hardening
- [Validation and Control Flow](/concepts/validation-and-control) - Per-resource access control
