# Security

Requests hit a short gate before the workflow DAG.

```text
request -> auth -> rate limit -> body size -> concurrency -> DAG
```

## API authentication

When `apiServer` is enabled, kdeps **will not start** without a token.

```bash
export KDEPS_API_AUTH_TOKEN="your-secret-token"
# or in ~/.kdeps/config.yaml:
# api_auth_token: "your-secret-token"
```

Clients send:

```http
Authorization: Bearer <token>
# or
X-Api-Key: <token>
```

Mark individual routes public when needed (e.g. health-style or browser-facing paths that cannot hold a header):

```yaml
routes:
  - path: /api/v1/chat
    methods: [POST]
  - path: /api/v1/open
    methods: [GET]
    public: true
```

Never put the token in `workflow.yaml`.

## Trusted proxies

Only honor `X-Forwarded-For` / `X-Real-IP` from known hops:

```yaml
settings:
  apiServer:
    trustedProxies:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
```

## Rate limits and body size

Configure under `apiServer` (names vary slightly by version — check `kdeps validate` and `--help` output). Typical knobs: requests per minute, burst, max body bytes, max concurrent in-flight requests.

## Secrets

| Keep in git | Keep local / secret store |
|-------------|---------------------------|
| workflow.yaml, resources | API tokens |
| non-secret defaults | Provider API keys |
| route shapes | DB passwords, SMTP |

## Management endpoints

`/_kdeps/*` (when enabled) uses a **management** token, not the API token. Treat it as admin access.

## TLS

See [TLS / HTTPS](/tls). Prefer HTTPS on public edges; keep internal mesh simple if a proxy already terminates TLS.
