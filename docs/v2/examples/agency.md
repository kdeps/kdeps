# Compose two agents into an agency

Build an agency: two agents in one project, where the
entry-point agent calls the other and returns the combined result. Each agent
is its own `workflow.yaml`. An `agency.yaml` manifest ties them together.

The agency manifest lists the agents and names the entry point:

```yaml
# agency.yaml
apiVersion: kdeps.io/v1
kind: Agency

metadata:
  name: greeter-agency
  version: "1.0.0"
  targetAgentId: greeter-agent   # the agent that serves HTTP

agents:
  - agents/greeter               # relative to this file
  - agents/responder
```

## Read next

- [Full walkthrough](/reference/examples/agency) - every step, file and command
- [AI agencies](/agencies/) - discovery, packaging, `.kagency`
- [Agent resource](/agencies/agent-resource) - the `agent:` reference
