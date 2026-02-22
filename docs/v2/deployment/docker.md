# Docker Deployment

KDeps can package your AI agent into optimized Docker images for production deployment.

## Overview

```bash
# Package workflow into .kdeps file
kdeps package workflow.yaml

# Build Docker image
kdeps build myagent-1.0.0.kdeps --tag myregistry/myagent:latest

# Or with GPU support
kdeps build myagent-1.0.0.kdeps --gpu cuda --tag myregistry/myagent:latest-gpu
```

## Packaging

The `package` command creates a `.kdeps` archive containing your workflow and resources:

```bash
kdeps package path/to/workflow.yaml
```

This creates `myagent-1.0.0.kdeps` (name and version from workflow metadata).

### What's Included

- `workflow.yaml` - Workflow configuration
- `resources/` - All resource YAML files
- `data/` - Data files and scripts
- `requirements.txt` - Python dependencies (if present)
- `public/` - Static files (if present)

## Building Docker Images

### Basic Build

```bash
kdeps build myagent-1.0.0.kdeps
```

Creates image: `kdeps-myagent:1.0.0`

### Custom Tag

```bash
kdeps build myagent-1.0.0.kdeps --tag myregistry/myagent:latest
```

### Show Dockerfile

View the generated Dockerfile without building:

```bash
kdeps build myagent-1.0.0.kdeps --show-dockerfile
```

## GPU Support

Build images with GPU acceleration:

```bash
# NVIDIA CUDA
kdeps build myagent-1.0.0.kdeps --gpu cuda

# AMD ROCm
kdeps build myagent-1.0.0.kdeps --gpu rocm

# Intel oneAPI
kdeps build myagent-1.0.0.kdeps --gpu intel

# Vulkan (cross-platform)
kdeps build myagent-1.0.0.kdeps --gpu vulkan
```

### GPU Runtime

When running GPU-enabled images:

```bash
# NVIDIA
docker run --gpus all myregistry/myagent:latest

# AMD
docker run --device=/dev/kfd --device=/dev/dri myregistry/myagent:latest
```

## Base OS Auto-Selection

KDeps automatically selects the base OS based on GPU requirements:

- **No `--gpu` flag** → **Alpine** (CPU-only, smallest images ~300MB)
- **`--gpu` specified** → **Ubuntu** (GPU support, glibc-based)

The OS is automatically chosen to ensure compatibility:

```bash
# CPU-only: Uses Alpine (smallest)
kdeps build myagent-1.0.0.kdeps

# GPU: Uses Ubuntu (required for GPU drivers)
kdeps build myagent-1.0.0.kdeps --gpu cuda
```

### Why Auto-Selection?

- **Alpine** uses musl libc and cannot run GPU workloads (NVIDIA CUDA, AMD ROCm require glibc)
- **Ubuntu** uses glibc and supports all GPU types
- Auto-selection prevents invalid combinations (e.g., Alpine + CUDA)

| Configuration | Base OS | Image Size | Use Case |
|---------------|---------|------------|----------|
| `kdeps build .` | **Alpine** | ~300MB | CPU-only, edge deployment |
| `kdeps build . --gpu cuda` | **Ubuntu** | ~800MB+ | NVIDIA GPU inference |
| `kdeps build . --gpu rocm` | **Ubuntu** | ~800MB+ | AMD GPU inference |
| `kdeps build . --gpu intel` | **Ubuntu** | ~600MB+ | Intel GPU inference |
| `kdeps build . --gpu vulkan` | **Ubuntu** | ~600MB+ | Cross-platform GPU |

## Offline Mode

Bake models into the image for air-gapped deployments:

```yaml
settings:
  agentSettings:
    offlineMode: true
    models:
      - llama3.2:1b
      - llama3.2-vision
```

Build with models included:

```bash
kdeps build myagent-1.0.0.kdeps
```

The resulting image contains all models and doesn't require internet access.

## Python Dependencies

### Using requirements.txt

```yaml
settings:
  agentSettings:
    requirementsFile: "requirements.txt"
```

