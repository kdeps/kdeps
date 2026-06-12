# Docker Configuration Example

This example demonstrates all Docker configuration features in KDeps v2.

## Features Demonstrated

1. **Base OS Selection** - Choose Alpine or Ubuntu
2. **OS Package Installation** - Install system-level packages
3. **Python Package Management** - Specify Python packages
4. **Auto-backend Installation** - Automatically install Ollama LLM backend

---

## Configuration via Workflow

### Base OS Selection

```yaml
agentSettings:
  # Options: alpine, ubuntu
  # Default: alpine
  baseOS: "alpine"
```

**GPU builds auto-select Ubuntu:**
```bash
kdeps build .              # CPU: alpine (or workflow baseOS)
kdeps build . --gpu cuda   # GPU: ubuntu + official ollama/ollama when Ollama is enabled
```

### OS Packages

Install OS-level packages (git, vim, curl, etc.):

```yaml
agentSettings:
  osPackages:
    - git
    - vim
    - curl
    - jq
    - postgresql-client  # Database client
    - redis-tools        # Redis CLI
```

**Package managers by OS:**
- Alpine: `apk` (e.g., `git`, `vim`, `curl`)
- Ubuntu: `apt` (e.g., `git`, `vim`, `curl`)

### Python Packages

```yaml
agentSettings:
  pythonVersion: "3.12"
  pythonPackages:
    - requests
    - numpy
    - pandas
    - scikit-learn

  # Or use requirements file
  requirementsFile: "requirements.txt"
```

### LLM Backend Installation

By default no LLM server is installed: chat resources run on the `file`
backend, and the referenced llamafile models are pre-baked into the image.
Ollama is an explicit opt-in via the `installOllama` flag:

```yaml
agentSettings:
  installOllama: true   # Bake the ollama server into the image
  env:
    KDEPS_DEFAULT_BACKEND: ollama  # route chat resources to ollama at runtime
```

Ollama is installed when:
- `installOllama: true` is explicitly set
- `KDEPS_DEFAULT_BACKEND=ollama` is set and the workflow has chat resources
- The LLM router config (`KDEPS_LLM_ROUTER`) contains ollama routes

---

## Examples

### Example 1: Lightweight Alpine with Ollama

```yaml
agentSettings:
  baseOS: "alpine"
  installOllama: true
  pythonPackages:
    - requests
  osPackages:
    - curl
```

**Build:**
```bash
kdeps build .
```

**Result:** ~70MB Ollama base (`alpine/ollama`) plus kdeps layers

---

### Example 2: Ubuntu with Cloud Providers

```yaml
agentSettings:
  baseOS: "ubuntu"
  installOllama: false  # No local LLM needed
  pythonPackages:
    - openai
    - anthropic
  osPackages:
    - git
    - build-essential

resources:
  - run:
      chat:
        backend: "openai"
        model: "gpt-4o"
        apiKey: "{{ get('OPENAI_API_KEY', 'env') }}"
```

**Build:**
```bash
kdeps build .
```

**Result:** Ubuntu image without Ollama (uses cloud APIs)

---

### Example 3: Ubuntu with Database Tools

```yaml
agentSettings:
  baseOS: "ubuntu"
  pythonPackages:
    - psycopg2-binary
    - sqlalchemy
  osPackages:
    - postgresql-client
    - redis-tools
    - git
```

**Build:**
```bash
kdeps build .
```

**Result:** Ubuntu image with PostgreSQL client and Redis tools

---

### Example 4: Data Science Stack

```yaml
agentSettings:
  baseOS: "ubuntu"
  pythonVersion: "3.11"
  pythonPackages:
    - numpy
    - pandas
    - scikit-learn
    - matplotlib
    - jupyter
  osPackages:
    - git
    - vim
    - graphviz
```

**Build:**
```bash
kdeps build . --tag datascience:latest
```

---

## Generated Dockerfile Preview

```bash
# Preview what will be generated
kdeps build . --show-dockerfile

# Preview with Ubuntu baseOS in workflow.yaml
kdeps build . --show-dockerfile
```

### Example Output (Alpine + Ollama):

