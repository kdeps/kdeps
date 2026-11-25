# Routing

With `name: router`, `~/.kdeps/config.yaml` picks the model for each request, not the resource.

```text
chat (model: router) -> ~/.kdeps/config.yaml strategy -> picked model -> backend
```

<div v-pre>

```yaml
# resources/llm.yaml
chat:
  model:
    name: router       # delegate to config.yaml router
  context:
    prompt: "{{ get('q') }}"
```

</div>

## Read next

- [Routing reference](/reference/resources/llm-routing) - every key, option and detail
- [LLM backends & routing](/workflow/resources/llm-backends) - backend configuration and the unified models list
