---
title: Local Models (Llamafile & Ollama)
description: Run kdeps entirely offline with local models. Your code and prompts never leave your machine.
---

# Local Models (Llamafile & Ollama)

kdeps can run entirely offline. When you use a local model backend, nothing is sent to external APIs - your prompts, code, and responses stay on your machine.

Two local backends are supported: **llamafile** (the default, zero-install) and **Ollama** (model manager with a broader catalog).

---

## Llamafile (default backend)

Llamafile is a model + server packaged into a single self-contained binary. It runs on any OS - Mac, Linux, Windows - without a GPU requirement, no server to install separately.

kdeps uses llamafile by default. The first time you run a `chat:` resource with no backend configured, kdeps downloads the model automatically and caches it in `~/.kdeps/models/`.

**Start kdeps with the default llamafile model:**

```bash
kdeps
# downloads llama3.2:1b (~1.1 GB) on first run, then starts the REPL
```

**Pick a different model by alias:**

```bash
kdeps --model llama3.1:8b    # 5.2 GB, better quality
kdeps --model llama3.2:3b    # 2.2 GB, good balance
```

**See all available aliases:**

```bash
kdeps llamafile list          # show known model aliases and sizes
kdeps llamafile update        # refresh the registry from HuggingFace
```

Known aliases and their sizes:

| Alias | Model | Size |
|-------|-------|------|
| `llama3.2` / `llama3.2:1b` | Llama 3.2 1B Instruct | ~1.1 GB |
| `llama3.2:3b` | Llama 3.2 3B Instruct | ~2.2 GB |
| `llama3.1:8b` | Llama 3.1 8B Instruct | ~5.2 GB |

The registry has 100+ aliases. Use `kdeps llamafile list` to see the full list.

**Use llamafile in a workflow:**

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file    # default -- no change needed unless you switched backends
```

```yaml
# resources/llm.yaml
chat:
  model: llama3.2:1b    # kdeps downloads this if not already cached
  role: user
  prompt: "{{ get('q') }}"
```

**Point to a local file directly:**

If you have a `.llamafile` binary you downloaded yourself:

```yaml
# resources/llm.yaml
chat:
  model: /path/to/mistral-7b-instruct.llamafile    # absolute path to the file
  role: user
  prompt: "{{ get('q') }}"
```

Or use a URL: the `model` field also accepts a direct download URL.

---

## Ollama

[Ollama](https://ollama.com) is a model manager that runs a local OpenAI-compatible server. It has a larger model catalog than the llamafile registry and supports GPU acceleration when available.

**Install Ollama:**

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh
```

**Pull a model:**

```bash
ollama pull llama3.2        # Meta Llama 3.2 3B
ollama pull deepseek-r1     # DeepSeek R1 reasoning model
ollama pull qwen2.5:7b      # Qwen 2.5 7B
```

**Run kdeps with Ollama:**

```bash
# CLI flag
kdeps --model llama3.2 --backend ollama

# or set it once in config and skip the flags
```

**Set Ollama as the default backend in config:**

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama
  base_url: http://localhost:11434    # default Ollama address
```

**Use Ollama in a workflow resource:**

```yaml
# resources/llm.yaml
chat:
  model: llama3.2          # must be already pulled via `ollama pull`
  role: user
  prompt: "{{ get('q') }}"
  ollamaPullModel: true    # pull automatically if not present (adds startup time)
  ollamaKeepAlive: "5m"   # keep model warm in memory for 5 min after request
```

**Extended thinking with Ollama (DeepSeek R1, QwQ):**

```yaml
# resources/llm.yaml
chat:
  model: deepseek-r1
  ollamaThink: true    # enable reasoning/thinking output for supported models
  role: user
  prompt: "{{ get('q') }}"
```

---

## GGUF backend (llama.cpp)

The `gguf` backend is a third option: it serves GGUF model files via `llama-server` (llama.cpp). This requires `llama-server` installed separately but gives you fine-grained control over quantization and context size.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: gguf
```

Known GGUF aliases: `qwen3.5-4b`, `qwen3.5-8b`, `llama3.2-3b`, `llama3.1-8b`, `phi4-mini`, `gemma3-4b`, `mistral-7b`, `deepseek-r1-7b`.

Environment overrides:
- `KDEPS_LLAMA_SERVER_BIN` - path to the `llama-server` binary
- `KDEPS_CTX_SIZE` - context window size

---

## Privacy

When using `file` (llamafile), `ollama`, or `gguf` backends:

- No request is made to any external API
- Model weights are stored locally in `~/.kdeps/models/`
- All inference happens in your process or a local server process

This makes kdeps suitable for working with sensitive codebases, proprietary documents, or any environment where data must not leave the machine.

---

## See Also

- [LLM Backends Reference](/resources/llm-backends) - Full backend config, routing strategies, all provider options
- [LLM Providers Reference](/reference/llm-providers) - Per-provider snippets for cloud backends
- [Run Locally in 30 Seconds](/getting-started/local-agent) - Quick start with the agent REPL
