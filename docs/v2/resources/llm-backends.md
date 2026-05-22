# LLM Backends Reference

kdeps separates two concerns: which model to call (set in the resource file) and where to call it (set in `~/.kdeps/config.yaml`). This lets you switch backends without touching your workflow.

## Where it runs

Backend configuration applies to both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). All `chat:` resources in both modes resolve their backend from `~/.kdeps/config.yaml`.

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
  backend: ollama              # default backend for all resources
  # base_url: http://localhost:11434
  # openai_api_key: sk-...
  # anthropic_api_key: sk-ant-...
  # groq_api_key: ...
```

Run `kdeps edit` to open the config file, or edit it directly.

## Unified Models List

`llm.models` in `~/.kdeps/config.yaml` serves dual purpose: it can act as a **plain allowlist** (model names only) or as a **router route table** (with routing metadata). The `llm.strategy` field switches between the two modes.

### Allowlist Mode (no strategy)

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

### Routing Mode (with strategy)

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

Plain string entries in routing mode (no `model:` key) are still allowed — they inherit the default `llm.backend`:

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

### Entry Fields

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

Routing delegates model selection from resource YAML to the config. Set a resource's `model` field to `router`:

```yaml
# resources/llm.yaml
chat:
  model: router       # delegate to config.yaml router
  role: user
  prompt: "{{ get('q') }}"
```

The router in `~/.kdeps/config.yaml` selects which model to use based on the configured strategy.

### Strategy: `token_threshold`

Routes by estimated prompt token count. The first entry where `min_tokens <= tokens <= max_tokens` wins. Falls through to the entry with `default: true` when no range matches.

```yaml
# resources/example.yaml
llm:
  strategy: token_threshold
  models:
    - model: gpt-4o-mini
      backend: openai
      max_tokens: 500         # short prompts use this
      default: true
    - model: gpt-4o
      backend: openai
      min_tokens: 501         # long prompts use this
```

Token counts are estimated using tiktoken.

### Strategy: `fallback`

Tries routes in priority order. On error, automatically retries the next route.

```yaml
# resources/example.yaml
llm:
  strategy: fallback
  models:
    - model: claude-sonnet-4-20250514
      backend: anthropic
      priority: 1
    - model: gpt-4o
      backend: openai
      priority: 2
    - model: llama3.2:1b
      backend: ollama
      priority: 3
      default: true
```

Lower priority values are tried first. `default: true` marks the catch-all route.

### Strategy: `cost_optimized`

Selects the cheapest route based on cost per 1K input tokens.

```yaml
# resources/example.yaml
llm:
  strategy: cost_optimized
  models:
    - model: gpt-4o-mini
      backend: openai
      cost_per_input_token: 0.00015   # $0.15/1M tokens
    - model: gpt-4o
      backend: openai
      cost_per_input_token: 0.0025    # $2.50/1M tokens
      default: true
```

Nil cost is treated as zero. Falls to `default: true` on tie.

### Strategy: `round_robin`

Distributes requests evenly across models using an atomic counter.

```yaml
# resources/example.yaml
llm:
  strategy: round_robin
  models:
    - model: gpt-4o
      backend: openai
    - model: claude-sonnet-4-20250514
      backend: anthropic
```

Counters are keyed by a fingerprint of the model list, so different route configs maintain independent counters.

## Supported Backends

kdeps supports Ollama (local) and any OpenAI-compatible API: OpenAI, Anthropic, Google, Mistral, Groq, Together AI, Perplexity, Cohere, DeepSeek, and self-hosted solutions (vLLM, TGI, LocalAI, LlamaCpp). See [LLM Provider Reference](/reference/llm-providers) for per-provider config snippets and available model names.

## Streaming (Ollama)

Set `streaming: true` on a `chat:` resource to have Ollama stream the response as NDJSON chunks. KDeps accumulates all chunks internally and returns the same response shape as a non-streaming call.

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

## See Also

- [LLM Provider Reference](/reference/llm-providers) - Per-provider config snippets and model names
- [LLM Resource](llm) - Complete LLM resource documentation
- [Tools](../concepts/tools) - LLM function calling
- [Docker Deployment](../deployment/docker) - Deploying with local models
