# Chapter 18: Kubernetes Deployment

`kdeps export k8s` generates Kubernetes manifests from your `workflow.yaml`. No manual YAML authoring required. Run the command, apply the output to your cluster, and your agent is running.

## Generating Manifests

```bash
# Print manifests to stdout
$ kdeps export k8s examples/chatbot

# Save to a file
$ kdeps export k8s examples/chatbot --output k8s.yaml

# Apply directly
$ kdeps export k8s examples/chatbot --output k8s.yaml && kubectl apply -f k8s.yaml
```

The command accepts a directory containing `workflow.yaml`, a path directly to `workflow.yaml`, or a `.kdeps` package archive.

## What Gets Generated

`kdeps export k8s` produces a Deployment and a Service, plus an optional NetworkPolicy (see below). Everything is derived from your `workflow.yaml`: each configured server contributes its own named port, probes match what the workflow actually serves, and the pod ships hardened by default.

**Deployment** — manages the pod(s) running your agent:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-agent
  labels:
    app: my-agent
    version: 1.0.0
    kdeps-component: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-agent
  template:
    metadata:
      labels:
        app: my-agent
    spec:
      automountServiceAccountToken: false  # kdeps workloads never call the Kubernetes API
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault     # runtime default syscall filter
      containers:
      - name: my-agent
        image: my-agent:1.0.0
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: false
          capabilities:
            drop: ["ALL"]          # no Linux capabilities — ports below 1024 cannot be bound
        ports:
        - containerPort: 16395
          name: api                # named port — probes and the Service reference it by name
        env:
        - name: KDEPS_API_AUTH_TOKEN
          valueFrom:
            secretKeyRef:
              name: my-agent-auth  # you create this Secret; kdeps never bakes tokens in
              key: api-token
        - name: KDEPS_MANAGEMENT_TOKEN
          valueFrom:
            secretKeyRef:
              name: my-agent-auth
              key: management-token
              optional: true       # omit the key if you do not use /_kdeps/* routes
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
          requests:
            cpu: "250m"
            memory: "256Mi"
        readinessProbe:            # generated automatically — traffic is held until this passes
          httpGet:
            path: /health
            port: api
          initialDelaySeconds: 10
          periodSeconds: 10
        livenessProbe:             # generated automatically — container restarts if this fails
          httpGet:
            path: /health
            port: api
          initialDelaySeconds: 30
          periodSeconds: 30
```

The pod security defaults (non-root, all capabilities dropped, runtime-default seccomp profile, no service account token) satisfy the Kubernetes Pod Security Standards "restricted" profile. One practical consequence: the container cannot bind ports below 1024, so keep `portNum` above that and let the Ingress map 80/443.

**Service** — exposes the deployment within the cluster:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-agent
  labels:
    app: my-agent
spec:
  selector:
    app: my-agent
  ports:
  - name: api
    port: 16395        # same as the workflow's portNum
    targetPort: api    # references the named container port
  type: ClusterIP
```

Ports follow your configuration: `apiServer` contributes an `api` port, `webServer` contributes a `web` port, and Ollama-backed workflows expose `backend` (11434). A workflow with no servers (a bot or file workflow) gets no ports and no probes. Web-only workflows get TCP probes on the `web` port instead of HTTP probes, because the web server has no `/health` endpoint.

## Command Reference

```bash
kdeps export k8s [path] [flags]
```

| Flag | Description |
|---|---|
| `--image`, `-i` | Container image to use (default: `{name}:{version}` from workflow) |
| `--output`, `-o` | Output file path (default: stdout) |
| `--replicas`, `-r` | Number of pod replicas (overrides `workflow.yaml`) |
| `--network-policy` | Also generate a NetworkPolicy restricting ingress to the configured ports |

### Custom Image

When your image lives in a private registry:

```bash
$ kdeps export k8s ./my-agent \
  --image registry.example.com/my-agent:1.0.0 \
  --output k8s.yaml
```

### Setting Replicas

```bash
$ kdeps export k8s ./my-agent --replicas 3 --output k8s.yaml
```

Or set replicas in `workflow.yaml`:

```yaml
settings:
  agentSettings:
    replicas: 3
```

### CPU and Memory Limits

Set resource requests and limits in `agentSettings.resources`. When present, kdeps includes both `limits:` and `requests:` in the generated Deployment:

```yaml
# workflow.yaml
settings:
  agentSettings:
    resources:
      cpuLimit: "1000m"      # hard cap — container is throttled at this limit
      memoryLimit: "1Gi"     # hard cap — container is OOM-killed if exceeded
      cpuRequest: "250m"     # guaranteed allocation used for scheduling
      memoryRequest: "256Mi" # guaranteed allocation used for scheduling
```

