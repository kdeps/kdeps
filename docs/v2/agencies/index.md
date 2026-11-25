# kdeps agencies

A kdeps agency composes several agents into one system. Each agent is a workflow. One agent calls another with the `agent:` resource.

```d2
direction: right

A: "calling agent\nagent: name: summariser"
B: "target agent\nits own workflow.yaml"

A -> B: params
B -> A: output
```

<div v-pre>

```yaml
# resources/draft.yaml - in the calling agent
agent:
  name: summariser-agent        # metadata.name in the target's workflow.yaml
  params:
    text: "{{ get('body') }}"   # becomes get('text') inside the target agent
```

</div>

## Read next

- [kdeps agencies reference](/reference/agencies/) - every key, option and detail
- [agent resource](/agencies/agent-resource) - call another agent
- [Components](/agencies/components) - reusable resource bundles
