# Event guards

Guard events stop a loop that repeats itself or uses too much: rounds, output size, call budgets, memory.

```yaml
# ~/.kdeps/events/identical-tool-calls.yaml
name: identical-tool-calls
on:
  rounds: 3   # how many times in a row the model may issue the exact same tool call
run: force_answer
```

## Read next

- [Event guards reference](/reference/agent-loop/events-guards) - every key, option and detail
