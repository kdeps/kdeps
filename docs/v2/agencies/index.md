# kdeps agencies

A kdeps agency composes several agents into one system. Each agent is its own `kind: Workflow` with its own model and resources, bundled under one `agency.yaml`. One agent delegates to another with the `agent:` resource, which runs that agent's whole pipeline and returns its result.

**Not this?** For one pipeline, start with the [workflow](/workflow/). A `chat` with `loop:` is the [agent loop](/workflow/agent-loop/). To reuse one bundle of resources, use a [component](/agencies/components).

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

```bash
kdeps run ./my-agency/          # runs the entry-point agent's pipeline
```

## Read next

- [kdeps agencies reference](/reference/agencies/) - directory layout, `agency.yaml`, inter-agent calls, packaging (`.kagency`), Docker, ISO and binaries
- [agent resource](/agencies/agent-resource) - call another agent
- [Components](/agencies/components) - reusable resource bundles
