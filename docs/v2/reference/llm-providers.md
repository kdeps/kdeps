# LLM Provider Reference

Per-provider configuration for all backends supported by kdeps. Backend and API keys go in `~/.kdeps/config.yaml`. See [LLM Backends](/resources/llm-backends) for routing, allowlists, and streaming.

## Local Backends

### Llamafile (Default)

The `file` backend is the default: models run as
[llamafiles](https://github.com/Mozilla-Ocho/llamafile) - single self-contained
binaries that kdeps downloads to `~/.kdeps/models/` and serves locally as an
OpenAI-compatible server. No server install, no API key.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file   # this is the default - the line can be omitted entirely
```

Model names like `llama3.2:1b` are registry aliases resolved to Mozilla's
HuggingFace llamafiles (`kdeps llamafile list` shows all; `kdeps llamafile update`
refreshes the registry). The `chat.model` field also accepts a direct URL or a
path to a `.llamafile`.

When building Docker images, the llamafiles for all chat models are pre-baked
into the image - see [Docker deployment](/deployment/docker#llm-backend-in-images).

### GGUF (llama.cpp)

The `gguf` backend serves GGUF model files via `llama-server` (llama.cpp). Full parity with the `file` backend: alias resolution, URL download with progress bar, shared cache at `~/.kdeps/models/`. llama-server is automatically downloaded and cached on first use — no manual install needed. Override with `KDEPS_LLAMA_SERVER_BIN` for a custom binary.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: gguf
```

| Alias | Model | Quant | Size |
|-------|-------|-------|------|
| `qwen3.5-4b` | Qwen3.5 4B | Q5_K_S | ~3.1 GB |
| `qwen3.5-8b` | Qwen3.5 8B | Q4_K_M | ~5.0 GB |
| `llama3.2-3b` | Llama 3.2 3B Instruct | Q5_K_M | ~2.4 GB |
| `llama3.1-8b` | Llama 3.1 8B Instruct | Q4_K_M | ~4.9 GB |
| `phi4-mini` | Phi-4 Mini | Q5_K_M | ~2.7 GB |
| `gemma3-4b` | Gemma 3 4B | Q5_K_M | ~3.1 GB |
| `mistral-7b` | Mistral 7B v0.3 | Q4_K_M | ~4.4 GB |
| `deepseek-r1-7b` | DeepSeek-R1 Distill 7B | Q4_K_M | ~5.0 GB |

The `chat.model` field also accepts a direct HuggingFace URL, an absolute/relative path to a `.gguf`, or a bare filename looked up in `~/.kdeps/models/`.

Set `KDEPS_GGUF_CTX_SIZE` to override the context window (default: `llama-server` default).

### Ollama (opt-in)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: ollama
  # base_url: http://custom-ollama:11434   # optional override
```

When building Docker images, Ollama is installed when `backend: ollama` is set. The `installOllama` workflow flag can force or suppress this:

```yaml
# workflow.yaml
settings:
  agentSettings:
    installOllama: true  # bake the ollama server into the image
```

**Provider-specific resource options:**

| Field | Type | Description |
|-------|------|-------------|
| `ollamaThink` | bool | Enable extended thinking (model must support it) |
| `ollamaKeepAlive` | string | Keep model loaded after request (e.g. `"5m"`, `"-1"` = forever, `"0"` = unload immediately) |
| `ollamaPullModel` | bool | Auto-pull model if not present locally |
| `ollamaPullTimeout` | string | Timeout for model pull (e.g. `"10m"`) |

## Cloud Backends

Any API that implements the OpenAI chat completions API works with kdeps.

### OpenAI

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  openai_api_key: sk-...
```

| Model | Description |
|-------|-------------|
| `gpt-4o` | Latest GPT-4 Omni |
| `gpt-4o-mini` | Smaller, faster GPT-4 |
| `gpt-4-turbo` | GPT-4 Turbo |
| `gpt-3.5-turbo` | Fast, cost-effective |

**Provider-specific resource options:**

| Field | Type | Description |
|-------|------|-------------|
| `openAILegacyMaxTokens` | bool | Send `max_tokens` instead of `max_completion_tokens` (for Azure and older-compat servers) |

### Anthropic (Claude)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: anthropic
  anthropic_api_key: sk-ant-...
```

| Model | Description |
|-------|-------------|
| `claude-sonnet-4-20250514` | Latest Claude Sonnet 4 |
| `claude-3-5-sonnet-20241022` | Claude 3.5 Sonnet |
| `claude-3-opus-20240229` | Most capable Claude 3 |
| `claude-3-haiku-20240307` | Fast, efficient |

**Provider-specific resource options:**

| Field | Type | Description |
|-------|------|-------------|
| `promptCaching` | bool | Add `prompt-caching-2024-07-31` beta header for server-side caching |
| `anthropicExtendedOutput` | bool | Enable 128K output tokens (adds `interleaved-thinking-2025-05-14` header) |
| `anthropicBetaHeaders` | list | Additional `anthropic-beta` header values |
| `scenario[].cacheControl` | string | Set to `"ephemeral"` to mark a scenario message as a cache boundary |

See [LLM Backends - Anthropic](/resources/llm-backends#anthropic-prompt-caching-and-extended-output) for examples.

### Google (Gemini / Vertex AI)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: google
  google_api_key: ...   # AI Studio key; omit to use Application Default Credentials for Vertex AI
```

| Model | Description |
|-------|-------------|
| `gemini-1.5-pro` | Latest Gemini Pro |
| `gemini-1.5-flash` | Fast inference |
| `gemini-pro` | Standard Gemini |

**Vertex AI:** Set `googleCloudProject` and `googleCloudLocation` on the `chat:` resource to route to Vertex AI instead of AI Studio. See [LLM Backends - Vertex AI](/resources/llm-backends#vertex-ai-google-cloud).

**Provider-specific resource options:**

| Field | Type | Description |
|-------|------|-------------|
| `googleCachedContent` | string | Name of a Google AI CachedContent resource to attach |
| `googleHarmThreshold` | int | Safety filter level: 0=default, 1=block-none, 2=block-few, 3=block-some, 4=block-most |
| `googleCloudProject` | string | Vertex AI GCP project ID |
| `googleCloudLocation` | string | Vertex AI region (e.g. `us-central1`) |

### Mistral

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: mistral
  mistral_api_key: ...
```

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

| Model | Description |
|-------|-------------|
| `llama-3.1-70b-versatile` | Llama 3.1 70B |
| `llama-3.1-8b-instant` | Llama 3.1 8B (fastest) |
| `mixtral-8x7b-32768` | Mixtral with 32K context |
| `gemma2-9b-it` | Google Gemma 2 9B |

### Together AI

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: together
  together_api_key: ...
```

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

| Model | Description |
|-------|-------------|
| `deepseek-chat` | General chat |
| `deepseek-coder` | Code generation |

### xAI (Grok)

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: xai
  xai_api_key: xai-...
```

| Model | Description |
|-------|-------------|
| `grok-2` | Grok 2 |
| `grok-beta` | Grok beta |
| `grok-vision-beta` | Grok with vision |

### OpenRouter

Access 100+ models from multiple providers through a single API.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openrouter
  openrouter_api_key: sk-or-...
```

Model names use the `provider/model` format, e.g. `openai/gpt-4o`, `anthropic/claude-3.5-sonnet`, `meta-llama/llama-3.1-70b-instruct`. See [openrouter.ai/models](https://openrouter.ai/models) for the full list.

## Self-Hosted Solutions

kdeps works with any self-hosted solution that implements the OpenAI API: vLLM, Text Generation Inference (TGI), LocalAI, LlamaCpp Server.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://your-vllm-server:8000/v1
```

## Custom Base URL

Override the default API URL via `base_url`:

```yaml
# Azure OpenAI
llm:
  backend: openai
  base_url: "https://my-resource.openai.azure.com/openai/deployments/my-deployment"
  openai_api_key: ...
```

## See Also

- [LLM Backends](/resources/llm-backends) - Routing, allowlists, streaming, feature matrix
- [LLM Resource](/resources/llm) - Complete LLM resource documentation
