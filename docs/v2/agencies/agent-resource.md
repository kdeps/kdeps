# Agent resource (delegation)

The `agent:` resource calls a sibling agent's entire workflow within the same [agency](/agencies/) and returns that agent's `apiResponse` output.

<div v-pre>

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

</div>

## Read next

- [Agent resource (delegation) reference](/reference/agencies/agent-resource) - every key, option and detail
- [Agencies](/agencies/) - multi-agent orchestration, and how to put agents in a bundle and name them
- [Component resource](/agencies/component-resource) - call a reusable resource bundle instead of a full sibling agent
