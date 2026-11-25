# Fenced tools

kdeps tells every model to use its fenced kdeps tools, not a built-in sandbox. When a model claims work but calls no tool, kdeps tells it to make the call.

```text
model: "Created [notes.md](...)"          <- no tool ran this turn
kdeps: nudge-unbacked-claim               <- "You made no tool call this turn..."
model: <invoke name="write_file">...      <- the real call
kdeps: write_file result -> model answers
```

## Read next

- [Fenced tools reference](/reference/agent-loop/fenced-tools) - every key, option and detail
