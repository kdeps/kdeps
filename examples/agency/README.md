# Agency Example

A simple two-agent agency that demonstrates:

- **`targetAgentId`** entrypoint declaration in `agency.yaml`
- **`agent` resource type** for inter-agent calls
- **Auto-discovered agents** under `agents/` subdirectory

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

## Packed agents (`.kdeps`)

Agents can also be distributed as `.kdeps` archive files.  To use a packed agent:

1. Build the agent: `kdeps build agents/responder`
2. Place the resulting `responder-1.0.0.kdeps` file in the `agents/` directory.
3. Reference it in `agency.yaml`:

```yaml
agents:
  - agents/greeter            # directory-based agent
  - agents/responder-1.0.0.kdeps  # packed agent archive
```
