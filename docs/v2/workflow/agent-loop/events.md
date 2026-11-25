# Reactive LLM events

An event monitors one number. When the number crosses a limit, the event runs one action. Every limit of the agent loop is an event in YAML.

```text
measure tokens --> compare to threshold --> fire action (compact / fold)
```

```yaml
# ~/.kdeps/events/auto-compact.yaml
name: auto-compact
on:
  ctxWindowFraction: 0.75  # fire when the context window is 75% full
run: compact               # summarize the old turns
```

## Read next

- [Reactive LLM events reference](/reference/agent-loop/events) - every key, option and detail