```dockerfile
FROM alpine/ollama:0.5.0

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    PATH=/opt/venv/bin:$PATH \
    OLLAMA_HOST=127.0.0.1 \
    OLLAMA_PORT=11434 \
    BACKEND_PORT=11434

# Install base dependencies
RUN apk add --no-cache \
    zstd \
    python3 \
    py3-pip \
    curl \
    bash \
    supervisor \
    ca-certificates \
    libstdc++ \
    rsync

# Install kdeps via official install script
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh -s -- -b /usr/local/bin

# Install OS packages
RUN apk add --no-cache git vim curl jq

# Ollama included in base image

# Install uv for Python package management
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv
RUN chmod +x /usr/local/bin/uv

# Create virtual environment
RUN uv venv /opt/venv

# Install Python packages
RUN --mount=type=cache,target=/root/.cache/uv \
    uv pip install requests numpy pandas

# Copy workflow files
COPY workflow.yaml /app/workflow.yaml
COPY resources/ /app/resources/
COPY data/ /app/data/

# Copy entrypoint and supervisor config
COPY entrypoint.sh /entrypoint.sh
COPY supervisord.conf /etc/supervisord.conf
RUN chmod +x /entrypoint.sh

WORKDIR /app

# Expose ports
EXPOSE 16395 11434

# Use entrypoint for backend management
ENTRYPOINT ["/entrypoint.sh"]
CMD ["supervisord", "-c", "/etc/supervisord.conf"]
```

---

## Build Process

### 1. Workflow `baseOS`

```yaml
agentSettings:
  baseOS: "ubuntu"
```

**Result:** Ubuntu base (unless `--gpu` is set, which also forces Ubuntu)

### 2. `--gpu` Flag

```bash
kdeps build . --gpu cuda
```

**Result:** Ubuntu + `ollama/ollama` when Ollama is enabled

### 3. Default Behavior

No `baseOS` in workflow + no `--gpu` = **Alpine** (default)

---

## OS Comparison

### Alpine
- **Size:** Smallest (~300MB without Ollama; ~70MB Ollama base via `alpine/ollama`)
- **Best for:** Lightweight APIs, CPU-only local LLM
- **Package Manager:** `apk`
- **Use when:** Image size is critical

### Ubuntu
- **Size:** Larger (~800MB+ with official `ollama/ollama`)
- **Best for:** GPU inference, complex applications, data science
- **Package Manager:** `apt`
- **Use when:** Need GPU drivers or maximum apt compatibility

---

## Hybrid Local + Cloud Workflows

You can combine local Ollama with cloud providers:

```yaml
agentSettings:
  installOllama: true  # Enable local Ollama

resources:
  # Fast local inference for simple tasks
  - metadata:
      actionId: quickChat
    chat:
      backend: "ollama"
      model: "llama3.2:1b"
      prompt: "Quick answer: {{ get('q') }}"

  # Cloud provider for complex tasks
  - metadata:
      actionId: deepAnalysis
    chat:
      backend: "anthropic"
      model: "claude-3-5-sonnet-20241022"
      apiKey: "{{ get('ANTHROPIC_API_KEY', 'env') }}"
      prompt: "Detailed analysis: {{ get('q') }}"
```

**Result:** Ollama installed locally, cloud APIs used for specific resources

---

## Testing

```bash
# Run the example
kdeps run workflow.yaml --dev

# Test the API
curl -X POST 'http://localhost:16395/api/v1/chat?q=Hello'

# Build Docker image
kdeps build . --tag docker-config:latest

# Run Docker image
docker run -p 16395:16395 docker-config:latest
```

---

## Tips

1. **Start with Alpine** for simplest CPU workflows
2. **Use Ubuntu** when you need GPU or specific apt packages
4. **Install only what you need** - keeps images small
5. **Test locally first** before building Docker
6. **Use `--show-dockerfile`** to preview before building
7. **Use `installOllama: false`** for cloud-only workflows

---

## Notes

- `baseOS` in workflow selects alpine or ubuntu; `--gpu` forces ubuntu
- Ollama is auto-detected from resources or explicitly controlled via `installOllama`
- OS packages use appropriate package manager (apk/apt)
- Python packages installed via uv (fast and reliable)
- All images include kdeps binary for execution
