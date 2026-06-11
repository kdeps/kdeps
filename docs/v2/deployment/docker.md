# Docker Deployment

`kdeps bundle build` packages your workflow into a Docker image that starts an API server when run. No Dockerfile needed -- kdeps generates one from your `workflow.yaml`.

## Overview

```bash
# Package workflow into .kdeps file
kdeps bundle package workflow.yaml

# Build Docker image
kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest

# Or with GPU support
kdeps bundle build myagent-1.0.0.kdeps --gpu cuda --tag myregistry/myagent:latest-gpu
```

## Packaging

The `package` command creates a `.kdeps` archive containing your workflow and resources:

```bash
kdeps bundle package path/to/workflow.yaml
```

This creates `myagent-1.0.0.kdeps` (name and version from workflow metadata).

### What's Included

```
myagent-1.0.0.kdeps
├── workflow.yaml          # workflow entry point
├── resources/             # all resource YAML files
├── data/                  # data files and scripts
├── requirements.txt       # Python dependencies (if present)
└── public/                # static files (if present)
```

## Building Docker Images

### Basic Build

```bash
kdeps bundle build myagent-1.0.0.kdeps
```

Creates image: `kdeps-myagent:1.0.0`

### Custom Tag

```bash
kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest
```

### Show Dockerfile

View the generated Dockerfile without building:

```bash
kdeps bundle build myagent-1.0.0.kdeps --show-dockerfile
```

## GPU Support

Build images with GPU acceleration:

```bash
# NVIDIA CUDA
kdeps bundle build myagent-1.0.0.kdeps --gpu cuda

# AMD ROCm
kdeps bundle build myagent-1.0.0.kdeps --gpu rocm

# Intel oneAPI
kdeps bundle build myagent-1.0.0.kdeps --gpu intel

# Vulkan (cross-platform)
kdeps bundle build myagent-1.0.0.kdeps --gpu vulkan
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
kdeps bundle build myagent-1.0.0.kdeps

# GPU: Uses Ubuntu (required for GPU drivers)
kdeps bundle build myagent-1.0.0.kdeps --gpu cuda
```

### Why Auto-Selection?

- **Alpine** uses musl libc and cannot run GPU workloads (NVIDIA CUDA, AMD ROCm require glibc)
- **Ubuntu** uses glibc and supports all GPU types
- Auto-selection prevents invalid combinations (e.g., Alpine + CUDA)

| Configuration | Base OS | Image Size | Use Case |
|---------------|---------|------------|----------|
| `kdeps bundle build .` | **Alpine** | ~300MB | CPU-only, edge deployment |
| `kdeps bundle build . --gpu cuda` | **Ubuntu** | ~800MB+ | NVIDIA GPU inference |
| `kdeps bundle build . --gpu rocm` | **Ubuntu** | ~800MB+ | AMD GPU inference |
| `kdeps bundle build . --gpu intel` | **Ubuntu** | ~600MB+ | Intel GPU inference |
| `kdeps bundle build . --gpu vulkan` | **Ubuntu** | ~600MB+ | Cross-platform GPU |

## Offline Mode

Bake models into the image for air-gapped deployments:

```yaml
# workflow.yaml
settings:
  agentSettings:
    offlineMode: true
    models:
      - llama3.2:1b
      - llama3.2-vision
```

Build with models included:

```bash
kdeps bundle build myagent-1.0.0.kdeps
```

The resulting image contains all models and doesn't require internet access.

## Python Dependencies

### Using requirements.txt

```yaml
# workflow.yaml
settings:
  agentSettings:
    requirementsFile: "requirements.txt"
```

KDeps uses [uv](https://github.com/astral-sh/uv) for fast Python package management (97% smaller than Anaconda).

### Inline Packages

```yaml
# workflow.yaml
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
# workflow.yaml
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
# workflow.yaml
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
# workflow.yaml
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
# docker-compose.yml
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

# Upgrade base packages (security patches), then install dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache curl bash python3 py3-pip

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
# workflow.yaml
settings:
  apiServer:
    routes:
      - path: /health
        methods: [GET]
```

In Docker Compose:
```yaml
# docker-compose.yml
services:
  myagent:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:16395/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Kubernetes Deployment

KDeps generates Kubernetes manifests directly from your `workflow.yaml` using `kdeps export k8s`. No manual YAML authoring needed.

```bash
# Build and push the Docker image first
kdeps bundle build . --tag myregistry/myagent:1.0.0
docker push myregistry/myagent:1.0.0

# Generate manifests
kdeps export k8s . \
  --image myregistry/myagent:1.0.0 \
  --output k8s.yaml

# Apply to cluster
kubectl apply -f k8s.yaml
```

Configure Kubernetes settings in `workflow.yaml`:

```yaml
# workflow.yaml
settings:
  portNum: 16395
  agentSettings:
    replicas: 3
    resources:
      cpuLimit: "2000m"
      memoryLimit: "4Gi"
      cpuRequest: "500m"
      memoryRequest: "1Gi"
    env:
      LOG_LEVEL: info
```

The generated manifest includes a `Deployment` with readiness/liveness probes and a `ClusterIP` `Service`, both derived from your workflow settings.

See the [Kubernetes Deployment guide](kubernetes) for the full reference.

## See Also

- [Docker Reference](/reference/docker-reference) - Production best practices, security hardening, troubleshooting
- [Workflow Configuration](../configuration/workflow) - Agent settings
- [WebServer Mode](webserver) - Serve frontends
- [LLM Backends](../resources/llm-backends) - Backend configuration
- [Management API](/reference/management-api) - Live workflow updates without rebuilding
