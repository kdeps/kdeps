# Concepts overview

A project is a folder. `workflow.yaml` names the workflow. Each file in `resources/` is one step, and kdeps runs the steps in `requires:` order.

```text
workflow.yaml -> resources/*.yaml (requires: order) -> targetActionId -> result
```

## Read next

- [Full text](/reference/start/concepts) - every concept, grouped
- [Workflow](/workflow/) - the YAML pipeline
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools
