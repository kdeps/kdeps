# Kubernetes deployment

`kdeps export k8s` writes Kubernetes manifests from your `workflow.yaml`. An agent with an API scales to zero by default: a request starts a pod, and the pod stops when it is idle.

```yaml
# workflow.yaml
deploy:
  scale:
    min: 0            # default; the first request wakes a pod (Knative)
    max: 5
    idle: 5m          # keep the last pod 5m after the last request
```

```bash
kdeps export k8s examples/chatbot --output k8s.yaml
kubectl apply -f k8s.yaml
```

## Read next

- [Kubernetes deployment reference](/reference/deploy/kubernetes) - every key, option and detail
- [Docker deployment](/deploy/docker) - build Docker images for your workflows
- [Standalone binaries](/deploy/binaries) - self-contained binaries for edge deployment
