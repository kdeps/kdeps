# LLM resource

The `chat:` resource sends a prompt to a language model and stores the reply as the resource's output.

```text
inputs -> prompt -> model (local or cloud) -> reply -> next resource
```

<div v-pre>

```yaml
# resources/answer.yaml
actionId: answer
chat:
  model:
    name: llama3.2:1b             # omit to use the machine default from ~/.kdeps/config.yaml
    timeout: 60s                  # fail the resource if the model has not answered in 60s
  context:
    prompt: "{{ get('q') }}"      # the request's q value
```

</div>

## Read next

- [LLM resource reference](/reference/resources/llm) - every key, option and detail
- [LLM backends](/workflow/resources/llm-backends) - local and cloud backends, API keys
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools over many rounds
