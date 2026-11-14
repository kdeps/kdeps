# Agent resource (delegation)

The `agent:` resource calls a sibling agent's entire workflow within the same [agency](/agencies/) and returns that agent's `apiResponse` output. It is how one agent delegates a subtask to another specialized agent.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). It is valid only between agents in the same `agency.yaml`. [Agencies](/agencies/) shows how to put agents in a group and how they find each other.

## Basic usage

```yaml
# resources/delegate.yaml
actionId: delegate
name: Ask the Research Agent
agent:
  name: research-agent          # metadata.name of the target agent in agency.yaml
  params:
    topic: "{{ get('q') }}"
    depth: deep
```

```yaml
# resources/respond.yaml
actionId: respond
flow:
  requires: [delegate]
apiResponse:
  success: true
  response:
    result: "{{ output('delegate') }}"
```

## Configuration options

| Option | Description |
|---|---|
| `name` | `metadata.name` of the target agent's workflow, as declared in `agency.yaml` (required) |
| `params` | Key-value pairs forwarded to the target agent. The target agent reads them via `get('key')`, same as it would read HTTP request fields |

## Output

The target agent's own `apiResponse` output, unchanged - whatever shape that agent returns is what `output('delegate')` gives you here.

## Parallel calls

`agent:` calls that don't depend on each other run at the same time, each on its own engine. A panel of reviewer agents takes as long as the slowest one:

<div v-pre>

```yaml
# resources/review-scope.yaml - one file per reviewer; all three start together
actionId: reviewScope
name: Scope review
agent:
  name: reviewer-agent
  params:
    focus: scope
    draft: "{{ get('q') }}"
---
# resources/report.yaml - starts once every review is back
actionId: report
name: Report
flow:
  requires: [reviewScope, reviewRisks, reviewClarity]
apiResponse:
  success: true
  response:
    scope: "{{ output('reviewScope') }}"
    risks: "{{ output('reviewRisks') }}"
    clarity: "{{ output('reviewClarity') }}"
```

</div>

[Execution flow](/workflow/execution-flow#independent-resources) tells how many run at the same time, and what runs alone.

## See also

- [Agencies](/agencies/) - multi-agent orchestration, and how to put agents in a bundle and name them
- [Component resource](/agencies/component-resource) - call a reusable resource bundle instead of a full sibling agent
- [Resources overview](/workflow/resources) - all resource types
