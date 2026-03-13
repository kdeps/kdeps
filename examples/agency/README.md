# Agency Example

A simple two-agent agency that demonstrates:

- **`targetAgentId`** entrypoint declaration in `agency.yaml`
- **`agent` resource type** for inter-agent calls (using `name:` key)
- **Auto-discovered agents** under `agents/` subdirectory
- **Agency packaging** into a portable `.kagency` archive

## Structure

```
agency/
├── agency.yaml               # Agency manifest (lists agents, sets entry point)
└── agents/
    ├── greeter/
    │   └── workflow.yaml     # Entry-point agent (API server, calls responder)
    └── responder/
        └── workflow.yaml     # Helper agent (builds greeting message)
```

## How it works

1. `kdeps run agency.yaml` reads the agency manifest.
2. `targetAgentId: greeter-agent` identifies `agents/greeter/workflow.yaml` as the entry point.
3. Both agents are indexed by their `metadata.name`.
4. `greeter-agent` starts an API server on port 17100.
5. On each request `greeter-agent` uses the `agent` resource to call `responder-agent`,
   forwarding the `name` parameter.
6. `responder-agent` builds `"Hello, <name>!"` and returns it.
7. `greeter-agent` wraps the result in an `apiResponse`.

### Agent resource syntax

```yaml
run:
  agent:
    name: responder-agent   # target agent metadata.name (preferred over legacy "agent:" key)
    params:
      name: "{{ get('name') }}"
```

## Running

```bash
# Run the full agency (starts greeter-agent's API server)
kdeps run examples/agency/agency.yaml

# Query the greeter endpoint
curl "http://localhost:17100/api/v1/greet?name=Alice"
```

Expected response:

```json
{
  "success": true,
  "data": "Hello, Alice! (from responder-agent)"
}
```

## Packaging the agency (`.kagency`)

An agency directory can be packed into a single portable `.kagency` archive
(a tar.gz containing `agency.yaml` and the full `agents/` sub-tree).

```bash
# Package the agency → produces greeter-agency-1.0.0.kagency
kdeps package examples/agency/

# Run the packed agency
kdeps run greeter-agency-1.0.0.kagency

# Build a Docker image from the packed agency (uses the entry-point agent)
kdeps build greeter-agency-1.0.0.kagency

# Export as a bootable ISO
kdeps export iso greeter-agency-1.0.0.kagency

# Embed in a self-contained binary (no separate kdeps install needed)
kdeps prepackage greeter-agency-1.0.0.kagency --output my-greeter
./my-greeter   # auto-detects and runs the embedded .kagency
```

## Packed agents (`.kdeps`)

Individual agents can also be distributed as `.kdeps` archive files.  To use a packed agent:

1. Package the agent: `kdeps package agents/responder`
2. Place the resulting `responder-agent-1.0.0.kdeps` file in the `agents/` directory.
3. Reference it in `agency.yaml`:

```yaml
agents:
  - agents/greeter                      # directory-based agent
  - agents/responder-agent-1.0.0.kdeps  # packed agent archive
```
