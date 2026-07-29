# LLM Server Appliance

Provision a **standalone LLM inference appliance** (no kdeps agent, no workflow) to Docker, ISO, or Kubernetes. Any kdeps host uses it as a client over **OpenAI-compatible `/v1`**.

Works for **both** workflow mode and agent mode clients — set `llm.backend: openai` and `llm.base_url` on the client machine.

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

Agent packaging (`kdeps bundle build` / `export iso` / `export k8s`) still deploys **agents**. `kdeps llm` deploys **only the inference server**.

## Commands

No workflow path argument — select engine and model only.

```bash
# List stock + user recipes
kdeps llm list

# Inspect a recipe
kdeps llm show ollama

# Print client config for a running appliance
kdeps llm client-config --url http://192.168.1.50:8000/v1
kdeps llm client-config --url http://llm:8000/v1 --format export
```

Build and export (see phases as they land):

```bash
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm run --engine ollama --model llama3.2 -p 8000
kdeps llm export k8s --image myorg/llm:1 --engine ollama -o llm.yaml
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
```

Or:

```bash
export KDEPS_DEFAULT_BACKEND=openai
export KDEPS_LLM_BASE_URL=http://192.168.1.50:8000/v1
```

Generate the same snippet with `kdeps llm client-config`.

Chat resources keep using `chat.model` as usual — only the backend URL changes.

## Recipe catalog

Stock recipes ship embedded:

| ID | Engine | Model strategy | Notes |
|----|--------|----------------|-------|
| `ollama` | Ollama | pull | Local models via Ollama |
| `llamafile` | Llamafile | copy | Self-contained `.llamafile` |
| `llama-server` | llama.cpp llama-server | copy | GGUF via kdeps llama-server path |
| `gguf` | llama-server (alias) | copy | Same as `llama-server` |
| `llamacpp` | llama.cpp server image | copy | Official llama.cpp server image |
| `vllm` | vLLM | pull | GPU, high throughput |
| `tgi` | Hugging Face TGI | pull | GPU |
| `sglang` | SGLang | pull | GPU |
| `localai` | LocalAI | pull | CPU/GPU OpenAI drop-in |
| `openai-compat` | custom | pull | Template for any OpenAI-compat image |

### Add or override a recipe

1. User: `~/.kdeps/llm-servers/<id>.yaml`
2. Project: `./llm-servers/<id>.yaml` (overrides user and stock for the same `id`)

Custom engines use `engine.kind: custom` with `install` + `command` + `api.*`. Clients still use `backend: openai`.

Minimal custom recipe:

```yaml
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
  kind: custom
  base_image: "vllm/vllm-openai:latest"
  command: ["python", "-m", "vllm.entrypoints.openai.api_server", "--host", "0.0.0.0", "--port", "8000"]
  openai_bridge: false
models:
  strategy: pull
resources:
  gpu: required
  memory_hint: "16Gi"
capabilities:
  - chat
```

## Auth

Default: no auth (private networks). Opt in with bearer auth via recipe `api.auth` and CLI flags when building.

## Related

- [LLM backends](/resources/llm-backends) — client-side backend selection
- [Docker deployment](/deployment/docker) — agent images (not appliances)
- [Kubernetes deployment](/deployment/kubernetes) — agent manifests
