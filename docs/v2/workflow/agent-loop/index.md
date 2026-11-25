# Agent loop

The agent loop is a workflow `chat` with [`loop:`](/reference/resources/llm#agent-loop). It calls tools over many rounds until the model gives a final answer.

```text
input -> chat (loop:) -> tool rounds -> answer -> next resource
```

<div v-pre>

```yaml
# resources/assistant.yaml
actionId: assistant
chat:
  context:
    prompt: "{{ input('message') }}"
  loop:
    tools: [read_file, edit_file, bash_exec]   # omit to offer every built-in tool
```

</div>

Run `kdeps new my-agent --template assistant` to get a working agent.

## Read next

- [Agent loop reference](/reference/agent-loop/) - every key, option and detail
- [Workflow](/workflow/) - the YAML pipeline
- [Input sources](/workflow/input-sources) - API, bot, file, or interactive prompt
