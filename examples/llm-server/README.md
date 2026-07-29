# LLM server appliance example

Standalone OpenAI-compatible inference — no workflow path.

```bash
# List engines
kdeps llm list
kdeps llm show ollama

# Preview Dockerfile
kdeps llm build --engine ollama --model llama3.2 --show-dockerfile

# Build and run (requires Docker)
kdeps llm build --engine ollama --model llama3.2 --tag kdeps-llm-ollama:dev
kdeps llm run --engine ollama --model llama3.2 --tag kdeps-llm-ollama:dev --build=false -p 8000

# Kubernetes manifests
kdeps llm export k8s --engine ollama --image kdeps-llm-ollama:dev --model llama3.2 -o llm.yaml

# Client host config
kdeps llm client-config --url http://127.0.0.1:8000/v1
```

Custom recipe: copy `custom-recipe.yaml` to `./llm-servers/` or `~/.kdeps/llm-servers/`.

Docs: [LLM Server Appliance](../../docs/v2/deployment/llm-server.md)

## Stock engines

```bash
kdeps llm list
# ollama, llamafile, llama-server, gguf, llamacpp, vllm, tgi, sglang, localai, openai-compat
```

GPU engines (`vllm`, `tgi`, `sglang`) require `--gpu cuda` (or another profile) at build time.
