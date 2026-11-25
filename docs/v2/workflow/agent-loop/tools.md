# Built-in tools

The [agent loop](/workflow/agent-loop/) can call a set of built-in tools: files, edits, shell, web search, memory, data and actions. Each tool is a YAML definition in the `tools` asset set. kdeps registers a tool that needs credentials only when you set them.

```yaml
# resources/assistant.yaml
chat:
  loop:
    tools: [read_file, edit_file, bash_exec]   # by name; omit to offer every built-in tool
```

Your own tools are YAML too: a declared `http:`, `sql:`, `exec:` or `python:` block, with no Go code.

## Read next

- [Built-in tools reference](/reference/agent-loop/tools) - asking the user, tool definitions, tool aliases, declared HTTP tools, declared SQL tools, declared exec tools, declared python tools, your own tools
- [Fenced tools](/workflow/agent-loop/fenced-tools) - invoke blocks and narration
- [Harness reminders](/reference/agent-loop/tool-reference#harness-reminders) - force a guidance section onto every turn
- [File tools](/reference/agent-loop/tool-reference#file-tools) - read, write, list
- [Edit file](/reference/agent-loop/tool-reference#edit-file) - replace or append on one line
- [Memory and identity](/reference/agent-loop/tool-reference#memory-and-identity) - memory_save, memory_search, memory_query, identity_get
- [Shell execution](/reference/agent-loop/tool-reference#shell-execution) - bash_exec, jobs
- Web and search - web_search, wikipedia, web_scraper
- [Data tools](/reference/agent-loop/tool-reference#data-tools) - calculator, SQL, embeddings, rerank
- [Action tools](/reference/agent-loop/tool-reference#action-tools) - HTTP, local search, documents, Zapier, git trailer
- [Agent loop](/workflow/agent-loop/) - overview
