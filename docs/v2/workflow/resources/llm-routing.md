# Routing

Routing delegates model selection from resource YAML to [config](/workflow/resources/llm-backends). It applies in both modes, wherever a `chat:` resource runs. Set a resource's `model` field to `router`:

```yaml
# resources/llm.yaml
chat:
  model:
    name: router       # delegate to config.yaml router
  context:
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

kdeps estimates token counts with tiktoken.

## Strategy: `fallback`

Tries routes in priority order. On error, automatically retries the next route.

```yaml
# resources/example.yaml
llm:
  strategy: fallback
  models:
    - model: claude-sonnet-4-6
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

kdeps tries lower priority values first. `default: true` marks the catch-all route.

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

kdeps uses zero for a nil cost. If two routes have the same cost, kdeps uses `default: true`.

## Strategy: `round_robin`

Distributes requests evenly across models using an atomic counter.

```yaml
# resources/example.yaml
llm:
  strategy: round_robin
  models:
    - model: gpt-4o
      backend: openai
    - model: claude-sonnet-4-6
      backend: anthropic
```

The key of each counter is a fingerprint of the model list. As a result, different route configurations keep independent counters.

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

kdeps gives a score only to entries on a local backend (`file`, `gguf`, `ollama`), because `llmfit` measures hardware fit. kdeps uses a cloud entry only through `default: true`, when no local entry gets a score. If `llmfit` is not installed, routing uses the fallback in the same way. kdeps calculates the `llmfit` index one time for each process and keeps it in a cache.

Agent mode has the same strategy with `--model auto` (or `KDEPS_AGENT_MODEL=auto`) - see [Local model management](/agent/models#how-a-model-is-picked-when-none-is-configured).

## `auto-router`: zero-config, fully automatic

`auto` scores *your* `models:` list under `llm:`: you pick the candidates, `auto` picks among them. `auto-router` needs no `models:` entry. Set a resource's `model` field to `auto-router` and kdeps discovers everything itself, every time it runs:

```yaml
# resources/llm.yaml
chat:
  model:
    name: auto-router   # zero config -- ignores llm.models/router entirely
  context:
    role: user
    prompt: "{{ get('q') }}"
```

Resolution order, with no config required:

1. **Best-fit installed local model** - `llmfit` gives a score to each cached llamafile, loadable GGUF, and pulled Ollama tag, as in the local tier of `auto`. This step needs `llmfit` on `PATH`. If `llmfit` is not installed, kdeps skips this step (no cost).
2. **Cloud fallback** - the first provider that has an API key env var (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, ...) and a known representative model (`gpt-4o` for OpenAI, `claude-sonnet-4-6` for Anthropic, ...). kdeps does not use providers with no single default model (OpenRouter, Hugging Face, Bedrock, Watsonx, ...). The list includes the free models of OpenRouter, from `models/openrouter`.
3. **Built-in default** - if the two steps above find nothing, kdeps uses the zero-config built-in llamafile (`llama3.2:1b`). This is the same model that kdeps uses when you do not set `model:`.

Agent mode has the same sentinel with `--model auto-router` - see [Local model management](/agent/models#-model-auto-router-zero-config-fully-automatic).

## See also

- [LLM backends & routing](/workflow/resources/llm-backends) - backend configuration and the unified models list
- [Local model management](/agent/models) - the agent-loop equivalent of `auto`/`auto-router`
