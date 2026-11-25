# LLM resource

The `chat:` resource sends a prompt to a language model and stores the reply as the resource's output.

```text
inputs -> prompt -> model (local or cloud) -> reply -> next resource
```

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

The reply text is at `get('answer').message.content`. With `json: true` under `harness.output`, the output is the parsed JSON object instead.

## Key groups

A chat's keys sit in six groups. The [LLM resource reference](/reference/resources/llm) lists every key.

| Group | Holds |
|---|---|
| `model` | which model, the backend, timeout, streaming, thinking, sampling |
| `context` | the prompt, role, message history, files, few-shot examples |
| `harness` | harness sections, prompt mode, output format (JSON, keys, schema) |
| `tools` | the tools the model can call, and how many rounds |
| `rag` | knowledge recall and retrieved passages |
| `guardrails` | judges that review the reply |

`loop:` sits beside the groups. It runs the chat as the [agent loop](/workflow/agent-loop/): the model calls tools over many rounds.

## Where config lives

The model goes in the resource. The backend and API keys go in `~/.kdeps/config.yaml`, so you can change backends with no change to the workflow. See [LLM backends](/workflow/resources/llm-backends).

## Read next

- [LLM resource reference](/reference/resources/llm) - every key, defaults, model resolution, tools, JSON output, judges, RAG, vision
- [LLM backends](/workflow/resources/llm-backends) - local and cloud backends, API keys
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools over many rounds
