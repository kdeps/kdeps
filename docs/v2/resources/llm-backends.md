# LLM Backends Reference

KDeps supports LLM integrations through Ollama for local model serving and any OpenAI-compatible API endpoint for cloud or self-hosted models.

## Configuration

**Model is set per resource** in `run.chat.model` in resource YAML. **Backend, base URL, and API keys** are configured in `~/.kdeps/config.yaml`.

```yaml
# resources/my-resource.yaml
run:
  chat:
    model: llama3.2:1b    # Per-resource model selection
    role: user
    prompt: "{{ get('q') }}"
```

Set `model: router` to delegate model selection to the LLM router (see [LLM Router](#llm-router) below).

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama              # Default backend
  # base_url: http://localhost:11434
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...
  # groq_api_key: ...
```

Run `kdeps edit` to open the config file, or edit it directly.

## Backend Overview

### Local Backend

| Backend | Name | Default URL | Description |
|---------|------|-------------|-------------|
| Ollama | `ollama` | `http://localhost:11434` | Local model serving (default) |

### Cloud/Remote Backends

KDeps supports **any OpenAI-compatible API endpoint**. This includes:
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude)
- Google (Gemini)
- Groq - native OpenAI compatibility
- Together AI - native OpenAI compatibility
- Any self-hosted solution (vLLM, TGI, LocalAI, LlamaCpp)

## Local Backend

### Ollama (Default)

Ollama is the default backend for local model serving. Configure it in `~/.kdeps/config.yaml`:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama
  # base_url: http://custom-ollama:11434   # optional override
```

**Docker Build:**

When building Docker images, Ollama is automatically installed when `backend: ollama` is set in config.yaml. The `installOllama` workflow flag can also force or suppress this:

```yaml
settings:
  agentSettings:
    ollamaImageTag: "0.13.5"  # Ollama version to install
    installOllama: true        # Force install (optional)
```

## OpenAI-Compatible Backends

Any API that implements the OpenAI chat completions API can be used with KDeps.

### OpenAI

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  openai_api_key: sk-...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `gpt-4o` | Latest GPT-4 Omni |
| `gpt-4o-mini` | Smaller, faster GPT-4 |
| `gpt-4-turbo` | GPT-4 Turbo |
| `gpt-3.5-turbo` | Fast, cost-effective |

### Anthropic (Claude)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: anthropic
  anthropic_api_key: sk-ant-...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `claude-3-5-sonnet-20241022` | Latest Claude 3.5 Sonnet |
| `claude-3-opus-20240229` | Most capable |
| `claude-3-sonnet-20240229` | Balanced |
| `claude-3-haiku-20240307` | Fast, efficient |

### Google (Gemini)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: google
  google_api_key: ...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `gemini-1.5-pro` | Latest Gemini Pro |
| `gemini-1.5-flash` | Fast inference |
| `gemini-pro` | Standard Gemini |

### Mistral

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: mistral
  mistral_api_key: ...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `mistral-large-latest` | Most capable |
| `mistral-medium-latest` | Balanced |
| `mistral-small-latest` | Fast, efficient |
| `open-mistral-7b` | Open-source 7B |
| `open-mixtral-8x7b` | MoE model |

### Groq

Ultra-fast inference with Groq hardware.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: groq
  groq_api_key: ...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `llama-3.1-70b-versatile` | Llama 3.1 70B |
| `llama-3.1-8b-instant` | Llama 3.1 8B (fastest) |
| `mixtral-8x7b-32768` | Mixtral with 32K context |
| `gemma2-9b-it` | Google Gemma 2 9B |

### Together AI

Access to many open-source models.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: together
  model: meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
  together_api_key: ...
```

**Popular models:**

| Model | Description |
|-------|-------------|
| `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo` | Llama 3.1 70B |
| `meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo` | Llama 3.1 8B |
| `mistralai/Mixtral-8x7B-Instruct-v0.1` | Mixtral 8x7B |
| `Qwen/Qwen2-72B-Instruct` | Qwen2 72B |

### Perplexity

Search-augmented LLM responses.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: perplexity
  perplexity_api_key: ...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `llama-3.1-sonar-large-128k-online` | Large with web search |
| `llama-3.1-sonar-small-128k-online` | Small with web search |
| `llama-3.1-sonar-large-128k-chat` | Large chat only |

### Cohere

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: cohere
  cohere_api_key: ...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `command-r-plus` | Most capable |
| `command-r` | Fast and capable |
| `command` | Standard |
| `command-light` | Fast, efficient |

### DeepSeek

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: deepseek
  deepseek_api_key: ...
```

**Available models:**

| Model | Description |
|-------|-------------|
| `deepseek-chat` | General chat |
| `deepseek-coder` | Code generation |

### Self-Hosted Solutions

KDeps works with any self-hosted LLM serving solution that implements the OpenAI API:

- **vLLM** - High-performance inference server
- **Text Generation Inference (TGI)** - Hugging Face's serving solution
- **LocalAI** - Drop-in replacement for OpenAI API
- **LlamaCpp Server** - Efficient CPU inference

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://your-vllm-server:8000/v1
  model: meta-llama/Llama-2-7b-chat-hf
```

## LLM Router

For multi-backend routing (e.g., send coding questions to one model, general questions to another), configure the router in `~/.kdeps/config.yaml`:

```yaml
llm:
  backend: ollama
  model: llama3.2:1b
  router:
    - condition: 'contains(prompt, "code") || contains(prompt, "python")'
      backend: openai
      model: gpt-4o
    - condition: 'contains(prompt, "image")'
      backend: anthropic
      model: claude-3-5-sonnet-20241022
```

See the [LLM Resource](llm) docs for router details.

## Model Allowlist

To restrict which models can be used at runtime, set `llm.models` in `~/.kdeps/config.yaml`:

```yaml
llm:
  backend: ollama
  model: llama3.2:1b
  models:
    - llama3.2:1b
    - nomic-embed-text
```

Any request for a model not in this list is overridden with the first model and a warning is logged.

For Docker/offline deployments, models listed here are pre-pulled into the image.

## Custom Base URL

Override the default API URL via `base_url` in config.yaml:

```yaml
# Azure OpenAI
llm:
  backend: openai
  base_url: "https://my-resource.openai.azure.com/openai/deployments/my-deployment"
  openai_api_key: ...
```

## Streaming (Ollama)

Set `streaming: true` on a `chat:` resource to have Ollama stream the response as NDJSON chunks. KDeps accumulates all chunks internally and returns the same response shape as a non-streaming call.

<div v-pre>

```yaml
run:
  chat:
    prompt: "{{ get('q') }}"
    streaming: true      # Ollama only
```

</div>

| `streaming` | What happens |
|-------------|-------------|
| `false` (default) | Single JSON response |
| `true` | Ollama streams NDJSON; KDeps accumulates and returns merged map |

`streaming: true` is silently ignored for non-Ollama backends.

## Feature Support

| Feature | Ollama | OpenAI | Anthropic | Google | Mistral | Groq |
|---------|--------|--------|-----------|--------|---------|------|
| JSON Response | Yes | Yes | Partial | Yes | Yes | Yes |
| Tools/Functions | Yes | Yes | No | Yes | Yes | Yes |
| Vision | Yes* | Yes | Yes | Yes | Yes | Yes |
| Streaming | Yes | No** | No** | No** | No** | No** |

*Requires vision-capable model (e.g., `llama3.2-vision`)
**Streaming is only supported for the Ollama backend.

## Troubleshooting

### Ollama Connection Issues

If Ollama cannot be reached:
1. Check Ollama is running: `ollama list`
2. Verify the URL in config.yaml (default: `http://localhost:11434`)
3. Check firewall settings

### API Key Issues

If you get authentication errors:
1. Verify the key is set in `~/.kdeps/config.yaml`
2. Or export the env var: `export OPENAI_API_KEY=sk-...`
3. Check the key has the correct permissions

### Model Not Found

If the model is not available:
1. For Ollama: Pull the model first with `ollama pull model-name`
2. For APIs: Verify the model name matches the provider's documentation
3. Check you have access to the model in your API account

### Rate Limiting

Handle rate limits with retry configuration on the resource:

<div v-pre>

```yaml
run:
  chat:
    prompt: "{{ get('q') }}"
    retry:
      maxAttempts: 3
      initialDelay: "1s"
      maxDelay: "30s"
      backoffMultiplier: 2
```

</div>

## See Also

- [LLM Resource](llm) - Complete LLM resource documentation
- [Tools](../concepts/tools) - LLM function calling
- [Docker Deployment](../deployment/docker) - Deploying with local models
