# LLM backends

A chat resource names the model. `~/.kdeps/config.yaml` names where it runs: a local backend or a cloud API. As a result, you can change backends with no change to the workflow.

```text
chat resource (model name) -> ~/.kdeps/config.yaml (backend, keys) -> local server or cloud API
```

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file                # default: a local llamafile, no server install
  # backend: ollama            # opt-in: needs the ollama server
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...
```

With no configuration, chats run on the `file` backend. kdeps downloads a [llamafile](https://github.com/Mozilla-Ocho/llamafile) one time, caches it in `~/.kdeps/models/`, and serves it locally. You need no GPU and no API key.

## Backends

| Kind | Backends |
|---|---|
| Local | `file` (llamafile, the default), `gguf` (llama.cpp), `ollama` |
| Cloud | OpenAI, Anthropic, Google, Mistral, Groq, Together AI, DeepSeek, OpenRouter, xAI and more |

Run `kdeps config --edit` to open the config file.

## Read next

- [LLM backends reference](/reference/resources/llm-backends) - model aliases, base URLs, providers, the models list, Vertex AI and provider options
- [Routing](/workflow/resources/llm-routing) - pick the model per request from `config.yaml`
- [LLM provider reference](/reference/llm-server/providers) - snippets and model names for each provider
