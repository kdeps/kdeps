# LLM server appliance example

Standalone OpenAI-compatible inference — no workflow path.

## Quick start

```bash
# Interactive TUI (engine + harvest model + build)
kdeps llm wizard

# List stock engines (ollama, llamafile, gguf, vllm, tgi, sglang, localai, ...)
kdeps llm list
kdeps llm show ollama
kdeps llm show vllm

# Preview Dockerfile
kdeps llm build --engine ollama --model llama3.2 --show-dockerfile
kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --show-dockerfile

# Build and run (requires Docker)
kdeps llm build --engine ollama --model llama3.2 --tag kdeps-llm-ollama:dev
kdeps llm run --engine ollama --model llama3.2 --tag kdeps-llm-ollama:dev --build=false -p 8000

# Kubernetes manifests
kdeps llm export k8s --engine ollama --image kdeps-llm-ollama:dev --model llama3.2 -o llm.yaml

# Client host config
kdeps llm client-config --url http://127.0.0.1:8000/v1
```

## Stock engines

| ID | Notes |
|----|--------|
| `ollama` | Local pull/serve |
| `llamafile` | Self-contained binary |
| `llama-server` / `gguf` | GGUF weights |
| `llamacpp` | Official llama.cpp server image |
| `vllm` / `tgi` / `sglang` | GPU — pass `--gpu cuda` |
| `localai` | OpenAI drop-in |
| `openai-compat` | Template for any `/v1` image |

Custom recipe: copy `custom-recipe.yaml` to `./llm-servers/` or `~/.kdeps/llm-servers/`.

Docs: [LLM Server Appliance](../../docs/v2/deployment/llm-server.md) · [CLI](../../docs/v2/reference/cli/llm.md)
