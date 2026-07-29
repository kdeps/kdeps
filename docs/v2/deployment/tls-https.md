# TLS and HTTPS (Custom Domains)

Serve the kdeps **API server** and **web server** over HTTPS for a **custom domain**. Two modes:

1. **Static certificates** — you supply PEM `certFile` / `keyFile`
2. **Let's Encrypt** — kdeps obtains and renews certificates with ACME for your domain

Works in **workflow mode** (HTTP API / web) and for any agent package that enables `apiServer` / `webServer`. Agent mode REPL does not listen for HTTPS itself.

## Choose a mode

```d2
direction: down

A: DNS {shape: oval}
B: "kdeps host" {shape: rectangle}
C: "TLS mode" {shape: diamond}
D: StaticPEM {shape: rectangle}
E: LetsEncrypt {shape: rectangle}
F: ReverseProxy {shape: rectangle}
G: port80 {shape: rectangle}
H: port443 {shape: rectangle}

A -> B
B -> C
C -> D: certs
C -> E: acme
C -> F: proxy
E -> G
E -> H
```

| Mode | When to use |
|------|-------------|
| Static PEM | Corporate CA, existing cert pipeline, secrets already mounted |
| Let's Encrypt | Public custom domain, automatic renew, no PEM files to manage |
| Reverse proxy (Caddy/Traefik/Nginx) | Multi-service mesh, K8s Ingress — kdeps stays HTTP internally |

**Priority:** if both static PEM and `letsEncrypt` are set, **static PEM wins**.

## Static certificates

```yaml
# workflow.yaml
settings:
  certFile: "/etc/certs/server.crt"
  keyFile: "/etc/certs/server.key"
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Paths are top-level under `settings` (not under `apiServer`). Same certs apply to `webServer` when enabled.

## Let's Encrypt (custom domain)

kdeps uses Go `autocert` (ACME) against Let's Encrypt production or staging.

### Minimal config

```yaml
# workflow.yaml
settings:
  letsEncrypt:
    domain: api.example.com    # required primary hostname
    email: ops@example.com     # recommended for LE expiry notices
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

### Full field reference

```yaml
settings:
  letsEncrypt:
    domain: api.example.com           # primary host (required unless domains[] only)
    domains:                          # optional additional SANs
      - www.example.com
      - app.example.com
    email: ops@example.com            # ACME account contact
    cacheDir: /var/lib/kdeps/letsencrypt   # default: ~/.kdeps/letsencrypt
    staging: false                    # true = staging CA (untrusted by browsers; no prod rate limits)
    # httpChallengeAddr: ":80"        # default; set "" to disable HTTP-01 listener
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
  # webServer also uses the same TLS settings when present
  # webServer:
  #   hostIp: "0.0.0.0"
  #   portNum: 443
```

| Field | Default | Purpose |
|-------|---------|---------|
| `domain` | — | Primary hostname on the cert |
| `domains` | `[]` | Extra hostnames (SANs); if only these are set, first becomes primary |
| `email` | `""` | Let's Encrypt registration email |
| `cacheDir` | `~/.kdeps/letsencrypt` | ACME account keys + issued certs (must be writable) |
| `staging` | `false` | Use LE staging directory |
| `httpChallengeAddr` | `":80"` | Bind address for HTTP-01; empty string disables it |

### DNS and ports

1. Create **A/AAAA** records for every hostname in `domain` / `domains` pointing at this machine (or load balancer that forwards 80/443).
2. Open **TCP 80** (HTTP-01 challenge) and **TCP 443** (HTTPS app + TLS-ALPN-01).
3. Prefer listening on **port 443** for production HTTPS.
4. Ensure the process can **write** `cacheDir` (persist this volume in Docker/K8s so renewals keep working).

### What kdeps starts

When `letsEncrypt` is active (and no static PEM):

- **HTTPS** on the configured API/web listen address (`ListenAndServeTLS` with autocert `GetCertificate`)
- **HTTP on `httpChallengeAddr`** (default `:80`) serving `/.well-known/acme-challenge/…`; other paths return a short plain-text hint (no open redirect)

Certificates renew automatically while the process is running and the cache directory is preserved.

### Staging checklist

Before production, validate the path with staging:

```yaml
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
    staging: true
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
```

Browsers will warn (staging CA is untrusted). When green, set `staging: false` and clear or use a separate `cacheDir` so production does not reuse staging material.

## Docker

Publish 80 and 443; mount a volume for the LE cache:

```bash
kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest

docker run --rm -p 80:80 -p 443:443 \
  -v kdeps-le:/var/lib/kdeps/letsencrypt \
  -e KDEPS_API_AUTH_TOKEN=... \
  myregistry/myagent:latest
```

```yaml
# workflow.yaml (inside the package)
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
    cacheDir: /var/lib/kdeps/letsencrypt
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
```

Alternatively mount static secrets as `certFile` / `keyFile` (see [Docker reference](/reference/docker-reference)).

## Kubernetes

Expose Service ports **80** and **443**, use a **persistent volume** for `cacheDir`, and set `hostIP`/`port` so the pod listens on 443.

```yaml
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
    cacheDir: /var/lib/kdeps/letsencrypt
  apiServer:
    hostIP: "0.0.0.0"
    portNum: 443
```

Many clusters prefer **Ingress TLS** (cert-manager) and keep kdeps on plain HTTP inside the cluster. That remains fully supported — omit `letsEncrypt` and terminate TLS at the Ingress.

```bash
kdeps export k8s myagent-1.0.0.kdeps --image REG/myagent:1 -o k8s.yaml
# Edit Service/Ingress to publish 80/443 and attach a PVC to cacheDir when using in-pod LE
kubectl apply -f k8s.yaml
```

## Reverse proxy (optional)

If you already run Caddy, Traefik, or Nginx for TLS:

- Leave kdeps on HTTP (`hostIP: 127.0.0.1`, default port)
- Do **not** set `letsEncrypt` or PEM paths on kdeps
- Proxy `https://api.example.com` → `http://127.0.0.1:16395`

## LLM server appliances

`kdeps llm` appliances expose OpenAI-compatible `/v1`. For a custom domain HTTPS front-end, either:

- Put **Caddy/Traefik/Nginx** with Let's Encrypt in front of the appliance port (8000), or
- Run a kdeps **agent/API** package with `letsEncrypt` that proxies to the appliance

The appliance recipes themselves do not embed ACME today.

## Troubleshooting

| Symptom | Check |
|---------|--------|
| Challenge fails | DNS points here? Port 80 open to the internet? |
| Rate limited by LE | Use `staging: true` while iterating; then production with clean cache |
| Permission denied writing certs | `cacheDir` writable by the kdeps user |
| Wrong certificate host | Hostname in request must match `domain` / `domains` whitelist |
| Still HTTP | Confirm no static empty cert paths; look for log `starting HTTPS server` / `letsencrypt:` |

## Related

- [Security reference — TLS](/reference/security#tls)
- [Workflow configuration](/configuration/workflow)
- [Docker deployment](/deployment/docker)
- [Kubernetes deployment](/deployment/kubernetes)
- [Docker reference](/reference/docker-reference)
