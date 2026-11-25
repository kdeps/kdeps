# Knowledge vault

The knowledge vault is a long-term knowledge base of markdown notes. Every `chat` call adds the notes related to its prompt. The recall makes no model call.

```text
chat prompt -> graph recall (no model) -> 3-8 matching notes (path + summary) -> model
```

<div v-pre>

```yaml
# resources/answer.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
  rag:
    knowledge: false     # skip recall for this call; default true
```

</div>

## Read next

- [Knowledge vault reference](/reference/agent-loop/knowledge) - every key, option and detail
