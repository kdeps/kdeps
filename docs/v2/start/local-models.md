---
title: Local Models (Llamafile & Ollama)
description: Run kdeps entirely offline with local models. Your code and prompts never leave your machine.
---

# Local models (llamafile & Ollama)

kdeps runs fully offline with a local model backend. Your prompts, code and replies stay on your machine.

| Backend | What it is |
|---|---|
| `file` (default) | a [llamafile](https://github.com/Mozilla-Ocho/llamafile): model and server in one binary, downloaded on first use to `~/.kdeps/models/`, no GPU needed |
| `ollama` | the Ollama model manager, with a larger catalog |
| `gguf` | a llama.cpp server for GGUF files |

A `chat` with no `model:` runs on [PyBot](https://github.com/huggingface/pybot), a rule-based chatbot with no model weights. Set `model:` to use a language model:

<div v-pre>

```yaml
chat:
  model:
    name: llama3.2:1b          # a llamafile on the file backend; omit for pybot
  context:
    prompt: "{{ get('q') }}"
```

</div>

## Read next

- [Local models (llamafile & Ollama) reference](/reference/start/local-models) - llamafile (default backend), pyBot (the model when a chat names none), pick a llamafile, ollama, GGUF backend (llama.cpp), privacy
- [LLM backends reference](/workflow/resources/llm-backends) - Full backend config, routing strategies, all provider options
- [LLM providers reference](/reference/llm-server/providers) - Per-provider snippets for cloud backends
- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools
