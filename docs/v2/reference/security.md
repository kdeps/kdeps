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

When `apiServer` is configured, authentication is required. Every request must include the token via `Authorization: Bearer <token>` or `X-Api-Key: <token>`. The `/health` endpoint is always exempt. kdeps refuses to start the API server without a token.

Set the token in `~/.kdeps/config.yaml` (machine-local, never committed):

```yaml
# ~/.kdeps/config.yaml
api_auth_token: "your-secret-token"
```

Or via environment variable:

```bash
export KDEPS_API_AUTH_TOKEN="your-secret-token"
```

No `auth:` block in `workflow.yaml`. The token is never stored in the workflow file.

## Trusted Proxies

`X-Forwarded-For` and `X-Real-IP` are ignored unless the direct TCP peer matches an entry in `trustedProxies`. This prevents clients from spoofing their IP for rate limiting and request context. Configure CIDRs or exact IPs for your load balancer or ingress.

```yaml
# workflow.yaml
settings:
  apiServer:
    trustedProxies:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
```

Without `trustedProxies`, kdeps uses `RemoteAddr` only. When both `apiServer` and `webServer` are configured, entries from both blocks are merged for rate limiting and request IP context.

## Security Headers

Both `apiServer` and `webServer` responses include defensive HTTP headers on every response:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains` (TLS only)

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
