# TLS and HTTPS (custom domains)

Serve the kdeps API server and web server over HTTPS for a custom domain. There are two modes:

1. **Static certificates** - you supply PEM `certFile` and `keyFile`.
2. **Let's Encrypt** - kdeps gets and renews certificates with ACME for your domain.

```yaml
# workflow.yaml
deploy:
  certFile: "/etc/certs/server.crt"   # static certificates mode
  keyFile: "/etc/certs/server.key"
  apiServer:
    portNum: 443
```

## Read next

- [TLS and HTTPS (custom domains) reference](/reference/deploy/tls-https) - choose a mode, static certificates, let's encrypt (custom domain), minimal config, full field reference, DNS and ports, what kdeps starts, staging checklist
- [Security reference - TLS](/reference/security#tls)
- [Workflow configuration](/workflow/configuration)
- [Docker deployment](/deploy/docker)
- [Kubernetes deployment](/deploy/kubernetes)
- [Docker reference](/reference/deploy/docker-reference)
