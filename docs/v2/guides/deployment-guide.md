# Deployment Guide

End-to-end CI/CD pipeline: package your workflow, build a Docker image, push to a registry, and deploy to Kubernetes.

## Overview

```d2
direction: right

A: "1. Validate\nkdeps validate\nworkflow.yaml"
B: "2. Package\nkdeps bundle\npackage"
C: "3. Build\nkdeps bundle\nbuild + docker push"
D: "4. Deploy\nkubectl\napply"

A -> B -> C -> D
```

Each step is a single `kdeps` command. No Dockerfiles, no manual YAML authoring, no glue scripts.

## Step 1: Validate

Run schema and dependency validation before packaging:

```bash
kdeps validate workflow.yaml
```

This catches YAML syntax errors, missing dependencies, circular references, and bad expressions before they reach production. Always run this in CI before packaging.

## Step 2: Package

Create a portable `.kdeps` archive containing the workflow and all resources:

```bash
kdeps bundle package . --output dist/
# Creates: dist/my-agent-1.0.0.kdeps
```

The archive includes `workflow.yaml`, all resource files, Python requirements, data files, and assets. Respects `.kdepsignore` exclusions.

## Step 3: Build Docker Image

Build a Docker image from the package:

```bash
kdeps bundle build dist/my-agent-1.0.0.kdeps \
  --tag registry.example.com/my-agent:v1.0.0
docker push registry.example.com/my-agent:v1.0.0
```

No Dockerfile needed -- kdeps generates a multi-stage build from your workflow config. GPU support is a flag away:

```bash
kdeps bundle build dist/my-agent-1.0.0.kdeps \
  --tag registry.example.com/my-agent:v1.0.0-gpu \
  --gpu cuda
docker push registry.example.com/my-agent:v1.0.0-gpu
```

See [Docker Deployment](/deployment/docker) for base OS selection, offline mode, and custom image configuration.

## Step 4: Deploy to Kubernetes

Generate Kubernetes manifests and apply them:

```bash
kdeps export k8s dist/my-agent-1.0.0.kdeps \
  --image registry.example.com/my-agent:v1.0.0 \
  --output k8s.yaml

kubectl apply -f k8s.yaml
kubectl rollout status deployment/my-agent
```

The generated manifests include Deployment, Service, and environment configuration -- all driven from `workflow.yaml`. Override replicas, resource limits, and env vars with flags.

See [Kubernetes Deployment](/deployment/kubernetes) for full manifest structure, health checks, and multi-replica configuration.

## CI/CD Pipeline Example

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    tags: ['v*']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install kdeps
        run: curl -fsSL https://kdeps.io/install.sh | bash

      - name: Validate
        run: kdeps validate workflow.yaml

      - name: Package
        run: kdeps bundle package . --output dist/

      - name: Build and Push
        run: |
          kdeps bundle build dist/*.kdeps \
            --tag ${{ secrets.REGISTRY }}/my-agent:${{ github.ref_name }}
          docker push ${{ secrets.REGISTRY }}/my-agent:${{ github.ref_name }}
        env:
          DOCKER_CONFIG: ${{ secrets.DOCKER_CONFIG }}

      - name: Deploy to K8s
        run: |
          kdeps export k8s dist/*.kdeps \
            --image ${{ secrets.REGISTRY }}/my-agent:${{ github.ref_name }} \
            --output k8s.yaml
          kubectl apply -f k8s.yaml
```

### GitLab CI

```yaml
# docker-compose.yml
deploy:
  stage: deploy
  only:
    - tags
  script:
    - curl -fsSL https://kdeps.io/install.sh | bash
    - kdeps validate workflow.yaml
    - kdeps bundle package . --output dist/
    - kdeps bundle build dist/*.kdeps --tag $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG
    - kdeps export k8s dist/*.kdeps --image $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG --output k8s.yaml
    - kubectl apply -f k8s.yaml
```

## Standalone Binaries (No Docker)

For edge deployments that can't run containers, use the prepackage flow:

```bash
kdeps bundle package . --output dist/
```

The `.kdeps` archive can be deployed directly on any machine with kdeps installed:

```bash
export KDEPS_API_AUTH_TOKEN=api-secret
kdeps run dist/my-agent-1.0.0.kdeps --port 16395
```

See [Standalone Binaries](/deployment/prepackage) for self-contained single-binary exports.

## See Also

- [Docker Deployment](/deployment/docker) -- image build details, base OS, GPU support
- [Kubernetes Deployment](/deployment/kubernetes) -- manifest structure, health checks
- [Standalone Binaries](/deployment/prepackage) -- single-binary edge exports
- [CLI: Packaging Commands](/reference/cli/packaging) -- all bundle and export commands
