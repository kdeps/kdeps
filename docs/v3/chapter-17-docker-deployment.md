# Chapter 17: Docker Deployment

`kdeps bundle build` packages your workflow into a Docker image that starts an API server when run. No Dockerfile required — kdeps generates one from your `workflow.yaml`.

## The Two-Step Build

```bash
# Step 1: package workflow into an archive
$ kdeps bundle package workflow.yaml

# Step 2: build a Docker image from the archive
$ kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest
```

The `.kdeps` archive is the portable representation of your workflow. The Docker image is one deployment target for that archive. The same archive can also produce a Kubernetes deployment, a standalone binary, or a bootable ISO.

## Packaging

`kdeps bundle package` creates a `.kdeps` archive:

```bash
$ kdeps bundle package workflow.yaml
# Creates: myagent-1.0.0.kdeps (name and version from workflow metadata)

$ kdeps bundle package ./my-agency/agency.yaml
# Creates: my-agency-1.0.0.kdeps (for agencies)
```

The archive name comes from `metadata.name` and `metadata.version` in your manifest. Bumping the version in `workflow.yaml` produces a differently named archive, which is how you version deployments.

### What Goes Into the Archive

```
myagent-1.0.0.kdeps
├── workflow.yaml          # workflow entry point
├── resources/             # all resource YAML files
├── components/            # any custom components
├── data/                  # data files and scripts
├── requirements.txt       # Python dependencies (if present)
└── public/                # static files (if present)
```

Everything in the project directory is packaged. Use a `.kdepsignore` file (same syntax as `.gitignore`) to exclude files:

```
# .kdepsignore
.env
*.log
tmp/
test/
```

## Building Docker Images

```bash
$ kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest
```

kdeps generates a Dockerfile internally, calls Docker to build, and tags the result. You need Docker installed and running. The build process:

1. Pulls a base image (Alpine Linux + kdeps binary + runtime dependencies)
2. Upgrades the base OS packages first, picking up security patches published after the base image was built
3. Copies the `.kdeps` archive into the image
4. Pre-bakes the llamafile for every chat model into `/app/.kdeps/models/` so the container needs no network for inference (default file backend; skipped when Ollama is opted in)
5. Sets the entrypoint to `kdeps run` with the embedded workflow
6. Applies your `agentSettings.osPackages` as additional package install layers (`apk` on Alpine, `apt-get` on Ubuntu/Debian)
7. Tags the image

With the default backend the base stays a vanilla `alpine:latest` or
`ubuntu:latest` — no LLM server is installed. Only an explicit ollama opt-in
(`installOllama: true` or `KDEPS_DEFAULT_BACKEND=ollama`) switches the base to
an `ollama/ollama` image.

### GPU Support

For workflows that opt into local Ollama with GPU inference, build a GPU-capable image:

```bash
$ kdeps bundle build myagent-1.0.0.kdeps --gpu cuda   --tag myregistry/myagent:latest-cuda
$ kdeps bundle build myagent-1.0.0.kdeps --gpu rocm   --tag myregistry/myagent:latest-rocm
$ kdeps bundle build myagent-1.0.0.kdeps --gpu intel  --tag myregistry/myagent:latest-intel
$ kdeps bundle build myagent-1.0.0.kdeps --gpu vulkan --tag myregistry/myagent:latest-vulkan
```

| Flag | Base image | Use case |
|---|---|---|
| `--gpu cuda` | NVIDIA CUDA | RTX / A100 / H100 / datacenter NVIDIA |
| `--gpu rocm` | AMD ROCm | AMD Radeon GPUs |
| `--gpu intel` | Intel oneAPI | Intel Arc / Xe GPUs |
| `--gpu vulkan` | Vulkan | Cross-platform; broad GPU support |

The same `.kdeps` archive is used for all variants — only the base image changes.

### Inspecting the Generated Dockerfile

View the Dockerfile kdeps would generate without actually building the image:

```bash
$ kdeps bundle build myagent-1.0.0.kdeps --show-dockerfile
```

This prints the generated Dockerfile to stdout. Useful for auditing, customising, or debugging the build before running it.

### Multi-Architecture Builds

Build for multiple CPU architectures:

```bash
$ kdeps bundle build myagent-1.0.0.kdeps \
  --tag myregistry/myagent:latest \
  --platform linux/amd64,linux/arm64
```

