# Llamafile Chat Example

LLM chatbot using the `file` backend (the kdeps default) - runs a
[llamafile](https://github.com/Mozilla-Ocho/llamafile) binary as a local
OpenAI-compatible server. No GPU, cloud API key, or server install required.

## What is llamafile?

A llamafile is a single self-contained binary that bundles a GGUF model with the llama.cpp runtime.
It runs on Linux (x86_64/ARM64), macOS (ARM64/x86_64), and Windows without any per-platform build.

## Quick Start

```bash
kdeps run examples/llamafile-chat/workflow.yaml
```

That's it. The `model: llama3.2:1b` alias resolves to Mozilla's
Llama 3.2 1B Instruct llamafile (Q4_K_M, ~1.1 GB), which is downloaded to
`~/.kdeps/models/` on first run and reused afterwards.

### Chat

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"q": "What is artificial intelligence?"}'
```

## Response

```json
{
  "data": {
    "answer": "Artificial intelligence (AI) is..."
  },
  "query": "What is artificial intelligence?"
}
```

## Structure

```
llamafile-chat/
├── workflow.yaml
└── resources/
    ├── llm.yaml        # file backend chat resource
    └── response.yaml   # API response resource
```

## Model aliases

Known aliases map to Mozilla's HuggingFace llamafiles. Quantization is part
of the alias name so you can trade size for quality:

| Alias | Model | Quant | Size |
|-------|-------|-------|------|
| `llama3.2` / `llama3.2:1b` | Llama 3.2 1B Instruct | Q4_K_M | ~1.1 GB |
| `llama3.2:1b-q6` | Llama 3.2 1B Instruct | Q6_K | ~1.5 GB |
| `llama3.2:1b-q8` | Llama 3.2 1B Instruct | Q8_0 | ~2.1 GB |
| `llama3.2:3b` | Llama 3.2 3B Instruct | Q4_K_M | ~2.2 GB |
| `llama3.2:3b-q6` | Llama 3.2 3B Instruct | Q6_K | ~2.9 GB |

List everything the registry knows (and refresh it from HuggingFace):

```bash
kdeps llamafile list
kdeps llamafile update
```

## Model field formats

| Format | Example | Behaviour |
|--------|---------|-----------|
| Registry alias | `llama3.2:1b` | Resolved to a known URL, downloaded and cached |
| Remote URL | `https://hf.co/.../model.llamafile` | Downloaded and cached in `~/.kdeps/models/` |
| Absolute path | `/home/user/models/model.llamafile` | Used directly |
| Relative path | `./models/model.llamafile` | Resolved from cwd |
| Bare filename | `model.llamafile` | Looked up in `~/.kdeps/models/` |

## Custom cache directory

Set `models_dir` in `~/.kdeps/config.yaml` to override the default cache location:

```yaml
llm:
  models_dir: /data/models
```

Or use the `KDEPS_MODELS_DIR` environment variable.
