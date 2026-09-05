# LLM backends reference

kdeps separates two concerns: which model to call (set in the resource file) and where to call it (set in `~/.kdeps/config.yaml`). This lets you switch backends without touching your workflow.

*Applies to both workflow mode and agent mode.*

Whichever backend you pick - cloud (OpenAI, Anthropic, Groq, ...) or local (llamafile, Ollama, GGUF) - the model is probabilistic. The [workflow pipeline](/modes/workflow-mode) around it is what makes a kdeps agent deterministic, not the model.

## The default: llamafile (file backend)

With no configuration at all, chat resources run on the `file` backend: the
model is a [llamafile](https://github.com/Mozilla-Ocho/llamafile) - a single
self-contained binary that kdeps downloads, caches in `~/.kdeps/models/`, and
serves as a local OpenAI-compatible server. No ollama, no GPU, no API key.

```text
chat resource --> kdeps resolves model alias --> downloads llamafile (once)
                                              --> serves it locally
                                              --> request answered
```

Known model aliases map to Mozilla's HuggingFace llamafiles. Quantization is
part of the alias so you can trade size for quality:

| Alias | Model | Quant | Size |
|-------|-------|-------|------|
| `llama3.2` / `llama3.2:1b` | Llama 3.2 1B Instruct (default) | Q4_K_M | ~1.1 GB |
| `ministral3` / `ministral3:3b` | Ministral 3 3B Instruct | Q4_K_M | ~3.1 GB |
| `llama3.2:1b-q6` | Llama 3.2 1B Instruct | Q6_K | ~1.5 GB |
| `llama3.2:1b-q8` | Llama 3.2 1B Instruct | Q8_0 | ~2.1 GB |
| `llama3.2:3b` | Llama 3.2 3B Instruct | Q4_K_M | ~2.2 GB |
| `llama3.1:8b` | Llama 3.1 8B Instruct | Q4_K_M | ~5.2 GB |

```bash
kdeps llamafile list      # all known aliases (the registry has 100+ models)
kdeps llamafile update    # refresh the registry from HuggingFace
```

The `chat.model` field also accepts a direct URL, an absolute/relative path to
a `.llamafile`, or a bare filename looked up in `~/.kdeps/models/`.

## GGUF backend (llama.cpp)

The `gguf` backend serves GGUF model files via `llama-server` (llama.cpp). Same download-and-cache flow as `file`, but requires `llama-server` installed separately.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: gguf
```

```text
chat resource --> kdeps resolves GGUF alias --> downloads .gguf (once)
                                            --> starts llama-server
                                            --> request answered
```

Known aliases: `qwen3.5`, `qwen3.5:4b`, `qwen3.5:9b`, `qwen3.5:27b`, `gemma4`, `glm4.5`, `yi34` (see `kdeps llamafile list` for the full registry). The `chat.model` field also accepts a direct URL, absolute/relative path to a `.gguf`, or a bare filename in `~/.kdeps/models/`.

Environment overrides: `KDEPS_LLAMA_SERVER_BIN` (binary path), `KDEPS_GGUF_CTX_SIZE` (context window).

## Ollama (opt-in)

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

See [Ollama: native options](#ollama-native-options) below for resource-level fields (`ollamaThink`, `ollamaKeepAlive`, ...).

## Where it runs

Backend configuration applies to both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). All `chat:` resources in both modes resolve their backend from `~/.kdeps/config.yaml`.

## Model configuration

Model is set per resource in `chat.model`:

```yaml
# resources/my-resource.yaml
chat:
  model: llama3.2:1b    # which model to call
  role: user
  prompt: "{{ get('q') }}"
```

Set `model: router` to delegate model selection to the router in `~/.kdeps/config.yaml` (see [Routing](#routing) below).

Backend, base URL, and API keys go in `~/.kdeps/config.yaml`:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: file                # default: local llamafile, no server install
  # backend: ollama            # opt-in: requires the ollama server
  # base_url: http://localhost:11434
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...
  # groq_api_key: ...
```

Run `kdeps edit` to open the config file, or edit it directly.

## Unified models list

`llm.models` in `~/.kdeps/config.yaml` serves dual purpose: it can act as a **plain allowlist** (model names only) or as a **router route table** (with routing metadata). The `llm.strategy` field switches between the two modes.

### Allowlist mode (no strategy)

When `strategy` is absent, `llm.models` is a simple list of permitted model names:

```yaml
# resources/example.yaml
llm:
  backend: ollama
  models:
    - llama3.2:1b        # plain string entry
    - nomic-embed-text
```

Each entry is a plain model name. Models can be specified as strings (as above) or as objects with only the `model` field set:

```yaml
# resources/example.yaml
llm:
  models:
    - model: llama3.2:1b  # object form (equivalent to "llama3.2:1b")
```

Any request for a model not in this list is overridden to the first model and a warning is logged. Models listed here are pre-pulled into Docker/ISO artifacts.

### Routing mode (with strategy)

When `strategy` is set, the models list acts as router routes:

```yaml
# resources/example.yaml
llm:
  strategy: token_threshold
  models:
    - model: gpt-4o-mini
      backend: openai
      max_tokens: 500
      default: true
    - model: gpt-4o
      backend: openai
      min_tokens: 501
```

Plain string entries in routing mode (no `model:` key) are still allowed - they inherit the default `llm.backend`:

```yaml
# resources/example.yaml
llm:
  backend: ollama
  strategy: fallback
  models:
    - llama3.2:1b          # plain string, uses backend: ollama
    - model: gpt-4o
      backend: openai
      priority: 1
```

### Entry fields

Each model entry supports these fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | yes | Model identifier (e.g. `gpt-4o`, `llama3.2:1b`) |
| `backend` | string | no | Backend for this model (overrides `llm.backend`) |
| `base_url` | string | no | Custom API URL for this backend |
| `priority` | int | no | Fallback order (lower = tried first) |
| `min_tokens` | int | no | Minimum prompt tokens for token_threshold |
| `max_tokens` | int | no | Maximum prompt tokens for token_threshold |
| `cost_per_input_token` | float | no | Cost per 1K input tokens for cost_optimized |
| `cost_per_output_token` | float | no | Cost per 1K output tokens for cost_optimized |
| `default` | bool | no | Catch-all route when no other rule matches |

## Routing

Routing delegates model selection from resource YAML to `~/.kdeps/config.yaml`. Set a resource's `model` field to `router` (or `auto-router` for a zero-config version that ignores `llm.models` entirely) and the config's `strategy` picks the model: `token_threshold`, `fallback`, `cost_optimized`, `round_robin`, or `auto` (hardware-fit scoring via `llmfit`).

See [Routing](/resources/llm/routing) for every strategy's config shape and resolution order.

## Supported backends

kdeps supports local backends (Llamafile, GGUF/llama.cpp, Ollama) and any OpenAI-compatible API: OpenAI, Anthropic, Google, Mistral, Groq, Together AI, Perplexity, Cohere, DeepSeek, xAI (Grok), OpenRouter, AWS Bedrock, IBM WatsonX, M365 Copilot, and self-hosted solutions (vLLM, TGI, LocalAI, LlamaCpp). See [LLM Provider Reference](/reference/llm-providers) for per-provider config snippets and available model names.

## Vertex AI (google cloud)

Target Google's Vertex AI endpoint instead of the standard AI Studio endpoint by setting `googleCloudProject` and `googleCloudLocation` on the `chat:` resource. The backend in `config.yaml` stays `google`; the two resource-level fields route the call to the regional Vertex endpoint.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: google
  google_api_key: ...   # or use Application Default Credentials (ADC)
```

```yaml
# resources/example.yaml
chat:
  model: gemini-1.5-pro
  googleCloudProject: my-gcp-project   # GCP project ID
  googleCloudLocation: us-central1     # Vertex AI region
  prompt: "{{ get('q') }}"
```

Vertex AI uses Application Default Credentials when no `google_api_key` is present. Run `gcloud auth application-default login` to authenticate locally.

## Google: cached content and safety threshold

`googleCachedContent` names a Google AI CachedContent resource to attach to the request - pre-cache large context with the `google_cache_create` built-in tool, then reference it here. `googleHarmThreshold` controls how aggressively Google's safety filters block responses.

```yaml
# resources/example.yaml
chat:
  model: gemini-1.5-pro
  googleCachedContent: "cachedContents/my-cached-doc"   # CachedContent resource name
  googleHarmThreshold: 1   # 0 provider default, 1 block none, 2 few, 3 some, 4 most
  prompt: "{{ get('q') }}"
```

## OpenAI: legacy token param

Older OpenAI-compatible servers (Azure, some self-hosted) expect `max_tokens` instead of `max_completion_tokens`. Set `openAILegacyMaxTokens: true` to send the old parameter name.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  openAILegacyMaxTokens: true   # send max_tokens instead of max_completion_tokens
  maxTokens: 1000
```

## Anthropic: prompt caching and extended output

Anthropic-specific options are set per resource, not in `config.yaml`.

### Prompt caching

`promptCaching: true` adds the `anthropic-beta: prompt-caching-2024-07-31` header. Anthropic caches the first qualifying prefix of the prompt (system + long context). Reduces latency and cost on repeated long system prompts.

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  promptCaching: true
  scenario:
    - role: system
      prompt: |
        You are an expert assistant with access to the following reference material:
        [... long document ...]
      cacheControl: "ephemeral"   # mark this message as the cache boundary
```

### Extended output (128k tokens)

`anthropicExtendedOutput: true` enables 128K output tokens and adds the `interleaved-thinking-2025-05-14` beta header automatically.

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicExtendedOutput: true
  maxTokens: 16000
  prompt: "{{ get('q') }}"
```

### Custom beta headers

Pass arbitrary beta feature strings via `anthropicBetaHeaders`:

```yaml
# resources/example.yaml
chat:
  model: claude-sonnet-4-20250514
  anthropicBetaHeaders:
    - output-128k-2025-02-19
    - interleaved-thinking-2025-05-14
```

## Ollama: native options

Extra controls for the Ollama backend, set per resource.

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  ollamaThink: true          # enable extended thinking (requires a thinking-capable model)
  ollamaKeepAlive: "5m"      # keep model in memory for 5 min after the request completes
  ollamaPullModel: true      # pull the model automatically if it is not present
  ollamaPullTimeout: "10m"   # how long to wait for the pull before failing
```

`ollamaKeepAlive` accepts Go duration strings: `"0"` unloads immediately; `"-1"` keeps the model loaded indefinitely; `"5m"`, `"1h"`, etc. set a timed expiry.

## Streaming (Ollama)

Set `streaming: true` on a `chat:` resource to have Ollama stream the response as NDJSON chunks. kdeps accumulates all chunks internally and returns the same response shape as a non-streaming call.

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  streaming: true      # Ollama only
```

</div>

| [`streaming`](/reference/glossary#streaming) | What happens |
|-------------|-------------|
| `false` (default) | Single JSON response |
| `true` | Ollama streams NDJSON; kdeps accumulates and returns merged map |

`streaming: true` is silently ignored for non-Ollama backends.

## Feature support

| Feature | Ollama | OpenAI | Anthropic | Google | Mistral | Groq |
|---------|--------|--------|-----------|--------|---------|------|
| JSON Response | Yes | Yes | Partial | Yes | Yes | Yes |
| Tools/Functions | Yes | Yes | Yes | Yes | Yes | Yes |
| Vision | Yes* | Yes | Yes | Yes | Yes | Yes |
| Streaming | Yes | No** | No** | No** | No** | No** |

*Requires vision-capable model (e.g., `llama3.2-vision`)
**Streaming is only supported for the Ollama backend.

## Troubleshooting

### Ollama connection issues

If Ollama cannot be reached:
1. Check Ollama is running: `ollama list`
2. Verify the URL in config.yaml (default: `http://localhost:11434`)
3. Check firewall settings

### API key issues

If you get authentication errors:
1. Verify the key is set in `~/.kdeps/config.yaml`
2. Or export the env var: `export OPENAI_API_KEY=sk-...`
3. Check the key has the correct permissions

### Model not found

If the model is not available:
1. For Ollama: Pull the model first with `ollama pull model-name`
2. For APIs: Verify the model name matches the provider's documentation
3. Check you have access to the model in your API account

### Rate limiting

Handle rate limits with retry configuration via `onError`:

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: "{{ get('q') }}"
  onError:
    action: "retry"
    maxRetries: 3
    retryDelay: "5s"
```

</div>

## Self-hosted LLM appliance

Deploy a dedicated OpenAI-compatible server with `kdeps llm` (Docker, ISO, Kubernetes), then point this host at it with `backend: openai` and `base_url`.

Stock engines: `ollama`, `llamafile`, `llama-server` / `gguf`, `llamacpp`, `vllm`, `tgi`, `sglang`, `localai`, `openai-compat`.

```bash
kdeps llm list
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm client-config --url http://192.168.1.50:8000/v1
```

```yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
```

See [LLM server appliance](/deployment/llm-server) and [LLM commands](/reference/cli/llm).

## See also

- [LLM provider reference](/reference/llm-providers) - Per-provider config snippets and model names
- [LLM resource](/resources/llm/) - Complete LLM resource documentation
- [Tools](/concepts/tools) - LLM function calling
- [Docker deployment](/deployment/docker) - Deploying with local models

