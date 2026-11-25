# Routing

```text
chat (model: router) -> ~/.kdeps/config.yaml strategy -> picked model -> backend
```

Routing delegates model selection from resource YAML to [config](/workflow/resources/llm-backends). It applies wherever a `chat:` resource runs. Set a resource's `model` field to `router`:

<div v-pre>

```yaml
# resources/llm.yaml
chat:
  model:
    name: router       # delegate to config.yaml router
  context:
    role: user
    prompt: "{{ get('q') }}"
```

</div>

## Read next

- [Routing reference](/reference/resources/llm-routing) - the strategies (`token_threshold`, `fallback`, `cost_optimized`, `round_robin`, `auto`), `auto-router`, and resolution order
- [LLM backends & routing](/workflow/resources/llm-backends) - backend configuration and the unified models list
