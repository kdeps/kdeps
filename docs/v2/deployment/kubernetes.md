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
| `--network-policy` | | Also generate a NetworkPolicy restricting ingress to the configured ports | Off (or `agentSettings.networkPolicy`) |

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
# workflow.yaml
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

    # Opt in to a NetworkPolicy that restricts pod ingress to the
    # configured server ports. Egress stays unrestricted so chat,
    # httpClient, searchWeb, and sql resources can reach external services.
    networkPolicy: true
```

### Ports

Ports are derived from your workflow settings:

| Setting | Port name in manifest | Description |
|---------|-----------------------|-------------|
| `apiServer:` with `portNum` | `api` | REST API server |
| `webServer:` with `portNum` | `web` | Web server |
| `installOllama: true` (or `KDEPS_DEFAULT_BACKEND=ollama` with chat resources) | `backend` | Ollama LLM backend (11434) |

Chat resources on the default `file` backend need no backend port: the
llamafile self-serves on localhost inside the pod.

Ports, probes, and NetworkPolicy rules are all derived from this configuration -- only what the workflow actually serves is exposed. A workflow with no `apiServer` or `webServer` (a bot or file workflow) gets no container ports and no probes.

### Resources

When `resources` is set, the manifest includes both `limits` and `requests`:

```yaml
# workflow.yaml
agentSettings:
  resources:
    cpuLimit: "500m"
    memoryLimit: "512Mi"
    cpuRequest: "100m"
    memoryRequest: "128Mi"
```

When `resources` is absent, no `resources:` block is emitted (Kubernetes defaults apply).

## Generated Manifests

`kdeps export k8s` produces a single YAML document with a Deployment and a Service separated by `---` (plus a NetworkPolicy when opted in).

### Deployment

```yaml
# k8s.yaml
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
            port: api       # named port -- follows apiServer.portNum automatically
          initialDelaySeconds: 10  # wait 10s after start before first probe
          periodSeconds: 10        # re-probe every 10s
        livenessProbe:      # container is restarted if this fails
          httpGet:
            path: /health
            port: api
          initialDelaySeconds: 30  # longer grace period than readiness
          periodSeconds: 30        # re-probe every 30s
```

### Service

```yaml
# k8s.yaml
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

Probes follow what the workflow serves: an `apiServer` workflow gets HTTP probes on `/health` at the `api` port; a web-only workflow gets TCP probes on the `web` port (the web server has no `/health` endpoint); a workflow with neither gets no probes.

### NetworkPolicy (opt-in)

Set `agentSettings.networkPolicy: true` in `workflow.yaml` (or pass `--network-policy` at export time) to append a NetworkPolicy. Ingress is allowed only on the ports the workflow actually serves; everything else is denied. Egress is deliberately unrestricted so `chat`, `httpClient`, `searchWeb`, and `sql` resources can reach external services.

```yaml
# k8s.yaml (appended when networkPolicy is enabled)
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: chatbot
  labels:
    app: chatbot
    kdeps-component: "true"
spec:
  podSelector:
    matchLabels:
      app: chatbot
  policyTypes:
  - Ingress          # egress is not listed, so it stays unrestricted
  ingress:
  - ports:
    - protocol: TCP
      port: 8080     # only the configured apiServer port accepts traffic
```

The Ollama backend port (11434) is never opened for ingress: Ollama binds `127.0.0.1` inside the pod, so it is only reachable from within the pod regardless. A workflow with no `apiServer` or `webServer` gets a policy with no ingress rules at all -- all ingress denied.

Your cluster must run a CNI that enforces NetworkPolicy (Calico, Cilium, etc.); on clusters without one the policy is accepted but has no effect.

When `apiServer` is configured, the Deployment references auth tokens from a Kubernetes Secret (never from `agentSettings.env`):

```yaml
# deploy/auth-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: chatbot-auth
type: Opaque
stringData:
  api-token: "your-api-secret"
  management-token: "your-mgmt-secret"   # optional; omit if you do not use /_kdeps/*
```

The generated Deployment expects `secretKeyRef.name` to be `{metadata.name}-auth` with keys `api-token` and `management-token` (management is optional). Create the Secret before applying the Deployment:

```bash
kubectl apply -f deploy/auth-secret.yaml
kdeps export k8s examples/chatbot --output k8s.yaml
kubectl apply -f k8s.yaml
```

Secret-like keys in `agentSettings.env` (for example `OPENAI_API_KEY`) are not baked into the manifest. Export emits `secretKeyRef` entries against `{metadata.name}-env` instead:

```yaml
# deploy/env-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: chatbot-env
type: Opaque
stringData:
  OPENAI_API_KEY: "sk-..."
```

```bash
kubectl apply -f deploy/env-secret.yaml
```

Pod `securityContext` defaults include `runAsNonRoot: true`, `seccompProfile.type: RuntimeDefault`, and `capabilities.drop: ["ALL"]`. The pod also sets `automountServiceAccountToken: false` since kdeps workloads never call the Kubernetes API.

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

## See Also

- [Docker Deployment](docker) - Build Docker images for your workflows
- [Standalone Executables](prepackage) - Self-contained binaries for edge deployment
- [CLI Reference](/reference/cli/) - Full command reference
