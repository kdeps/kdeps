# Knowledge vault

The knowledge vault is a long-term knowledge base of markdown notes. Every `chat` call adds the notes related to its prompt: kdeps follows the link graph of the vault from the words of the prompt, with no model call, and puts the best note paths and summaries after the system prompt.

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

`knowledge.enabled: false` in `~/.kdeps/config.yaml` turns recall off for every call.

## Read next

- [Knowledge vault reference](/reference/agent-loop/knowledge) - one home per fact, projects, set up, cards, notes, workflow mode, the graph, model extraction
