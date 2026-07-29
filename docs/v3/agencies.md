# Agencies

An **agency** is several workflows that collaborate. Each agent stays a normal kdeps project; the agency wires them.

| Single agent | Agency |
|--------------|--------|
| One workflow | Specialized agents |
| Coupled resources | Independently testable agents |
| Copy-paste reuse | Package / share agents |
| No delegation | `agent:` resource calls another agent |

## Layout

```text
my-agency/
├── agency.yaml
└── agents/
    ├── greeter/
    │   ├── workflow.yaml
    │   └── resources/
    ├── researcher/
    │   └── …
    └── writer/
        └── …
```

## agency.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Agency

metadata:
  name: my-agency
  version: "1.0.0"
  description: "Research and write"
  targetAgentId: greeter-agent   # metadata.name of entry agent

agents:
  - agents/greeter
  - agents/researcher
  - agents/writer
# omit agents: to auto-discover agents/*
```

`targetAgentId` must match an agent's `metadata.name`.

## Delegate with `agent:`

```yaml
actionId: delegateResearch
requires: [understand]
agent:
  target: researcher-agent
  input: "{{ get('userQuery') }}"
```

1. Load target workflow  
2. Pass `input` (like request body)  
3. Run full DAG  
4. Return that agent's `apiResponse` as this resource's output  

Chain with `requires:` like any other step.

## Run and package

```bash
kdeps validate ./my-agency/
kdeps run ./my-agency/                 # or path to agency.yaml / .kagency
kdeps ./my-agency/                     # agent mode: entry + tools
kdeps bundle package ./my-agency/
kdeps bundle build ./my-agency/
```

[Workflow mode](/workflow) · [Coding agent](/agent) · [Components](/components).
