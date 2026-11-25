# Fenced tools

Each model gets an instruction to use the fenced kdeps tools and to say what it will do before each call. The tool list is on [Built-in tools](/workflow/agent-loop/tools).

```text
model: "Created [notes.md](...)"          <- no tool ran this turn
kdeps: nudge-unbacked-claim               <- "You made no tool call this turn..."
model: <invoke name="write_file">...      <- the real call
kdeps: write_file result -> model answers
```

## Read next

- [Fenced tools reference](/reference/agent-loop/fenced-tools) - use the fenced kdeps tools, not a built-in sandbox, claimed actions need a tool result, parser and interpreter, the available_tools block, simulated sandbox sessions are caught, premature refusals after real progress, fictional "I can't", continue means do the next action
