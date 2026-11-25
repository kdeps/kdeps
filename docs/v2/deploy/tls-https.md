# TLS and HTTPS (custom domains)

kdeps serves the API server and web server over HTTPS for a custom domain, with your certificate files.

```yaml
# workflow.yaml
deploy:
  certFile: "/etc/certs/server.crt"   # static certificates mode
  keyFile: "/etc/certs/server.key"
  apiServer:
    portNum: 443
```

## Read next

- [TLS and HTTPS (custom domains) reference](/reference/deploy/tls-https) - every key, option and detail
- [Workflow configuration](/workflow/configuration)
- [Docker deployment](/deploy/docker)
