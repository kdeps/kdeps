---
title: Local Models (Llamafile & Ollama)
description: Run kdeps entirely offline with local models. Your code and prompts never leave your machine.
---

# Local models (llamafile & Ollama)

kdeps can run fully offline. With a local model backend, kdeps sends no data to external APIs. Your prompts, code, and responses stay on your machine. For the full self-hosted picture (your own LLM server, air-gapped bundles), see [Data sovereignty](/start/data-sovereignty).

kdeps supports two local backends: llamafile (the default, with no installation) and Ollama (a model manager with a larger catalog).

---

## Llamafile (default backend)

Llamafile is a model + server packaged into a single self-contained binary. It runs on any OS - Mac, Linux, Windows - without a GPU requirement, no server to install separately.

kdeps uses the file backend by default. It downloads a model on first use and caches it in `~/.kdeps/models/`.

## PyBot (the model when a chat names none)

A `chat` with no `model:` runs on [PyBot](https://github.com/huggingface/pybot). PyBot is a rule-based chatbot in plain Python, with no model weights. kdeps downloads a pinned copy (14 MB) on first use and runs it on a local port. It needs `python3`.

```yaml
chat:
  context:
    prompt: "{{ get('q') }}"   # no model: -> pybot answers; set model: for a language model
```

- PyBot reads only the user turns. It ignores system prompts.
- PyBot does not call tools. A chat with `loop:` on PyBot only answers, and kdeps logs a warning. Set `model:` (for example `ministral3`) for an agent.
- An image built for such a workflow contains PyBot, so it answers offline.

## Pick a llamafile

**Pick a model by alias** in the chat resource:

```yaml
chat:
  model:
    name: llama3.2:1b    # ~1.1 GB; llama3.2:3b is 2.2 GB, llama3.1:8b is 5.2 GB
```

**See all available aliases:**

```bash
kdeps llamafile list          # show known model aliases and sizes
kdeps update models           # newest registry (harvested nightly from HuggingFace)
```

Known aliases and their sizes:

| Alias | Model | Size |
|-------|-------|------|
| `llama3.2` / `llama3.2:1b` | Llama 3.2 1B Instruct | ~1.1 GB |
| `ministral3` / `ministral3:3b` | Ministral 3 3B Instruct (default of the `assistant` template; calls tools) | ~3.1 GB |
| `llama3.2:3b` | Llama 3.2 3B Instruct | ~2.2 GB |
| `llama3.1:8b` | Llama 3.1 8B Instruct | ~5.2 GB |

A chat with `loop:` needs a model that calls tools. A 1B model does not. Use `ministral3` or larger.

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
  model:
    name: llama3.2:1b    # kdeps downloads this if not already cached
  context:
    role: user
    prompt: "{{ get('q') }}"
```

**Point to a local file directly:**

If you have a `.llamafile` binary you downloaded yourself:

```yaml
# resources/llm.yaml
chat:
  model:
    name: /path/to/mistral-7b-instruct.llamafile    # absolute path to the file
  context:
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

**Set Ollama as the default backend in config:**

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama
  base_url: http://localhost:11434/v1    # Ollama OpenAI-compatible endpoint
```

**Use Ollama in a workflow resource:**

```yaml
# resources/llm.yaml
chat:
  model:
    name: llama3.2          # must be already pulled via `ollama pull`
    ollamaPullModel: true    # pull automatically if not present (adds startup time)
    ollamaKeepAlive: "5m"   # keep model warm in memory for 5 min after request
  context:
    role: user
    prompt: "{{ get('q') }}"
```

**Extended thinking with Ollama (DeepSeek R1, QwQ):**

```yaml
# resources/llm.yaml
chat:
  model:
    name: deepseek-r1
    ollamaThink: true    # enable reasoning/thinking output for supported models
  context:
    role: user
    prompt: "{{ get('q') }}"
```

---

## GGUF backend (llama.cpp)

The `gguf` backend is a third option. It serves GGUF model files through `llama-server` (llama.cpp). You must install `llama-server` separately. In return, you get fine control of quantization and context size.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: gguf
```

Known GGUF aliases: `qwen3.5`, `qwen3.5:4b`, `qwen3.5:9b`, `qwen3.5:27b`, `qwen3.5:122b`, `qwen3`, `gemma4`, `glm4.5`, `glm4.7`, `yi34`, `yi6`, `dolphin2.9`, `hermes2.5`. Each has quantization variants (`-q4`, `-q6`, `-q8`, ...). Find more with `kdeps llamafile list`.

Environment overrides:
- `KDEPS_LLAMA_SERVER_BIN` - path to the `llama-server` binary
- `KDEPS_CTX_SIZE` - context window size of the local model server. The default is 8192 tokens (`chat.context_length` in the executor defaults). A chat with `loop:` needs about 4000 tokens for its own prompt.

---

## Privacy

When you use the `file` (llamafile), `ollama`, or `gguf` backends:

- kdeps sends no request to an external API
- kdeps keeps model weights locally in `~/.kdeps/models/`
- All inference happens in your process or a local server process

This makes kdeps suitable for working with sensitive codebases, proprietary documents, or any environment where data must not leave the machine.

---

## See also

- [LLM backends reference](/workflow/resources/llm-backends) - Full backend config, routing strategies, all provider options
- [LLM providers reference](/llm-server/providers) - Per-provider snippets for cloud backends
- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools
