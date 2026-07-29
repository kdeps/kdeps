# LLM Server Appliance

Provision a **standalone LLM inference appliance** (no kdeps agent, no workflow) to Docker, ISO, or Kubernetes. Any kdeps host uses it as a client over **OpenAI-compatible `/v1`**.

Works for **both** workflow mode and agent mode clients — set `llm.backend: openai` and `llm.base_url` on the client machine.

Agent packaging (`kdeps bundle build` / `export iso` / `export k8s`) still deploys **agents**. `kdeps llm` deploys **only the inference server**.

## How it fits

```d2
direction: right
catalog: Recipe catalog {
  shape: rectangle
}
build: kdeps llm build {
  shape: rectangle
}
targets: Deploy targets {
  docker: Docker
  iso: ISO
  k8s: Kubernetes
}
client: kdeps client {
  shape: rectangle
}
catalog -> build: "engine + model"
build -> targets.docker
build -> targets.iso
build -> targets.k8s
targets.docker -> client: "OpenAI /v1"
targets.iso -> client: "OpenAI /v1"
targets.k8s -> client: "OpenAI /v1"
```

## Commands

No workflow path argument — select engine and model only.

### Interactive wizard (TUI)

On a terminal, pick engine → harvest model → GPU → action (build/run/export):

```bash
kdeps llm wizard
# or bare:
kdeps llm
```

Uses the llamafile/GGUF harvest for `llamafile`, `llama-server` / `gguf`, and `llamacpp`. For `ollama` / `vllm` / `tgi` / `sglang` / `localai`, type a model id. Filter lists by typing; ↑/↓ navigate; enter select; esc cancel.

```bash
# List stock + user recipes
kdeps llm models   # harvest (llamafile + GGUF) available
kdeps llm list

# Inspect a recipe
kdeps llm show ollama
kdeps llm show vllm

# Print client config for a running appliance
kdeps llm client-config --url http://192.168.1.50:8000/v1
kdeps llm client-config --url http://llm:8000/v1 --format export
```

Build and export:

```bash
# Local / Docker
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm build --engine ollama --model llama3.2 --show-dockerfile
kdeps llm run --engine ollama --model llama3.2 -p 8000

# GPU engines (required for vllm / tgi / sglang)
kdeps llm build --engine vllm --model meta-llama/Llama-3.2-1B-Instruct --gpu cuda --tag myorg/vllm:1

# Kubernetes
kdeps llm export k8s --image myorg/llm:1 --engine ollama -o llm.yaml

# ISO / bootable (LinuxKit)
kdeps llm export iso --engine ollama --model llama3.2 --show-config
kdeps llm export iso --engine ollama --model llama3.2 -o llm.iso
kdeps llm export iso --engine ollama --model llama3.2 --config-only -o llm.yml
```

## Client contract

Every appliance exposes OpenAI-compatible chat completions. On the **client** kdeps host:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
  # openai_api_key: "..."   # if appliance auth enabled
  models:
    - llama3.2              # optional allowlist
```

Or:

```bash
export KDEPS_DEFAULT_BACKEND=openai
export KDEPS_LLM_BASE_URL=http://192.168.1.50:8000/v1
```

Generate the same snippet with `kdeps llm client-config`.

Chat resources keep using `chat.model` as usual — only the backend URL changes. Applies to **workflow mode** and **agent mode**.

## Stock recipes

Stock formulas ship embedded (`kdeps llm list`):

| ID | Engine | Model strategy | GPU | Notes |
|----|--------|----------------|-----|-------|
| `ollama` | Ollama | pull | optional | Local models via Ollama; native `/v1` |
| `llamafile` | Llamafile | copy | optional | Self-contained `.llamafile` |
| `llama-server` | llama.cpp llama-server | copy | optional | GGUF via kdeps llama-server path |
| `gguf` | llama-server (alias) | copy | optional | Same as `llama-server` |
| `llamacpp` | llama.cpp server image | copy | optional | Official llama.cpp server image |
| `vllm` | vLLM | pull | **required** | High throughput; pass `--gpu cuda` |
| `tgi` | Hugging Face TGI | pull | **required** | GPU; health on `/health` |
| `sglang` | SGLang | pull | **required** | GPU serving runtime |
| `localai` | LocalAI | pull | optional | OpenAI drop-in (CPU/GPU images) |
| `openai-compat` | custom template | pull | optional | Fork for any OpenAI-compat image |

### Add or override a recipe

1. User: `~/.kdeps/llm-servers/<id>.yaml`
2. Project: `./llm-servers/<id>.yaml` (overrides user and stock for the same `id`)

Custom engines use `engine.kind: custom` (or a stock kind) with `install` + `command` + `api.*`. Clients still use `backend: openai`.

```yaml
# ./llm-servers/my-vllm.yaml
id: my-vllm
name: My vLLM
description: vLLM OpenAI-compat server
version: "1"
api:
  port: 8000
  base_path: /v1
  chat_path: /v1/chat/completions
  health:
    method: GET
    path: /v1/models
  auth:
    mode: none
engine:
  kind: vllm
  base_image: "vllm/vllm-openai:latest"
  command:
    - sh
    - -c
    - exec python3 -m vllm.entrypoints.openai.api_server --host 0.0.0.0 --port 8000 --model "${LLM_MODEL}"
  openai_bridge: false
models:
  strategy: pull
resources:
  gpu: required
  memory_hint: "16Gi"
capabilities:
  - chat
```

## Docker

```bash
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
docker run --rm -p 8000:8000 myorg/llm:1
```

Preview without building:

```bash
kdeps llm build --engine ollama --model llama3.2 --show-dockerfile
```

GPU profiles:

```bash
kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --tag myorg/vllm:1
```

Optional auth fail-fast at container start:

```bash
kdeps llm build --engine ollama --model llama3.2 --require-auth --api-key-env LLM_API_KEY
```

Example compose: `examples/llm-server/docker-compose.yml`.

## Kubernetes

```bash
kdeps llm build --engine ollama --model llama3.2 --tag REG/llm:1
docker push REG/llm:1
kdeps llm export k8s --engine ollama --image REG/llm:1 --model llama3.2 -o llm.yaml
kubectl apply -f llm.yaml
```

Manifests include Deployment, Service (ClusterIP), readiness/liveness on the recipe health path, and optional `--api-key-secret`.

```bash
kdeps llm client-config --url http://kdeps-llm-ollama:8000/v1 --model llama3.2
```

## ISO / bootable image

```bash
kdeps llm export iso --engine ollama --model llama3.2 --show-config
kdeps llm export iso --engine ollama --model llama3.2 --config-only -o llm.yml
kdeps llm export iso --engine ollama --model llama3.2 -o llm.iso
```

Requires Docker and linuxkit (auto-downloaded, same tooling family as agent `export iso`). Formats: `iso` (default), `qcow2`.

## Auth

Default: no auth (private networks — document the risk). Opt in with recipe `api.auth` and CLI `--require-auth` / `--api-key-env`. On Kubernetes, put the key in a Secret (`--api-key-secret`).

## Related

- [LLM backends](/resources/llm-backends) — client-side backend selection
- [LLM Commands](/reference/cli/llm) — full CLI flag reference
- [Docker deployment](/deployment/docker) — **agent** images (not appliances)
- [Kubernetes deployment](/deployment/kubernetes) — **agent** manifests
- Example project: `examples/llm-server/`
