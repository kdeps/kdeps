# Llamafile Chat Example

LLM chatbot using the `file` backend - runs a [llamafile](https://github.com/Mozilla-Ocho/llamafile)
binary as a local OpenAI-compatible server. No GPU or cloud API key required.

## What is llamafile?

A llamafile is a single self-contained binary that bundles a GGUF model with the llama.cpp runtime.
It runs on Linux (x86_64/ARM64), macOS (ARM64/x86_64), and Windows without any per-platform build.

## Quick Start

### 1. Get a llamafile

Download a pre-built llamafile from Mozilla's HuggingFace collection, e.g.:

```bash
# Download TinyLlama (637 MB) - fast and small
wget https://huggingface.co/Mozilla/TinyLlama-1.1B-Chat-v1.0-llamafile/resolve/main/TinyLlama-1.1B-Chat-v1.0.Q5_K_M.llamafile \
     -O ~/.kdeps/models/tinyllama.llamafile
chmod +x ~/.kdeps/models/tinyllama.llamafile
```

Or use a URL directly as the `model` field - kdeps downloads and caches it automatically.

### 2. Run the workflow

```bash
# Point to your llamafile (bare filename in ~/.kdeps/models/)
LLAMAFILE_MODEL=tinyllama.llamafile kdeps run examples/llamafile-chat/workflow.yaml

# Or use a full path
LLAMAFILE_MODEL=/path/to/model.llamafile kdeps run examples/llamafile-chat/workflow.yaml

# Or use a remote URL (auto-downloaded on first run)
LLAMAFILE_MODEL=https://huggingface.co/Mozilla/TinyLlama-1.1B-Chat-v1.0-llamafile/resolve/main/TinyLlama-1.1B-Chat-v1.0.Q5_K_M.llamafile \
  kdeps run examples/llamafile-chat/workflow.yaml
```

### 3. Chat

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

## Model field formats

| Format | Example | Behaviour |
|--------|---------|-----------|
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
