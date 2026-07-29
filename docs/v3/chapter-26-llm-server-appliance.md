# Chapter 26: LLM Server Appliance

Chapters 17–19 deploy **agents** — packages that contain `workflow.yaml`, the kdeps runtime, and optionally a local model sidecar. This chapter is different.

`kdeps llm` provisions a **standalone LLM inference appliance**: a long-lived OpenAI-compatible server with no agent, no workflow path, and no DAG. Any kdeps host (workflow mode or agent mode) points at it with `backend: openai` and `base_url`.

```text
Recipe catalog  ->  kdeps llm build  ->  Docker image
                                      ->  K8s Deployment + Service
                                      ->  ISO / qcow2 (LinuxKit)
                                              |
                                              v
                              kdeps client (any machine)
                              llm.backend: openai
                              llm.base_url: http://HOST:8000/v1
```

**Modes:** the appliance is infrastructure for **both** workflow mode and agent mode. The server does not run workflows; clients do.

## Why a Separate Command

| Agent deploy (`bundle` / `export`) | LLM appliance (`kdeps llm`) |
|------------------------------------|-----------------------------|
| Requires workflow / package path | **No workflow path** — engine + model only |
| Runs kdeps + optional Ollama | Runs only the inference engine |
| API/web ports for the agent | Single OpenAI-compatible `/v1` surface |
| Client is HTTP callers of your agent | Client is **kdeps itself** (or any OpenAI client) |

## Commands

### Interactive wizard

```bash
$ kdeps llm wizard
$ kdeps llm          # same on a TTY
```

Select engine → **browse available harvest models** (llamafile + GGUF, with size/quant) or type an Ollama/HF id → GPU → build/run/export. Filter by typing; ↑/↓; enter; esc.

```bash
# List stock + user recipes
$ kdeps llm models   # show harvest (llamafile + GGUF) available
$ kdeps llm list

# Inspect ports, model strategy, client contract
$ kdeps llm show ollama
$ kdeps llm show vllm

# Print client config for a running appliance
$ kdeps llm client-config --url http://192.168.1.50:8000/v1
$ kdeps llm client-config --url http://llm:8000/v1 --format export
```

Build and export:

```bash
$ kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
$ kdeps llm build --engine ollama --model llama3.2 --show-dockerfile
$ kdeps llm run --engine ollama --model llama3.2 -p 8000

# GPU engines require --gpu
$ kdeps llm build --engine vllm --model meta-llama/Llama-3.2-1B-Instruct --gpu cuda --tag myorg/vllm:1

$ kdeps llm export k8s --image myorg/llm:1 --engine ollama -o llm.yaml
$ kdeps llm export iso --engine ollama --model llama3.2 --show-config
$ kdeps llm export iso --engine ollama --model llama3.2 -o llm.iso
$ kdeps llm export iso --engine ollama --model llama3.2 --config-only -o llm.yml
```

None of these take a `workflow.yaml` argument.

## Client Contract

Every appliance exposes OpenAI-compatible chat completions. On the **client** kdeps host:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
  # openai_api_key: "..."   # if appliance auth enabled
```

Or environment variables:

```bash
$ export KDEPS_DEFAULT_BACKEND=openai
$ export KDEPS_LLM_BASE_URL=http://192.168.1.50:8000/v1
```

Generate the same snippet with `kdeps llm client-config`. Chat resources keep using `chat.model` as usual — only the backend URL changes.

This is the same remote-OpenAI path described in Chapter 6 under “Any OpenAI-Compatible Endpoint,” except you own the server.

## Stock Recipes (Default Formulas)

Stock formulas ship embedded. Run `kdeps llm list` to see the live catalog.

| ID | Engine | Model strategy | GPU | Notes |
|----|--------|----------------|-----|-------|
| `ollama` | Ollama | pull | optional | Local models via Ollama |
| `llamafile` | Llamafile | copy | optional | Self-contained `.llamafile` |
| `llama-server` | llama.cpp llama-server | copy | optional | GGUF via kdeps path |
| `gguf` | llama-server (alias) | copy | optional | Same as `llama-server` |
| `llamacpp` | llama.cpp server image | copy | optional | Official llama.cpp image |
| `vllm` | vLLM | pull | **required** | High throughput; `--gpu cuda` |
| `tgi` | Hugging Face TGI | pull | **required** | GPU |
| `sglang` | SGLang | pull | **required** | GPU |
| `localai` | LocalAI | pull | optional | OpenAI drop-in |
| `openai-compat` | custom template | pull | optional | Fork for any `/v1` image |

### Override or add recipes

1. User: `~/.kdeps/llm-servers/<id>.yaml`
2. Project: `./llm-servers/<id>.yaml` (overrides user and stock for the same `id`)

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
$ kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
$ docker run --rm -p 8000:8000 myorg/llm:1
```

