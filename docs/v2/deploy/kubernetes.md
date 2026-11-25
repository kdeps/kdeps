# Kubernetes deployment

`kdeps export k8s` generates a Kubernetes `Deployment` and `ClusterIP` `Service` from your `workflow.yaml`. You write no manifest by hand.

```bash
kdeps export k8s examples/chatbot --output k8s.yaml
kubectl apply -f k8s.yaml
```

```yaml
# workflow.yaml
deploy:
  runtime:
    replicas: 3              # pod replicas
    networkPolicy: true      # opt-in: restrict pod ingress to the server ports
```

## Read next

- [Kubernetes deployment reference](/reference/deploy/kubernetes) - quick start, command reference, workflow configuration, generated manifests, deployment, service, networkPolicy (opt-in), auth tokens
- [Docker deployment](/deploy/docker) - build Docker images for your workflows
- [Standalone binaries](/deploy/binaries) - self-contained binaries for edge deployment
- [LLM server appliance](/llm-server/) - inference-only deployment
- [CLI reference](/reference/cli) - full command reference
