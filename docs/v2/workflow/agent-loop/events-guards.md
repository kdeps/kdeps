# Event guards

These events fire on a round count, a truncation size, a call budget, or a memory cap. The two token events, auto-compact and fold, stay on [Reactive LLM events](/workflow/agent-loop/events).

```yaml
# ~/.kdeps/events/identical-tool-calls.yaml
name: identical-tool-calls
on:
  rounds: 3   # how many times in a row the model may issue the exact same tool call
run: force_answer
```

## Read next

- [Event guards reference](/reference/agent-loop/events-guards) - round-count events, A sensible floor of 1, truncation events, call-budget events, your own tool budgets, memory events, thinking budgets
