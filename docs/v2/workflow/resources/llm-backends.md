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

## Read next

- [LLM backends reference](/reference/resources/llm-backends) - every key, option and detail
- [Routing](/workflow/resources/llm-routing) - pick the model per request from `config.yaml`
