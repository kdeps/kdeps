# Docker Configuration Example

This example demonstrates all Docker configuration features in KDeps v2.

## Features Demonstrated

1. **Base OS Selection** - Choose Alpine, Ubuntu, or Debian
2. **OS Package Installation** - Install system-level packages
3. **Python Package Management** - Specify Python packages
4. **Auto-backend Installation** - Automatically install Ollama LLM backend

---

## Configuration via Workflow

### Base OS Selection

```yaml
agentSettings:
  # Options: alpine, ubuntu, debian
  # Default: alpine
  baseOS: "alpine"
```

**Can be overridden via CLI:**
```bash
# Use workflow's baseOS (alpine)
kdeps build .

# Override with Ubuntu
kdeps build . --os ubuntu

# Override with Debian
kdeps build . --os debian
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
- Ubuntu/Debian: `apt` (e.g., `git`, `vim`, `curl`)

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

Control Ollama installation with the `installOllama` flag:

```yaml
agentSettings:
  installOllama: true   # Install Ollama for local LLM support
  # or
  installOllama: false  # Disable Ollama (for cloud-only workflows)
```

When `installOllama: true`, Ollama is installed via the official install script and you can use local LLM resources:

```yaml
resources:
  - metadata:
      actionId: llm
    run:
      chat:
        backend: "ollama"
        model: "llama3.2:1b"
        prompt: "{{ get('q') }}"
```

Ollama is automatically installed when:
- `installOllama: true` is explicitly set
- Models are configured in `agentSettings.models` (implies Ollama usage)

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
# Or explicitly:
kdeps build . --os alpine
```

**Result:** ~200MB image with Ollama

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

### Example 3: Debian with Database Tools

```yaml
agentSettings:
  baseOS: "debian"
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

**Result:** Debian image with PostgreSQL client and Redis tools

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

# Preview with different OS
kdeps build . --show-dockerfile --os ubuntu
```

### Example Output (Alpine + Ollama):

```dockerfile
FROM alpine:3.18

# Install Python and base dependencies (Alpine)
RUN apk add --no-cache python3 py3-pip curl bash && \
    python3 -m ensurepip && \
    pip3 install --upgrade pip

# Install OS packages
RUN apk add --no-cache git vim curl jq

# Install Ollama
RUN curl -fsSL https://ollama.com/install.sh | sh

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv
RUN chmod +x /usr/local/bin/uv

# Create virtual environment
RUN uv venv /opt/venv
ENV PATH=/opt/venv/bin:$PATH

# Install Python packages
RUN uv pip install requests numpy pandas

# Copy workflow files
COPY workflow.yaml /app/workflow.yaml
COPY resources/ /app/resources/
COPY data/ /app/data/

WORKDIR /app

# Run kdeps
CMD ["kdeps", "run", "workflow.yaml"]
```

---

## Build Process

### 1. Workflow Configuration Takes Precedence

```yaml
agentSettings:
  baseOS: "debian"
```

**Result:** Builds with Debian unless overridden

### 2. CLI Override

```bash
kdeps build . --os ubuntu
```

**Result:** Builds with Ubuntu (overrides workflow)

### 3. Default Behavior

No baseOS in workflow + no CLI flag = **Alpine** (default)

---

## OS Comparison

### Alpine
- **Size:** Smallest (~50-100MB less than Ubuntu/Debian)
- **Best for:** Lightweight APIs, simple workflows
- **Package Manager:** `apk`
- **Use when:** Image size is critical

### Ubuntu
- **Size:** Largest (but most packages available)
- **Best for:** Complex applications, data science
- **Package Manager:** `apt`
- **Use when:** Need maximum compatibility

### Debian
- **Size:** Medium (between Alpine and Ubuntu)
- **Best for:** Production workloads, stability
- **Package Manager:** `apt`
- **Use when:** Need balance of size and features

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
    run:
      chat:
        backend: "ollama"
        model: "llama3.2:1b"
        prompt: "Quick answer: {{ get('q') }}"

  # Cloud provider for complex tasks
  - metadata:
      actionId: deepAnalysis
    run:
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
curl -X POST 'http://localhost:3000/api/v1/chat?q=Hello'

# Build Docker image
kdeps build . --tag docker-config:latest

# Run Docker image
docker run -p 3000:3000 docker-config:latest
```

---

## Tips

1. **Start with Alpine** for simplest workflows
2. **Use Ubuntu** when you need specific packages
3. **Use Debian** for production stability
4. **Install only what you need** - keeps images small
5. **Test locally first** before building Docker
6. **Use `--show-dockerfile`** to preview before building
7. **Use `installOllama: false`** for cloud-only workflows

---

## Notes

- baseOS can be specified in workflow AND overridden via CLI
- Ollama is auto-detected from resources or explicitly controlled via `installOllama`
- OS packages use appropriate package manager (apk/apt)
- Python packages installed via uv (fast and reliable)
- All images include kdeps binary for execution
