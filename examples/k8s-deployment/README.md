# Kubernetes Deployment Example

This example demonstrates how to use `kdeps` to generate Kubernetes manifests for your AI workflows.

## Workflow Configuration

The `workflow.yaml` file includes several Kubernetes-specific settings:

- **Scale:** `scale` (`min: 0`, `max: 3`, `idle: 5m`) writes a Knative Service. The agent scales to zero, and the first request starts a pod. Set `engine: keda` for KEDA, or `min` equal to `max` for a fixed count.
- **Resource Limits:** `resources` block specifies CPU and Memory limits and requests.
- **Environment Variables:** `env` block is automatically mapped to Kubernetes container environment variables.
- **Ports:** `portNum` and `apiServerMode: true` are used to generate a Kubernetes Service.
- **Health Checks:** `kdeps` automatically generates readiness and liveness probes based on your API port.

## How to Export Manifests

Generate the Kubernetes manifests:

```bash
kdeps export k8s .
```

### Options

- **Specify Image:** Use `--image` to set the container image to use in the manifest.
  ```bash
  kdeps export k8s . --image my-registry/k8s-example:1.0.0
  ```
- **Save to File:** Use `--output` to save the manifests to a file.
  ```bash
  kdeps export k8s . --output deployment.yaml
  ```

## Deploying to Kubernetes

Once you have the YAML, you can deploy it using `kubectl`:

```bash
kdeps export k8s . --output deployment.yaml
kubectl apply -f deployment.yaml
```
