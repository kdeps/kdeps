# Docker deployment

`kdeps bundle build` packages your workflow into a Docker image that starts an API server when run. No Dockerfile needed - kdeps generates one from your `workflow.yaml`.

*Applies to workflow mode.*

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

### What's included

```
myagent-1.0.0.kdeps
├── workflow.yaml          # workflow entry point
├── resources/             # all resource YAML files
├── data/                  # data files and scripts
├── requirements.txt       # Python dependencies (if present)
└── public/                # static files (if present)
```

## Building Docker images

### Basic build

```bash
kdeps bundle build myagent-1.0.0.kdeps
```

Creates image: `kdeps-myagent:1.0.0`

### Custom tag

```bash
kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest
```

### Show dockerfile

View the generated Dockerfile without building:

```bash
kdeps bundle build myagent-1.0.0.kdeps --show-dockerfile
```

## GPU support

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

### GPU runtime

When running GPU-enabled images:

```bash
# NVIDIA
docker run --gpus all myregistry/myagent:latest

# AMD
docker run --device=/dev/kfd --device=/dev/dri myregistry/myagent:latest
```

## Base OS auto-selection

kdeps automatically selects the base OS based on GPU requirements:

- **No `--gpu` flag** → **Alpine** (CPU-only, smallest images ~300MB)
- **`--gpu` specified** → **Ubuntu** (GPU support, glibc-based)

The OS is automatically chosen to ensure compatibility:

```bash
# CPU-only: Uses Alpine (smallest)
kdeps bundle build myagent-1.0.0.kdeps

# GPU: Uses Ubuntu (required for GPU drivers)
kdeps bundle build myagent-1.0.0.kdeps --gpu cuda
```

### Why auto-selection?

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

Override the auto-selected distro in `workflow.yaml`:

```yaml
settings:
  agentSettings:
    baseOS: ubuntu  # alpine (default) or ubuntu
```

`--gpu` always forces Ubuntu. `debian` is not supported.

## LLM backend in images

By default no LLM server is installed: chat resources run on the `file`
backend, and the llamafile model for each chat resource is downloaded into the
image at build time (`/app/.kdeps/models/`). The container is a vanilla
`alpine:latest` or `ubuntu:latest` base plus kdeps and the baked models -
the llamafile self-serves inside the container, no extra process to manage.

```text
default (file backend)     → FROM alpine:latest or ubuntu:latest
                             + pre-baked llamafile per chat model (~1.1 GB for llama3.2:1b)
no chat resources          → FROM alpine:latest or ubuntu:latest (vanilla)
```

## Ollama Docker images (opt-in)

Ollama is bundled only when `installOllama: true`, `KDEPS_DEFAULT_BACKEND=ollama` is set with chat resources, or `KDEPS_LLM_ROUTER` routes to Ollama.

```text
needs Ollama + alpine CPU  → FROM alpine/ollama:<tag>   (~70MB third-party CPU image)
needs Ollama + ubuntu CPU  → FROM ollama/ollama:<tag>   (official image)
needs Ollama + --gpu cuda  → FROM ollama/ollama:<tag>   (runtime: docker run --gpus all)
needs Ollama + --gpu rocm  → FROM ollama/ollama:rocm    (runtime: --device /dev/kfd --device /dev/dri)
no Ollama                  → FROM alpine:latest or ubuntu:latest
```

| Image | Size | GPU | Source |
|-------|------|-----|--------|
| `alpine/ollama` | ~70MB | CPU only | [alpine-docker/ollama](https://github.com/alpine-docker/ollama) |
| `ollama/ollama` | ~4GB | CPU + NVIDIA | [Official Ollama Docker](https://hub.docker.com/r/ollama/ollama) |
| `ollama/ollama:rocm` | varies | AMD | Official `:rocm` variant |

When the base image already includes Ollama, kdeps does not `COPY --from` a second Ollama layer.

## Offline mode

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

## Python dependencies

### Using requirements.txt

```yaml
# workflow.yaml
settings:
  agentSettings:
    requirementsFile: "requirements.txt"
```

kdeps uses [uv](https://github.com/astral-sh/uv) for fast Python package management (97% smaller than Anaconda).

### Inline packages

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

## System packages

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

## Package version pinning

On every `bundle build`, kdeps resolves package versions before generating the Dockerfile:

- **kdeps** - latest [GitHub release](https://github.com/kdeps/kdeps/releases); `install.sh` is fetched from that tag and the same tag is passed to the installer (never `main` + floating latest)
- **ollama** - latest [ollama/ollama](https://github.com/ollama/ollama/releases) release as the Docker image tag for `ollama/ollama` and `alpine/ollama` (never `:latest`). `--gpu rocm` uses the fixed `ollama/ollama:rocm` tag instead.
- **uv** - latest [astral-sh/uv](https://github.com/astral-sh/uv/releases) release as the `ghcr.io/astral-sh/uv` tag

Override any field with an explicit semver (`v1.2.3` or `1.2.3`). Use `latest` or omit a field to accept the resolved value at build time.

```yaml
# workflow.yaml
settings:
  agentSettings:
    versions:
      kdeps: v2.0.0    # optional - default: newest GitHub release when bundle build runs
      ollama: 0.5.4    # optional - default: newest ollama/ollama release
      uv: 0.6.3        # optional - default: newest astral-sh/uv release
```

Preview resolved pins:

```bash
kdeps bundle build myagent-1.0.0.kdeps --show-dockerfile
```

When no Ollama base image is selected, kdeps uses `alpine:latest` or `ubuntu:latest`. Python defaults to `3.12` when `pythonVersion` is omitted.

## Environment variables

### Build-time args

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

### Runtime environment

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

## Docker compose

kdeps generates a `docker-compose.yml`:

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

## Optimized build process

kdeps uses a streamlined build process that leverages the official installation script. This ensures the smallest possible image size and maximum compatibility.

```dockerfile
# Example of generated Dockerfile logic
FROM alpine:3.18

# Upgrade base packages (security patches), then install dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache curl bash python3 py3-pip

# Install kdeps via official install script. Released CLIs pin both the
# script ref and the binary tag to their own version; dev builds use main.
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/v2.1.0/install.sh | sh -s -- -b /usr/local/bin v2.1.0

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

## Health checks

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

## Kubernetes deployment

kdeps generates Kubernetes manifests directly from your `workflow.yaml` using `kdeps export k8s`. No manual YAML authoring needed.

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
  apiServer:
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

## See also

- [Docker Reference](/reference/docker-reference) - Production best practices, security hardening, troubleshooting
- [Workflow Configuration](../configuration/workflow) - Agent settings
- [WebServer Mode](webserver) - Serve frontends
- [LLM Backends](../resources/llm-backends) - Backend configuration
- [Management API](/reference/management-api) - Live workflow updates without rebuilding

## LLM server appliance (not an agent image)

This page packages **agents** (`workflow.yaml` + kdeps). For a **standalone** OpenAI-compatible model server:

```bash
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --tag myorg/vllm:1
```

See [LLM Server Appliance](/deployment/llm-server).

## HTTPS / custom domain

For automatic Let's Encrypt certificates on a custom domain, set `settings.letsEncrypt` in the workflow and publish ports **80** and **443**. Persist the certificate cache:

```bash
docker run -p 80:80 -p 443:443 \
  -v kdeps-le:/var/lib/kdeps/letsencrypt \
  myagent:latest
```

```yaml
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
    cacheDir: /var/lib/kdeps/letsencrypt
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 443
```

See [TLS and HTTPS (Custom Domains)](/deployment/tls-https). Static PEM mounts via `certFile`/`keyFile` remain supported.
