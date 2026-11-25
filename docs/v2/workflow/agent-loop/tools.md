# Built-in tools

The [agent loop](/workflow/agent-loop/) can call a set of built-in tools: files, edits, shell, web search, memory, data and actions. Each tool is a YAML definition in the `tools` asset set. kdeps registers a tool that needs credentials only when you set them.

```yaml
# resources/assistant.yaml
chat:
  loop:
    tools: [read_file, edit_file, bash_exec]   # by name; omit to offer every built-in tool
```

## Read next

- [Built-in tools reference](/reference/agent-loop/tools) - every key, option and detail
- [Fenced tools](/workflow/agent-loop/fenced-tools) - invoke blocks and narration
- [Agent loop](/workflow/agent-loop/) - overview