When `resources` is omitted, no `resources:` block is emitted and Kubernetes defaults apply.

A rule of thumb for LLM-heavy agents: set `memoryLimit` generously (each in-flight request holds DAG state) and `cpuRequest` conservatively (LLM calls are mostly I/O-bound).

## Adding Secrets and Config

The generator wires secrets up for you — it never bakes credential values into the manifest. You create the Secrets it references; no editing of the generated YAML required.

Two Secret names are reserved by convention:

- `{name}-auth` — API auth tokens. When `apiServer` is configured, the Deployment references `KDEPS_API_AUTH_TOKEN` (key `api-token`) and `KDEPS_MANAGEMENT_TOKEN` (key `management-token`, optional) from this Secret.
- `{name}-env` — secret-like keys from `agentSettings.env`. Anything that looks like a credential (for example `OPENAI_API_KEY`) is emitted as a `secretKeyRef` against this Secret instead of a literal value. Non-secret env values are inlined as plain `value:` entries.

```bash
# Auth tokens for the API server
$ kubectl create secret generic my-agent-auth \
  --from-literal=api-token="secret123" \
  --from-literal=management-token="mgmt-secret"

# Secret-like keys referenced from agentSettings.env
$ kubectl create secret generic my-agent-env \
  --from-literal=OPENAI_API_KEY="sk-..."
```

Create both Secrets before applying the Deployment — pods will not start with the references unresolved (except `management-token`, which is marked `optional`).

### Using ConfigMaps for Non-Secret Config

```bash
$ kubectl create configmap myagent-config \
  --from-literal=LOG_LEVEL="info" \
  --from-literal=API_BASE_URL="https://api.internal.example.com"
```

```yaml
envFrom:
  - configMapRef:
      name: myagent-config
```

## Persistent Storage

For workflows that use SQLite databases or local file storage, mount a PersistentVolumeClaim:

```yaml
# pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myagent-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

```yaml
# Add to the generated deployment
spec:
  volumes:
    - name: agent-data
      persistentVolumeClaim:
        claimName: myagent-data
  containers:
    - name: my-agent
      volumeMounts:
        - name: agent-data
          mountPath: /data
```

Note: `ReadWriteOnce` means one pod can write at a time. If you run multiple replicas, use `ReadWriteMany` with a network filesystem, or switch to a shared database backend for session storage.

## Exposing the Agent

The generated Service is `ClusterIP` — accessible within the cluster only. To expose it externally:

### Ingress (recommended)

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-agent
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - api.example.com
      secretName: myagent-tls
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /api/
            pathType: Prefix
            backend:
              service:
                name: my-agent
                port:
                  name: api   # the generated Service names its ports after the servers
```

### LoadBalancer Service

```yaml
# service-external.yaml
apiVersion: v1
kind: Service
metadata:
  name: my-agent-external
spec:
  selector:
    app: my-agent
  ports:
    - port: 80
      targetPort: api   # maps external port 80 to the workflow's api port
  type: LoadBalancer
```

## Health Checks

Probes are generated automatically and follow what the workflow serves — you do not write them by hand:

| Workflow | Readiness/liveness probes |
|---|---|
| `apiServer` configured | HTTP `GET /health` on the named `api` port |
| `webServer` only | TCP socket check on the named `web` port (no `/health` endpoint exists) |
| Neither (bot/file workflow) | No probes |

Because probes reference ports by name (`port: api`), changing `portNum` in `workflow.yaml` and regenerating keeps everything consistent — there is no second place to update.

If you need different timings (for example, a slow-starting Ollama model pull), edit the generated `initialDelaySeconds`/`periodSeconds` after export — the defaults are 10s/10s for readiness and 30s/30s for liveness.

## NetworkPolicy (Opt-In)

Lock pod ingress down to exactly the ports the workflow serves. Enable it in `workflow.yaml`:

```yaml
settings:
  agentSettings:
    networkPolicy: true   # appends a NetworkPolicy to the export
```

Or at export time:

```bash
$ kdeps export k8s ./my-agent --network-policy --output k8s.yaml
```

The generated policy:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: my-agent
  labels:
    app: my-agent
    kdeps-component: "true"
spec:
  podSelector:
    matchLabels:
      app: my-agent
  policyTypes:
  - Ingress            # egress is not listed, so it stays unrestricted
  ingress:
  - ports:
    - protocol: TCP
      port: 16395      # only the configured apiServer port accepts traffic
