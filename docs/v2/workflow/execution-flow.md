# Execution flow

How kdeps resolves, orders, and runs resources in a workflow. `kdeps run` runs that DAG directly.

<div v-pre>

```yaml
# workflow.yaml (conceptual)
metadata:
  targetActionId: respond            # engine walks backward from here

resources:
  - actionId: fetchData
    component:
      name: http_request
      with:
        url: https://api.example.com/data

  - actionId: analyzeData
    flow:
      requires: [fetchData]            # runs only after fetchData succeeds
    chat:
      context:
        prompt: "Analyze: {{ output('fetchData') }}"

  - actionId: respond
    flow:
      requires: [analyzeData]          # runs only after analyzeData succeeds
    apiResponse:
      response: "{{ output('analyzeData') }}"
```

</div>

## Read next

- [Execution flow reference](/reference/workflow/execution-flow) - overview, execution order, dependency graph, cycle detection, skip vs check, loop execution
- [Workflow](/workflow/) - the YAML pipeline
- [Agent loop](/workflow/agent-loop/) - the model calls tools
- [Validation & control flow](/workflow/validation) - skip, check, and error handling
- [Loop](/workflow/loop) - a loop that repeats when a condition is true
