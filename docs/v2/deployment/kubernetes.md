# Kubernetes Deployment

`kdeps export k8s` generates a Kubernetes `Deployment` and `ClusterIP` `Service` from your `workflow.yaml` -- no manual YAML authoring required.

## Quick Start

```bash
# Generate manifests and print to stdout
kdeps export k8s examples/chatbot

# Save to a file and apply
kdeps export k8s examples/chatbot --output k8s.yaml
kubectl apply -f k8s.yaml
```

## Command Reference

```bash
kdeps export k8s [path] [flags]
```

**Arguments:**
- `path` - Directory containing `workflow.yaml`, a `workflow.yaml` file directly, or a `.kdeps` package

**Flags:**

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--image` | `-i` | Container image to use | `{name}:{version}` from workflow |
| `--output` | `-o` | Output file path | stdout |
| `--replicas` | `-r` | Number of replicas (overrides `workflow.yaml`) | From workflow |

**Examples:**

```bash
# Stdout output using default image name
kdeps export k8s examples/chatbot

# Custom image from a private registry
kdeps export k8s examples/chatbot --image registry.example.com/chatbot:v1.2.0

# Override replicas at export time
kdeps export k8s examples/chatbot --replicas 5

# Save directly to a file
kdeps export k8s examples/chatbot --output deploy/k8s.yaml

# Full example with all flags
kdeps export k8s examples/chatbot \
  --image registry.example.com/chatbot:v1.2.0 \
  --replicas 3 \
  --output deploy/k8s.yaml
```

## Workflow Configuration

Configure Kubernetes-specific settings in `workflow.yaml` under `agentSettings`:

```yaml
settings:
  portNum: 8080
  agentSettings:
    # Number of pod replicas
    replicas: 3

    # CPU and memory limits/requests
    resources:
      cpuLimit: "1000m"
      memoryLimit: "1Gi"
      cpuRequest: "250m"
      memoryRequest: "256Mi"

    # Env vars mapped to container environment
    env:
      APP_MODE: production
      DATABASE_URL: "postgres://user:pass@db-service:5432/mydb"
```

### Ports

Ports are derived from your workflow settings:

| Setting | Port name in manifest | Description |
|---------|-----------------------|-------------|
| `apiServer:` with `portNum` | `api` | REST API server |
| `webServer:` with `portNum` | `web` | Web server |
| `installOllama: true` (or auto-detected) | `backend` | Ollama LLM backend (11434) |

Ollama port is auto-detected when any resource uses a `chat:` executor.

### Resources

When `resources` is set, the manifest includes both `limits` and `requests`:

```yaml
agentSettings:
  resources:
    cpuLimit: "500m"
    memoryLimit: "512Mi"
    cpuRequest: "100m"
    memoryRequest: "128Mi"
```

When `resources` is absent, no `resources:` block is emitted (Kubernetes defaults apply).

## Generated Manifests

`kdeps export k8s` produces a single YAML document with two resources separated by `---`.

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chatbot
  labels:
    app: chatbot
    version: 1.0.0
    kdeps-component: "true"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: chatbot
  template:
    metadata:
      labels:
        app: chatbot
    spec:
      containers:
      - name: chatbot
        image: registry.example.com/chatbot:v1.0.0
        ports:
        - containerPort: 8080
          name: api
        env:
        - name: APP_MODE
          value: "production"
        resources:
          limits:
            cpu: "1000m"    # hard cap -- container is throttled at this limit
            memory: "1Gi"   # hard cap -- container is OOM-killed if exceeded
          requests:
            cpu: "250m"     # guaranteed allocation used for scheduling
            memory: "256Mi" # guaranteed allocation used for scheduling
        readinessProbe:     # traffic is held until this passes
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10  # wait 10s after start before first probe
          periodSeconds: 10        # re-probe every 10s
        livenessProbe:      # container is restarted if this fails
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30  # longer grace period than readiness
          periodSeconds: 30        # re-probe every 30s
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: chatbot
  labels:
    app: chatbot
spec:
  selector:
    app: chatbot
  ports:
  - name: api
    port: 8080
    targetPort: api
  type: ClusterIP
```

Health check probes are automatically generated from the configured port.

## Typical Workflow

```bash
# 1. Build the Docker image
kdeps bundle build examples/chatbot --tag registry.example.com/chatbot:v1.0.0

# 2. Push to registry
docker push registry.example.com/chatbot:v1.0.0

# 3. Generate manifests with the pushed image
kdeps export k8s examples/chatbot \
  --image registry.example.com/chatbot:v1.0.0 \
  --output k8s.yaml

# 4. Apply to your cluster
kubectl apply -f k8s.yaml

# 5. Verify rollout
kubectl rollout status deployment/chatbot
```

## Example: k8s-deployment

The `examples/k8s-deployment/` directory contains a full workflow demonstrating Kubernetes settings:

```bash
kdeps export k8s examples/k8s-deployment --image my-registry/k8s-example:1.0.0
```

See `examples/k8s-deployment/README.md` for details.

## Related

- [Docker Deployment](docker) - Build Docker images for your workflows
- [Standalone Executables](prepackage) - Self-contained binaries for edge deployment
- [CLI Reference](/reference/cli/) - Full command reference
