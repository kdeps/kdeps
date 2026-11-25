# Judge panel

A judge panel reviews an answer before the chat returns it. A rejected answer is revised and judged again.

```text
answer -> [judge 1] -> [judge 2] -> [judge 3] -> all approved? -> answer
              |             |             |
              v (tools)     v (tools)     v (tools)
          read files    run tests    search the web
                    any rejected -> revise -> re-judge
```

```yaml
# resources/answer.yaml
chat:
  guardrails:
    judges: auto      # kdeps picks the judges for the task
```

## Read next

- [Judge panel reference](/reference/agent-loop/judges) - every key, option and detail
- [Agent loop](/workflow/agent-loop/) - overview
