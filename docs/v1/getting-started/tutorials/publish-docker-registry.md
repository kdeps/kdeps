---
outline: deep
---

# Publish your KDeps image to a Docker registry

This guide shows how to push a local Docker image to a registry (e.g., Docker Hub) so you can run it on services like Google Cloud Run or cloud providers that offer GPUs.

## Prerequisites

- You have built a local image (example: `kdeps-whois:latest`)
- You have a Docker Hub account (example namespace: `jjuliano`)

## 1) Log in to Docker Hub

```bash
docker login
```

Enter your Docker Hub username and password (or a personal access token).

## 2) Tag your local image

Assuming your local image is `kdeps-whois:latest`:

```bash
docker tag kdeps-whois:latest jjuliano/kdeps-whois:latest
```

If you’re unsure about tags, list images with:

```bash
docker images
```

## 3) Push the image

```bash
docker push jjuliano/kdeps-whois:latest
```

## 4) Verify on Docker Hub

Open your repositories page to verify:

`https://hub.docker.com/repositories/<your-username>`

Example: `https://hub.docker.com/repositories/jjuliano`

## Optional: Use versioned tags

```bash
docker tag kdeps-whois:latest jjuliano/kdeps-whois:v1.0.0
docker push jjuliano/kdeps-whois:v1.0.0
```

GPU notes:

- For NVIDIA GPUs, use the `*_docker-compose-nvidia.yaml` and ensure NVIDIA drivers/container toolkit are installed.
- For AMD GPUs, use the `*_docker-compose-amd.yaml` VM images/drivers that expose `/dev/kfd` and `/dev/dri`.

## Deploy Docker Compose to Google Cloud Run

If you’re part of the Cloud Run Compose (private preview), you can deploy your Docker Compose configuration directly with a single command.

1) Rename the generated Compose file to `compose.yaml`

```bash
# Example generated file: whois_docker-compose-nvidia.yaml
cp whois_docker-compose-nvidia.yaml compose.yaml
```

2) Update the image reference to your registry

Edit `compose.yaml` and change the `image:` to either Docker Hub or Artifact Registry:

```yaml
services:
  whois-nvidia:
    image: jjuliano/kdeps-whois:latest  # Docker Hub example
    # image: us-central1-docker.pkg.dev/PROJECT_ID/kdeps-repo/kdeps-whois:latest  # Artifact Registry example
```

3) Deploy using gcloud

```bash
gcloud run compose up
```

Tips:

- Pick the variant that matches your hardware needs before renaming (CPU, NVIDIA, AMD).
- Keep external volumes (like `ollama`, `kdeps`) defined in the compose file if supported in your preview setup.

## Deploy a single container to Cloud Run (normal path)

If you’re not in the Compose private preview, deploy the single container image referenced in your compose file.

1) Enable Cloud Run API

```bash
gcloud services enable run.googleapis.com
```

2) Identify the image from your compose

Open your compose (or generated compose variant) and copy the `image:` reference you pushed (Docker Hub or Artifact Registry):

```yaml
services:
  whois-cpu:
    image: jjuliano/kdeps-whois:latest
```

3) Deploy the image to Cloud Run

```bash
gcloud run deploy kdeps-whois \
  --image=jjuliano/kdeps-whois:latest \
  --region=us-central1 \
  --platform=managed \
  --allow-unauthenticated \
  --port=16395 \
  --memory=8Gi \
  --cpu=2 \
  --min-instances=1 \
  --timeout=600
```

Notes:

- If your compose exposes both API and Web ports, deploy each as a separate Cloud Run service (one per container) or use the Compose preview to keep them together.

