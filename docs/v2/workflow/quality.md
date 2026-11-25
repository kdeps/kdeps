# Answer quality

A workflow `chat` turns on kdeps's quality features with keys in its `harness` and `guardrails` groups. They help a small local model answer better.

```text
prompt -> model call -> tool loop (promptMode) -> judges (review, revise) -> output
```

```yaml
# resources/answer.yaml
chat:
  harness:
    sections: [accuracy, honesty]   # system-prompt sections
  guardrails:
    judges: auto                    # review the answer before it returns
```

## Read next

- [Answer quality reference](/reference/workflow/quality) - every key, option and detail
- [LLM resource](/workflow/resources/llm) - every chat key
- [Reactive LLM events](/workflow/agent-loop/events) - the limits
