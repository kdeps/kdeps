# Docker Deployment Reference

Production best practices, troubleshooting, and security hardening for kdeps Docker deployments. See [Docker Deployment](/deployment/docker) for the core packaging and build workflow.

## Production Best Practices

### Use Specific Tags

```bash
# Good
kdeps bundle build app.kdeps --tag myregistry/myagent:1.0.0

# Avoid
kdeps bundle build app.kdeps --tag myregistry/myagent:latest
```

### Set Resource Limits

```yaml
# docker-compose.yml
services:
  myagent:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

### Use Secrets

```bash
# Create secret
echo "my-api-key" | docker secret create api_key -

# Use in container
docker service create \
  --secret api_key \
  myregistry/myagent:latest
```

### Enable Logging

```yaml
# docker-compose.yml
services:
  myagent:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
```

### Network Security

```yaml
# docker-compose.yml
services:
  myagent:
    networks:
      - internal
    ports:
      - "127.0.0.1:16395:16395"  # Only local access

networks:
  internal:
    internal: true
```

## Security Hardening

Generated images run as the unprivileged `kdeps` user, including Ollama-backed images. Ollama models are stored under `/app/.ollama/models`.

Before exposing a container externally, add these fields to `workflow.yaml`:

```yaml
# workflow.yaml
settings:
  certFile: "/run/secrets/server.crt"  # mount cert into container; enables HTTPS
  keyFile:  "/run/secrets/server.key"
  apiServer:
    # auth token (required): KDEPS_API_AUTH_TOKEN env var or api_auth_token in ~/.kdeps/config.yaml
    rateLimit:
      requestsPerMinute: 60            # sustained per-IP rate
      burst: 10                        # burst allowance above the sustained rate
    maxBodyBytes: 1048576              # 1 MB cap on request body size
    maxConcurrent: 50                  # excess requests get 503 immediately
```

See [Security](../configuration/advanced#security) for the full reference.

## Troubleshooting

### Build Fails

```bash
# Show detailed output
kdeps bundle build app.kdeps --show-dockerfile

# Check Docker daemon
docker info
```

### Image Too Large

1. Use `alpine` base OS
2. Remove unnecessary packages
3. Use optimized templates (automatic)
4. Avoid `offlineMode` unless needed

### Model Download Slow

```bash
# Pre-pull models before build
ollama pull llama3.2:1b

# Or use offline mode
offlineMode: true
```

### Check Workflow Status

```bash
curl http://localhost:16395/_kdeps/status
```

```json
{
  "status": "ok",
  "workflow": {
    "name": "my-agent",
    "version": "2.0.0",
    "description": "My AI agent",
    "resources": 3
  }
}
```

### Docker Compose with Management API

```yaml
# docker-compose.yml
services:
  myagent:
    image: myregistry/myagent:latest
    ports:
      - "16395:16395"
    environment:
      - KDEPS_MANAGEMENT_TOKEN=${KDEPS_MANAGEMENT_TOKEN}
    restart: unless-stopped
```

Set the token in your `.env` file (never commit this file):

```bash
# .env
KDEPS_MANAGEMENT_TOKEN=mysecret
```

For the full management API reference see [Management API](/reference/management-api).

## See Also

- [Docker Deployment](/deployment/docker) - Core packaging and build workflow
- [Kubernetes Deployment](/deployment/kubernetes) - Cluster deployment
- [Management API](/reference/management-api) - Live workflow updates without rebuilding
