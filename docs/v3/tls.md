# TLS / HTTPS

Serve the API or web server over HTTPS with a custom domain. Workflow mode only (packaged HTTP servers). The agent REPL does not terminate TLS itself.

## Pick an approach

| Mode | Use when |
|------|----------|
| Static PEM | You already have certs |
| Let's Encrypt | Public domain, auto renew |
| Reverse proxy | Caddy / Traefik / Ingress in front; kdeps stays HTTP |

If both PEM paths and `letsEncrypt` are set, **static PEM wins**.

## Static certificates

```yaml
settings:
  certFile: "/etc/certs/server.crt"
  keyFile: "/etc/certs/server.key"
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 443
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Paths are top-level under `settings` so API and web share the same certs.

## Let's Encrypt

```yaml
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
    # domains: [www.example.com]   # optional SANs
    # staging: true                # untrusted CA; good for dry runs
    # cacheDir: /var/lib/kdeps/letsencrypt
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 443
```

Requirements:

- DNS for the domain points at this host
- Port **80** reachable for HTTP-01 (default challenge)
- Port **443** for HTTPS traffic

## Reverse proxy

Terminate TLS at the proxy. Keep kdeps on plain HTTP inside the network. Set `trustedProxies` so client IPs and rate limits stay honest — see [Security](/security).