This is important for:
- Deploying to AWS Graviton (arm64) instances
- Deploying to Apple Silicon (arm64) development machines
- Running on Raspberry Pi or other ARM edge devices

## Running the Docker Image

```bash
$ docker run -p 16395:16395 \
  -e DATABASE_URL="postgresql://..." \
  -e KDEPS_API_AUTH_TOKEN="secret123" \
  -e KDEPS_MANAGEMENT_TOKEN="mgmt-secret" \
  myregistry/myagent:latest
```

The container starts the kdeps server on the port configured in `workflow.yaml`. Map it with `-p hostPort:containerPort`.

Pass credentials and environment-specific config via `-e` flags. Never bake secrets into the image. When `apiServer` is configured, `KDEPS_API_AUTH_TOKEN` is required — the container exits on startup without it. Set `KDEPS_MANAGEMENT_TOKEN` if you need live workflow updates via `/_kdeps/*`.

kdeps images run as the non-root `kdeps` user. Pre-baked llamafiles live under `/app/.kdeps/models`; Ollama-enabled images store models under `/app/.ollama/models`.

### With Volumes

For workflows that use SQLite databases, embedding stores, or file I/O, mount a persistent volume:

```bash
$ docker run -p 16395:16395 \
  -v /data/myagent:/data \
  -e DATABASE_URL="..." \
  myregistry/myagent:latest
```

The workflow's `sqlConnections.cache.path: "/data/cache.db"` and `session.path: "/data/sessions.db"` write to the mounted volume and persist across container restarts.

### With Ollama (Local LLM, Opt-In)

The default file backend needs no companion container — the llamafile is baked
into the image and serves itself. If you have opted into Ollama, run the agent
alongside an Ollama instance:

```yaml
# docker-compose.yaml
version: "3.9"
services:
  ollama:
    image: ollama/ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
  
  myagent:
    image: myregistry/myagent:latest
    ports:
      - "16395:16395"
    environment:
      DATABASE_URL: "${DATABASE_URL}"
      KDEPS_API_AUTH_TOKEN: "${KDEPS_API_AUTH_TOKEN}"
      KDEPS_MANAGEMENT_TOKEN: "${KDEPS_MANAGEMENT_TOKEN}"
    depends_on:
      - ollama
    volumes:
      - agent-data:/data

volumes:
  ollama-data:
  agent-data:
```

The agent connects to Ollama at `http://ollama:11434` (using the service name as hostname in the Docker network). Configure this in `~/.kdeps/config.yaml` embedded in the image, or as an environment variable.

## Pushing to a Registry

```bash
$ docker login myregistry.example.com
$ docker push myregistry/myagent:latest
$ docker push myregistry/myagent:1.0.0    # also push version tag
```

For CI/CD pipelines:

```bash
# Build and push in one script
VERSION=$(grep 'version:' workflow.yaml | head -1 | awk '{print $2}' | tr -d '"')

kdeps bundle package workflow.yaml
kdeps bundle build myagent-${VERSION}.kdeps \
  --tag myregistry/myagent:${VERSION} \
  --tag myregistry/myagent:latest

docker push myregistry/myagent:${VERSION}
docker push myregistry/myagent:latest
```

## Environment Configuration Patterns

**Development (local):**

```bash
$ docker run -p 16395:16395 --env-file .env myregistry/myagent:latest
```

**.env file:**

```
DATABASE_URL=postgresql://user:pass@localhost:5432/dev
KDEPS_API_AUTH_TOKEN=dev-api-token
KDEPS_MANAGEMENT_TOKEN=dev-mgmt-token
OPENAI_API_KEY=sk-...
```

**Production (Docker Swarm / Compose):**

```yaml
services:
  myagent:
    image: myregistry/myagent:1.0.0
    environment:
      DATABASE_URL: "${DATABASE_URL}"              # from host environment
      KDEPS_API_AUTH_TOKEN: "${KDEPS_API_AUTH_TOKEN}"
      KDEPS_MANAGEMENT_TOKEN: "${KDEPS_MANAGEMENT_TOKEN}"
    secrets:
      - api_token      # inject via your orchestrator; never commit values
      - mgmt_token
```

**Production (Kubernetes):**