KDeps uses [uv](https://github.com/astral-sh/uv) for fast Python package management (97% smaller than Anaconda).

### Inline Packages

```yaml
settings:
  agentSettings:
    pythonVersion: "3.12"
    pythonPackages:
      - pandas>=2.0
      - numpy
      - scikit-learn
```

## System Packages

Install OS-level packages:

```yaml
settings:
  agentSettings:
    osPackages:
      - ffmpeg
      - imagemagick
      - tesseract-ocr
      - poppler-utils
    repositories:
      - ppa:alex-p/tesseract-ocr-devel
```

## Environment Variables

### Build-time Args

```yaml
settings:
  agentSettings:
    args:
      BUILD_VERSION: ""
```

Pass during build:
```bash
docker build --build-arg BUILD_VERSION=1.0.0 ...
```

### Runtime Environment

```yaml
settings:
  agentSettings:
    env:
      LOG_LEVEL: "info"
      API_TIMEOUT: "30"
```

Override at runtime:
```bash
docker run -e LOG_LEVEL=debug myregistry/myagent:latest
```

## Docker Compose

KDeps generates a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  myagent:
    image: kdeps-myagent:1.0.0
    ports:
      - "16395:16395"      # API server
      - "16395:16395"      # Web server (if enabled)
    environment:
      - LOG_LEVEL=info
    volumes:
      - ollama:/root/.ollama
      - kdeps_data:/agent/volume
    restart: on-failure
    # For GPU:
    # deploy:
    #   resources:
    #     reservations:
    #       devices:
    #         - driver: nvidia
    #           count: 1
    #           capabilities: [gpu]

volumes:
  ollama:
  kdeps_data:
```

Run with:
```bash
docker-compose up -d
```

## Optimized Build Process

KDeps uses a streamlined build process that leverages the official installation script. This ensures the smallest possible image size and maximum compatibility.

```dockerfile
# Example of generated Dockerfile logic
FROM alpine:3.18

# Install dependencies
RUN apk add --no-cache curl bash python3 py3-pip

# Install kdeps via official install script
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh -s -- -b /usr/local/bin

# Copy agent files
COPY workflow.yaml /app/workflow.yaml
COPY resources/ /app/resources/

WORKDIR /app
ENTRYPOINT ["kdeps"]
CMD ["run", "workflow.yaml"]
```

The build process also automatically handles:
- **Python environments**: Integrated `uv` for 97% smaller virtual environments.
- **Model management**: Pre-pulling models for offline readiness.
- **Service orchestration**: Lightweight `supervisor` to manage API and LLM processes.

## Health Checks

Add a health endpoint:

```yaml
settings:
  apiServer:
    routes:
      - path: /health
        methods: [GET]
```

In Docker Compose:
```yaml
services:
  myagent:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:16395/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Production Best Practices

### 1. Use Specific Tags

```bash
# Good
kdeps build app.kdeps --tag myregistry/myagent:1.0.0

# Avoid
kdeps build app.kdeps --tag myregistry/myagent:latest
```

### 2. Set Resource Limits

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

### 3. Use Secrets

```bash
# Create secret
echo "my-api-key" | docker secret create api_key -

# Use in container
docker service create \
  --secret api_key \
  myregistry/myagent:latest
```

### 4. Enable Logging

```yaml
services:
  myagent:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
```

### 5. Network Security

```yaml
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

## Kubernetes Deployment

Example Kubernetes manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myagent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myagent
  template:
    metadata:
      labels:
        app: myagent
    spec:
      containers:
      - name: myagent
        image: myregistry/myagent:1.0.0
        ports:
        - containerPort: 16395
        env:
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 16395
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 16395
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: myagent
spec:
  selector:
    app: myagent
  ports:
  - port: 80
    targetPort: 16395
```

## Troubleshooting

### Build Fails

```bash
# Show detailed output
kdeps build app.kdeps --show-dockerfile

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

## Live Workflow Updates (Management API)

Every running kdeps container exposes a built-in management API that lets you update the workflow without rebuilding or redeploying the image.

### Enable the Management API

Set `KDEPS_MANAGEMENT_TOKEN` when starting the container:

```bash
docker run -e KDEPS_MANAGEMENT_TOKEN=mysecret -p 16395:16395 myregistry/myagent:latest
```

### Push a New Workflow

```bash
# Push a local workflow directory
kdeps push --token mysecret ./my-agent http://localhost:16395

# Push a packaged .kdeps archive
kdeps push --token mysecret myagent-2.0.0.kdeps http://localhost:16395
```

The token can also be supplied via the environment so it is not in shell history:

```bash
export KDEPS_MANAGEMENT_TOKEN=mysecret
kdeps push ./my-agent http://localhost:16395
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

For the full management API reference see [Management API](../concepts/management-api).

## Next Steps

- [Workflow Configuration](../configuration/workflow) - Agent settings
- [WebServer Mode](webserver) - Serve frontends
- [LLM Backends](../resources/llm-backends) - Backend configuration
- [Management API](../concepts/management-api) - Live workflow updates without rebuilding
