# Kubernetes deployment

`kdeps export k8s` generates a Kubernetes `Deployment` and `ClusterIP` `Service` from your `workflow.yaml`. You write no manifest by hand.

```bash
kdeps export k8s examples/chatbot --output k8s.yaml
kubectl apply -f k8s.yaml
```

## Read next

- [Kubernetes deployment reference](/reference/deploy/kubernetes) - every key, option and detail
- [Docker deployment](/deploy/docker) - build Docker images for your workflows
- [Standalone binaries](/deploy/binaries) - self-contained binaries for edge deployment
