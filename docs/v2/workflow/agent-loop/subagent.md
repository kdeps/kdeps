# Subagents

A subagent is a fresh agent loop. It starts without the conversation and sees only its task. The model starts one with the `subagent` tool.

```text
turn -> subagent tool (task) -> fresh loop (read-only tools) -> report -> turn
```

```yaml
# ~/.kdeps/config.yaml
llm:
  subagent: current    # current | free | a model id
```

## Read next

- [Subagents reference](/reference/agent-loop/subagent) - every key, option and detail