Preview without building:

```bash
$ kdeps llm build --engine ollama --model llama3.2 --show-dockerfile
```

GPU profiles (required when the recipe sets `resources.gpu: required`):

```bash
$ kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --tag myorg/vllm:cuda
```

Optional auth fail-fast:

```bash
$ kdeps llm build --engine ollama --model llama3.2 --require-auth --api-key-env LLM_API_KEY
```

Compose example: `examples/llm-server/docker-compose.yml`.

## Kubernetes

```bash
$ kdeps llm build --engine ollama --model llama3.2 --tag REG/llm:1
$ docker push REG/llm:1
$ kdeps llm export k8s --engine ollama --image REG/llm:1 --model llama3.2 -o llm.yaml
$ kubectl apply -f llm.yaml
```

On a client host inside the cluster:

```bash
$ kdeps llm client-config --url http://kdeps-llm-ollama:8000/v1 --model llama3.2
```

## ISO / Bootable Image

```bash
$ kdeps llm export iso --engine ollama --model llama3.2 --show-config
$ kdeps llm export iso --engine ollama --model llama3.2 --config-only -o llm.yml
$ kdeps llm export iso --engine ollama --model llama3.2 -o llm.iso
```

Default format is `iso` (linuxkit `iso-efi`). Use `--format qcow2` for qcow2. After boot, point clients at `http://HOST:8000/v1`.

## End-to-End Walkthrough

1. **Provision** the appliance:

```bash
$ kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
$ docker run -d --name kdeps-llm -p 8000:8000 myorg/llm:1
```

2. **Configure** a second machine that runs agents:

```bash
$ kdeps llm client-config --url http://192.168.1.50:8000/v1 --model llama3.2
# Paste into ~/.kdeps/config.yaml
```

3. **Run** a workflow or agent as usual — chat resources use `model: llama3.2` and hit the remote appliance.

## Auth and Security

- Default: no auth (assume private network). Document the risk in production.
- Opt in: recipe `api.auth` and CLI `--require-auth` / `--api-key-env`.
- K8s: put the key in a Secret (`--api-key-secret`); never commit keys into recipe YAML.

## What This Is Not

- Not a replacement for in-process local backends when you run `kdeps run` with `backend: file` / `ollama` on the same machine.
- Not a multi-model router inside the appliance (use client-side `llm.strategy` / router from Chapter 6).
- Not an agent package: do not put `workflow.yaml` on these commands.

## Related Chapters

- Chapter 6 — LLM resources and remote OpenAI-compatible backends
- Chapter 17 — Docker **agent** images (`bundle build`)
- Chapter 18 — Kubernetes **agent** manifests (`export k8s`)
- Chapter 19 — Standalone **agent** binaries (`prepackage`)

Canonical docs: `docs/v2/deployment/llm-server.md`, `docs/v2/reference/cli/llm.md`.

X>
## Exercise

X>
X> Provision a local OpenAI-compatible appliance and point a workflow at it.
X>
X> 1. Run `kdeps llm list` and confirm stock engines include `ollama`, `llamafile`, `gguf`, `vllm`, `tgi`, `sglang`, and `localai`.
X> 2. `kdeps llm show ollama` and `kdeps llm show vllm`. Note GPU requirements on vLLM.
X> 3. Preview images: `kdeps llm build --engine ollama --model llama3.2 --show-dockerfile` and `kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --show-dockerfile`.
X> 4. Generate client config: `kdeps llm client-config --url http://127.0.0.1:8000/v1 --format yaml`.
X> 5. Export K8s YAML: `kdeps llm export k8s --engine ollama --image kdeps-llm-ex:1 -o /tmp/llm.yaml` and inspect probes and ports.
X>
X> **Stretch goal:** add `./llm-servers/custom.yaml` with `engine.kind: custom` pointing at any OpenAI-compat image; confirm `kdeps llm list` shows it as `project` source.
X>
