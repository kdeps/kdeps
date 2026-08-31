# Agent resource (delegation)

The `agent:` resource calls a sibling agent's entire workflow within the same [agency](/concepts/agency) and returns that agent's `apiResponse` output. It is how one agent delegates a subtask to another specialized agent.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). Only valid between agents bundled in the same `agency.yaml` - see [Agencies](/concepts/agency) for how agents are grouped and discover each other.

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

## See also

- [Agencies](/concepts/agency) - multi-agent orchestration, how agents are bundled and named
- [Component resource](component) - call a reusable resource bundle instead of a full sibling agent
- [Resources overview](overview) - all resource types
