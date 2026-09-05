# LLM provider reference

Per-provider configuration for all backends supported by kdeps. Backend and API keys go in `~/.kdeps/config.yaml`. See [LLM backends](/resources/llm/backends) for routing, allowlists, and streaming.

*Applies to both workflow mode and agent mode.*

Every model listed here - OpenAI, Anthropic (Claude), Google (Gemini), Groq, Ollama, local llamafile / GGUF, and the rest - is **probabilistic**: the same prompt can return different text on each call. Determinism in kdeps comes from [workflow mode](/modes/workflow-mode) wrapping the model, not from the model itself. See [Deterministic by design](/concepts/why-kdeps#deterministic-by-design).

## Local backends

kdeps runs models locally with no API key: `file` (llamafile, the default),
`gguf` (llama.cpp), or `ollama` (opt-in, requires the Ollama server). Setup,
model alias tables, and the Docker `installOllama` flag live on one page - see
[LLM backends - local backends](/resources/llm/backends#the-default-llamafile-file-backend).

## Cloud backends

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

Azure and older-compat servers: see [LLM Backends - legacy token param](/resources/llm/backends#openai-legacy-token-param).

### Anthropic (claude)

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

Prompt caching, extended 128K output, and custom beta headers: see
[LLM Backends - Anthropic](/resources/llm/backends#anthropic-prompt-caching-and-extended-output).

### Google (gemini / vertex AI)

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

**Vertex AI:** Set `googleCloudProject` and `googleCloudLocation` on the `chat:` resource to route to Vertex AI instead of AI Studio. See [LLM Backends - Vertex AI](/resources/llm/backends#vertex-ai-google-cloud).

CachedContent and safety threshold options (`googleCachedContent`, `googleHarmThreshold`): see [LLM Backends - Google](/resources/llm/backends#google-cached-content-and-safety-threshold).

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
| `deepseek-v4-pro` | Flagship |
| `deepseek-v4-flash` | Fast |
| `deepseek-coder` | Code generation |

### xAI (grok)

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

### AWS bedrock

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: bedrock
```

Authenticates via the standard AWS SDK credential chain, not a `config.yaml` API key - set `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_REGION` as environment variables (or use an IAM instance role). `model` is the Bedrock model ID for your region, e.g. `anthropic.claude-3-5-sonnet-20241022-v2:0` or `meta.llama3-1-70b-instruct-v1:0`.

### IBM WatsonX

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: watsonx
  watsonx_api_key: ...
```

Also requires `WATSONX_PROJECT_ID` as an environment variable (no `config.yaml` field for it). `model` is a WatsonX model ID, e.g. `ibm/granite-13b-chat-v2` or `meta-llama/llama-3-70b-instruct`.

### M365 Copilot

Talks to Microsoft 365 Copilot's chat service through a local OpenAI-compatible server kdeps runs in front of it (`m365` backend) - no API key, authentication is a signed-in Microsoft 365 account via a browser sign-in flow (interactive) or a scripted `secrets.json` (headless/CI).

See [M365 Copilot](/reference/llm-providers-m365) for the full setup: sign-in flows, Linux dependencies, headless fallback, and the model list.

## Self-hosted solutions

kdeps works with any self-hosted solution that implements the OpenAI API: vLLM, Text Generation Inference (TGI), LocalAI, LlamaCpp Server, LM Studio.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://your-vllm-server:8000/v1
```

## Custom base URL

Override the default API URL via `base_url`:

```yaml
# Azure OpenAI
llm:
  backend: openai
  base_url: "https://my-resource.openai.azure.com/openai/deployments/my-deployment"
  openai_api_key: ...
```

## See also

- [LLM backends](/resources/llm/backends) - Routing, allowlists, streaming, feature matrix
- [LLM resource](/resources/llm/) - Complete LLM resource documentation
