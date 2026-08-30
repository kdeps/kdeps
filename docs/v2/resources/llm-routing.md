# Routing

Routing delegates model selection from resource YAML to [config](/resources/llm-backends). Set a resource's `model` field to `router`:

```yaml
# resources/llm.yaml
chat:
  model: router       # delegate to config.yaml router
  role: user
  prompt: "{{ get('q') }}"
```

The router in `~/.kdeps/config.yaml` selects which model to use based on the configured strategy.

## Strategy: `token_threshold`

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

## Strategy: `fallback`

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

## Strategy: `cost_optimized`

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

## Strategy: `round_robin`

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

## Strategy: `auto`

Picks the best hardware-fit model among `models` using [`llmfit`](https://github.com/AlexsJones/llmfit) (`brew install AlexsJones/llmfit/llmfit`), instead of a fixed rule like `token_threshold`/`cost_optimized`/`round_robin`.

```yaml
# resources/example.yaml
llm:
  strategy: auto
  models:
    - model: llama3.2:1b
      backend: file
    - model: qwen2.5:7b
      backend: gguf
    - model: gpt-4o
      backend: openai
      default: true          # only ever used as the fallback -- see below
```

Only entries on a local backend (`file`, `gguf`, `ollama`) are scored -- `llmfit` measures hardware fit, which is meaningless for a remote API model, so a cloud entry is never *preferred*, only ever reached via `default: true` when no local entry scores. Falls back the same way when `llmfit` isn't installed. The `llmfit` index is computed once per process and cached, not re-run on every request.

Agent-loop mode has the same strategy available via `--model auto` (or `KDEPS_AGENT_MODEL=auto`) -- see [Local Model Management](/modes/agent-loop-models#how-a-model-is-picked-when-none-is-configured).

## `auto-router`: zero-config, fully automatic

`auto` still scores *your configured* `llm.models` -- you pick the candidates, `auto` picks among them. `auto-router` needs no `llm.models` entry at all. Set a resource's `model` field to `auto-router` and kdeps discovers everything itself, every time it runs:

```yaml
# resources/llm.yaml
chat:
  model: auto-router   # zero config -- ignores llm.models/router entirely
  role: user
  prompt: "{{ get('q') }}"
```

Resolution order, with no config required:

1. **Best-fit installed local model** -- every cached llamafile, loadable GGUF, and pulled Ollama tag is scored via `llmfit`, same as `auto`'s local tier. Requires `llmfit` on `PATH`; skipped entirely (no cost) when it isn't installed.
2. **Cloud fallback** -- the first provider with both an API key env var set (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, ...) and a known representative model (`gpt-4o` for OpenAI, `claude-sonnet-4-6` for Anthropic, etc.) is used. Providers that host many models with no single canonical default (OpenRouter, Hugging Face, Bedrock, Watsonx, ...) don't participate in this fallback.
3. **Built-in default** -- if neither of the above finds anything, kdeps falls back to the same zero-config built-in llamafile (`llama3.2:1b`) used when `model:` is omitted entirely.

Agent-loop mode has the same sentinel via `--model auto-router` -- see [Local Model Management](/modes/agent-loop-models#-model-auto-router-zero-config-fully-automatic).

## See also

- [LLM Backends & Routing](/resources/llm-backends) -- backend configuration and the unified models list
- [Local Model Management](/modes/agent-loop-models) -- the agent-loop equivalent of `auto`/`auto-router`
