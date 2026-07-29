# LLM server

Provision a **standalone inference appliance** — no agent, no workflow. Clients talk OpenAI-compatible **`/v1`**.

Agent packaging stays on `kdeps bundle …`. Inference appliances use **`kdeps llm`** only (no workflow path argument).

```text
recipe catalog -> kdeps llm build/run/export -> Docker | ISO | Kubernetes
                                              -> any OpenAI client (/v1)
```

## Wizard

```bash
kdeps llm          # TUI on a TTY
kdeps llm wizard
```

Pick engine → model → GPU → build / run / export.

## Common commands

```bash
kdeps llm list
kdeps llm models                 # llamafile + GGUF harvest
kdeps llm show ollama

kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm run   --engine ollama --model llama3.2 -p 8000

# GPU engines need --gpu
kdeps llm build --engine vllm --model meta-llama/Llama-3.2-1B-Instruct --gpu cuda

kdeps llm export k8s --image myorg/llm:1 --engine ollama -o llm.yaml
kdeps llm export iso --engine ollama --model llama3.2 -o llm.iso
```

## Client contract

Point any kdeps host (workflow or agent) at the appliance:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
```

```bash
export KDEPS_DEFAULT_BACKEND=openai
export KDEPS_LLM_BASE_URL=http://192.168.1.50:8000/v1
```

Generate a paste-ready snippet:

```bash
kdeps llm client-config --url http://192.168.1.50:8000/v1
kdeps llm client-config --url http://llm:8000/v1 --format export
```

## Stock recipes

| ID | Notes |
|----|--------|
| `ollama` | Local pull/serve |
| `llamafile` | Self-contained binary |
| `llama-server` / `gguf` | GGUF via llama-server |
| `llamacpp` | Official llama.cpp image |
| `vllm` / `tgi` / `sglang` | GPU required (`--gpu`) |
| `localai` | OpenAI drop-in |
| `openai-compat` | Template for custom images |

Add overrides in `~/.kdeps/llm-servers/*.yaml` or `./llm-servers/*.yaml` (project wins on same id).

Refresh the model harvest:

```bash
kdeps llamafile update
```

CLI details: [CLI](/cli).
