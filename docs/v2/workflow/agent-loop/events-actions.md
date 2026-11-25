# Event actions

An action is what an [event](/workflow/agent-loop/events) runs when its limit is crossed. The model does not choose it.

```yaml
# ~/.kdeps/actions/force_answer.yaml (built-in default -- shown for reference)
name: force_answer
kind: rounds     # the trigger shape this action is meant for: tokens, bytes, distinctCalls, items, or rounds
description: End the tool rounds and force a final answer.
```

## Read next

- [Event actions reference](/reference/agent-loop/events-actions) - every key, option and detail
