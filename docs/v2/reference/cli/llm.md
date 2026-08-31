# LLM commands

`kdeps llm` provisions **standalone LLM server appliances**. These are not agent packages - there is **no workflow path argument**.

Architecture, stock recipes, and client wiring: [LLM server appliance](/deployment/llm-server).

## Stock engines

```bash
kdeps llm list
```

| ID | Typical use |
|----|-------------|
| `ollama` | Local Ollama pull/serve |
| `llamafile` | Self-contained llamafile binary |
| `llama-server` / `gguf` | GGUF via llama-server |
| `llamacpp` | Official llama.cpp server image |
| `vllm` | GPU high-throughput (requires `--gpu`) |
| `tgi` | Hugging Face TGI (requires `--gpu`) |
| `sglang` | SGLang (requires `--gpu`) |
| `localai` | LocalAI OpenAI drop-in |
| `openai-compat` | Template for any OpenAI-compat image |

Override or add recipes in `~/.kdeps/llm-servers/*.yaml` or `./llm-servers/*.yaml`.

## kdeps llm wizard

Interactive TUI to select engine, model (from harvest or typed), GPU, and action.

```bash
kdeps llm wizard
kdeps llm   # same on a TTY
```

Requires a TTY. Non-interactive environments should use the flag-based commands below.

## kdeps llm models

List models available from the llamafile/GGUF harvest (what the wizard shows when picking a model).

```bash
kdeps llm models
kdeps llm models --type gguf
kdeps llm models --type llamafile
kdeps llamafile update   # refresh harvest (GGUF + Chinese labs: Qwen, DeepSeek, Yi, ...)
```

## kdeps llm list

List stock and user/project recipes.

```bash
kdeps llm list
```

## kdeps llm show

```bash
kdeps llm show ollama
kdeps llm show vllm
```

Prints ports, health path, model strategy, engine command, and a `client-config` example.

## kdeps llm client-config

Print a ready-to-paste `~/.kdeps/config.yaml` snippet (or shell env) that points kdeps at an appliance over OpenAI-compatible `/v1`.

| Flag | Description |
|------|-------------|
| `--url` | OpenAI-compat base URL (required), e.g. `http://host:8000/v1` |
| `--api-key` | Optional bearer key |
| `--model` | Optional model allowlist (yaml only) |
| `--format` | `yaml` (default), `env`, or `export` |

```bash
kdeps llm client-config --url http://192.168.1.50:8000/v1
kdeps llm client-config --url http://llm:8000/v1 --format export
kdeps llm client-config --url http://host:8000/v1 --model llama3.2 --api-key secret
```

## kdeps llm build

Build a Docker image for an appliance.

| Flag | Description |
|------|-------------|
| `--engine` | Recipe id (required) |
| `--model` | Model name / HF id / path |
| `--tag` | Image tag (default `kdeps-llm-<engine>:latest`) |
| `--gpu` | `cuda` \| `rocm` \| `intel` \| `vulkan` (required when recipe `resources.gpu: required`) |
| `--show-dockerfile` | Print Dockerfile + entrypoint; do not build |
| `--pull-at-build` | Materialize models during image build |
| `--api-key-env` | Env var name for bearer API key |
| `--require-auth` | Fail container start if API key env empty |
| `--no-client-hint` | Skip printing client-config after build |

```bash
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --tag myorg/vllm:1
kdeps llm build --engine ollama --model llama3.2 --show-dockerfile
```

## kdeps llm run

Build (optional) and run the appliance with Docker on the local daemon.

| Flag | Description |
|------|-------------|
| `--engine` | Recipe id (required) |
| `--model` | Model name |
| `--tag` | Image tag |
| `-p` / `--port` | Host port (default: recipe `api.port`) |
| `-d` / `--detach` | Background container |
| `--build` | Build before run (default true) |
| `--gpu` | GPU profile for build |

```bash
kdeps llm run --engine ollama --model llama3.2 -p 8000
kdeps llm run --engine ollama --model llama3.2 --tag myorg/llm:1 --build=false
```

## kdeps llm export k8s

Generate Deployment + Service YAML for a pre-built appliance image (not agent `export k8s`).

| Flag | Description |
|------|-------------|
| `--engine` | Recipe id (required) |
| `--image` | Container image (required) |
| `--model` | Model env on the pod |
| `--name` | Resource name (default `kdeps-llm-<engine>`) |
| `--replicas` | Replicas (default 1) |
| `-o` / `--output` | Write file (default stdout) |
| `--api-key-secret` | Secret name with key `api-key` → `LLM_API_KEY` |
| `--no-client-hint` | Skip client-config hint |

```bash
kdeps llm export k8s --engine ollama --image REG/llm:1 --model llama3.2 -o llm.yaml
kubectl apply -f llm.yaml
```

## kdeps llm export iso

Build (optional) Docker image and assemble LinuxKit config / bootable image.

| Flag | Description |
|------|-------------|
| `--engine` | Recipe id (required) |
| `--model` | Model name |
| `--tag` | Docker image tag |
| `-o` / `--output` | Output path for `.iso` / `.qcow2` or YAML when `--config-only` |
| `--format` | `iso` (default) or `qcow2` |
| `--arch` | Architecture |
| `--hostname` | Hostname |
| `--size` | LinuxKit disk size (e.g. `8192M`) |
| `--show-config` | Print LinuxKit YAML and exit |
| `--config-only` | Write LinuxKit YAML only (no linuxkit build) |
| `--skip-build` | Assume `--tag` image already exists |
| `--gpu` | GPU profile for image build |

```bash
kdeps llm export iso --engine ollama --model llama3.2 --show-config
kdeps llm export iso --engine ollama --model llama3.2 --config-only -o llm.yml
kdeps llm export iso --engine ollama --model llama3.2 -o llm.iso
```

## See also

- [LLM server appliance](/deployment/llm-server)
- [LLM backends](/resources/llm-backends)
- [Packaging commands](/reference/cli/packaging) - agent bundle/export (different product surface)
