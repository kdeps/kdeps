# Execution flow

kdeps orders the resources by `requires:` and runs them one after another. Each resource reads the outputs of the resources before it.

```text
request -> resource A -> resource B (requires: A) -> target resource -> result
```

## Read next

- [Execution flow reference](/reference/workflow/execution-flow) - every key, option and detail
- [Workflow](/workflow/) - the YAML pipeline
- [Agent loop](/workflow/agent-loop/) - the model calls tools