```yaml
# k8s-deployment.yaml
env:
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: myagent-secrets
        key: database_url
  - name: KDEPS_API_AUTH_TOKEN
    valueFrom:
      secretKeyRef:
        name: myagent-secrets
        key: api_token
  - name: KDEPS_MANAGEMENT_TOKEN
    valueFrom:
      secretKeyRef:
        name: myagent-secrets
        key: mgmt_token
```

The workflow file never changes. Only the environment variables change between deployments.

## Image Size Optimization

Default kdeps images are minimal (Alpine Linux base). The main size contributors are:
- Python packages (`agentSettings.pythonPackages`)
- OS packages (`agentSettings.osPackages`)
- Bundled data files

To keep images lean:
- Pin Python package versions to avoid large transitive dependency trees
- Use only the OS packages your workflow actually needs
- Keep data files out of the image; mount them via volumes

For reference: a base kdeps image with no additional dependencies is typically under 100MB. Adding pandas and numpy adds ~200MB. Adding a full CUDA stack for GPU support adds ~2GB.

## CI/CD Integration

A complete GitHub Actions workflow:

```yaml
# .github/workflows/deploy.yaml
name: Build and Deploy

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install kdeps
        run: curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
      
      - name: Package workflow
        run: kdeps bundle package workflow.yaml
      
      - name: Build Docker image
        run: |
          VERSION=$(grep 'version:' workflow.yaml | head -1 | awk '{print $2}' | tr -d '"')
          kdeps bundle build myagent-${VERSION}.kdeps \
            --tag $&#123;&#123; secrets.REGISTRY &#125;&#125;/myagent:${VERSION} \
            --tag $&#123;&#123; secrets.REGISTRY &#125;&#125;/myagent:latest
      
      - name: Push to registry
        run: |
          echo $&#123;&#123; secrets.REGISTRY_PASSWORD &#125;&#125; | docker login $&#123;&#123; secrets.REGISTRY &#125;&#125; -u $&#123;&#123; secrets.REGISTRY_USERNAME &#125;&#125; --password-stdin
          docker push $&#123;&#123; secrets.REGISTRY &#125;&#125;/myagent:latest
      
      - name: Deploy to Kubernetes
        run: kubectl set image deployment/myagent myagent=$&#123;&#123; secrets.REGISTRY &#125;&#125;/myagent:latest
```

In the next chapter, we cover Kubernetes deployment in detail — from the generated manifests to production-grade configurations.

X> ## Exercise
X>
X> Package the chatbot from Chapter 2 into a Docker image and run it as a container.
X>
X> 1. Package the workflow: `kdeps bundle package workflow.yaml`. Verify a `.kdeps` archive was created.
X> 2. Build a Docker image: `kdeps bundle build myagent-1.0.0.kdeps --tag myagent:exercise`. Inspect the generated Dockerfile first with `--show-dockerfile` and note the layers.
X> 3. Run the container:
X>    ```bash
X>    docker run -p 16395:16395 \
X>      -e OLLAMA_HOST="http://host.docker.internal:11434" \
X>      myagent:exercise
X>    ```
X> 4. Hit the endpoint from outside the container: `curl -X POST localhost:16395/api/v1/chat -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" -d '{"q":"hello"}'`
X> 5. Create a `.kdepsignore` file that excludes `.env`, `*.log`, and any `test/` directory. Rebuild and verify the archive size decreased.
X>
X> Add a GPU variant build (`--gpu cuda`) and compare the image sizes with `docker images`.
X>
X> **Stretch goal:** Write a `docker-compose.yaml` that starts both Ollama and your agent container, with the agent configured to reach Ollama via the service name `ollama` inside the Docker network.


## LLM server appliance (not an agent image)

Agent images from this chapter package a workflow and the kdeps runtime. To deploy a **standalone** OpenAI-compatible LLM server (no workflow path, no agent process), use `kdeps llm`:

```bash
$ kdeps llm list
$ kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
$ kdeps llm run --engine ollama --model llama3.2 -p 8000
$ kdeps llm client-config --url http://host:8000/v1
```

Stock engines: ollama, llamafile, GGUF/llama-server, vLLM, TGI, SGLang, LocalAI. Full walkthrough: **Chapter 26**. Reference docs: `docs/v2/deployment/llm-server.md`.

## HTTPS / custom domain

For automatic Let's Encrypt certificates on a public hostname, set `settings.letsEncrypt` (Chapter 14) and publish ports **80** and **443**. Persist the cert cache volume/PVC.

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

Docs: `docs/v2/deployment/tls-https.md`.