```

Three deliberate design choices:

- **Egress is unrestricted.** Your `chat`, `httpClient`, `searchWeb`, and `sql` resources must reach LLM providers, APIs, and databases. Restricting egress would break them.
- **The Ollama backend port is never opened.** Ollama binds `127.0.0.1` inside the pod, so it is unreachable from outside the pod regardless — no ingress rule needed.
- **No servers means deny-all ingress.** A bot or file workflow gets a policy with no ingress rules: nothing should be connecting to it.

Your cluster must run a CNI that enforces NetworkPolicy (Calico, Cilium, and most managed offerings do). On clusters without enforcement the policy is accepted but has no effect.

## Resource Limits

The generated manifest includes reasonable defaults. Adjust for your actual workload:

```yaml
resources:
  requests:
    memory: "256Mi"     # guaranteed minimum
    cpu: "250m"         # 0.25 CPU cores
  limits:
    memory: "2Gi"       # max (OOMKilled if exceeded)
    cpu: "2000m"        # max 2 CPU cores
```

LLM-heavy workflows need more memory if they are loading models. If you are using remote LLM providers (OpenAI, Anthropic), the resource requirements stay low — the compute is remote.

## Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-agent
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-agent
  minReplicas: 1
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

## Complete Production Setup

```bash
# 1. Build and push the image
$ kdeps bundle build ./my-agent --tag registry.example.com/my-agent:1.0.0
$ docker push registry.example.com/my-agent:1.0.0

# 2. Generate Kubernetes manifests (with ingress locked to the served ports)
$ kdeps export k8s ./my-agent \
  --image registry.example.com/my-agent:1.0.0 \
  --replicas 2 \
  --network-policy \
  --output k8s-base.yaml

# 3. Create the Secrets the manifests reference
$ kubectl create secret generic my-agent-auth \
  --from-literal=api-token="$(openssl rand -hex 32)"
$ kubectl create secret generic my-agent-env \
  --from-env-file=.env.prod          # OPENAI_API_KEY and friends

# 4. Apply everything
$ kubectl apply -f pvc.yaml
$ kubectl apply -f k8s-base.yaml
$ kubectl apply -f ingress.yaml
$ kubectl apply -f hpa.yaml

# 5. Verify
$ kubectl get pods -l app=my-agent
$ kubectl logs -l app=my-agent --tail=50
$ kubectl exec -it deploy/my-agent -- kdeps doctor
```

## Updating a Deployment

When you update the workflow and push a new image:

```bash
# Rolling update
$ kubectl set image deployment/my-agent my-agent=registry.example.com/my-agent:1.1.0

# Watch the rollout
$ kubectl rollout status deployment/my-agent

# Roll back if needed
$ kubectl rollout undo deployment/my-agent
```

Kubernetes handles the rolling update, keeping old pods running while new ones come up and pass health checks.

X> ## Exercise
X>
X> Deploy the Docker image from Chapter 17 to a local Kubernetes cluster (use minikube or kind if you do not have a cluster).
X>
X> 1. Generate Kubernetes manifests: `kdeps export k8s myagent-1.0.0.kdeps --image myagent:exercise --output k8s.yaml`. Read the generated file and locate the Deployment and Service resources.
X> 2. Add `agentSettings.resources` to `workflow.yaml` with: `cpuRequest: "100m"`, `memoryRequest: "128Mi"`, `cpuLimit: "500m"`, `memoryLimit: "512Mi"`. Regenerate and verify the `resources:` block appears in the Deployment.
X> 3. Create the auth Secret the generated Deployment references: `kubectl create secret generic myagent-auth --from-literal=api-token=testtoken`.
X> 4. Apply the manifests: `kubectl apply -f k8s.yaml`. Wait for the pod to be Running: `kubectl get pods -l app=myagent`.
X> 5. Port-forward and test: `kubectl port-forward svc/myagent 8080:16395` then `curl -X POST -H "Authorization: Bearer testtoken" localhost:8080/api/v1/chat -d '{"q":"hello"}'`.
X>
X> Inspect the generated probes with `kubectl describe pod` — readiness and liveness probes on `/health` are emitted automatically for apiServer workflows.
X>
X> **Stretch goal 1:** Scale to 2 replicas with `kubectl scale deployment/myagent --replicas=2`, send 20 rapid requests, and observe in `kubectl logs` that requests were distributed across both pods.
X>
X> **Stretch goal 2:** Regenerate with `--network-policy`, apply it, and verify with `kubectl describe networkpolicy myagent` that ingress is allowed only on your configured port.


## LLM appliance manifests

Agent manifests come from `kdeps export k8s` (this chapter). For an **inference-only** Deployment + Service with no workflow:

```bash
$ kdeps llm build --engine ollama --model llama3.2 --tag REG/llm:1
$ docker push REG/llm:1
$ kdeps llm export k8s --engine ollama --image REG/llm:1 --model llama3.2 -o llm.yaml
$ kubectl apply -f llm.yaml
```

Client hosts set `llm.backend: openai` and `llm.base_url` to the Service (see `kdeps llm client-config`). Full appliance guide: **Chapter 26**.

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
