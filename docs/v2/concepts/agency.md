# Agency — Multi-Agent Orchestration

An **agency** is a collection of kdeps agents that cooperate to handle complex tasks.
Each agent in the agency has its own `workflow.yaml` (its own resources, routes, and
settings), but they are bundled together under a single `agency.yaml` manifest.

## Why Use an Agency?

| Single workflow | Agency |
|---|---|
| One workflow file, one port | Multiple specialised agents, each on their own port |
| All resources coupled together | Each agent is independently deployable and testable |
| Hard to reuse logic across projects | Agents can be packaged as `.kdeps` archives and reused |
| No inter-agent delegation | Resources can delegate work to other agents via `run.agent:` |

## Directory Structure

```
my-agency/
├── agency.yaml               # Agency manifest
└── agents/
    ├── greeter/
    │   ├── workflow.yaml     # Entry-point agent
    │   └── resources/
    ├── summariser/
    │   ├── workflow.yaml
    │   └── resources/
    └── packed-helper-1.0.0.kdeps   # Packed agent archive
```

## Agency Manifest (`agency.yaml`)

```yaml
apiVersion: kdeps.io/v1
kind: Agency

metadata:
  name: my-agency
  version: "1.0.0"
  description: "A multi-agent pipeline"
  # Entry-point agent — resolved by metadata.name in an agent's workflow.yaml.
  # If omitted, the first discovered agent is used.
  targetAgentId: greeter-agent

# Optional: explicit agent list.
# If omitted, all agents/ sub-directories and agents/*.kdeps are auto-discovered.
agents:
  - agents/greeter            # directory-based agent
  - agents/summariser         # directory-based agent
  - agents/packed-1.0.0.kdeps # packed agent archive
```

### Agent Discovery

When the `agents:` list is **omitted**, kdeps auto-discovers agents in two ways:

1. **Directory-based** — any `agents/**/workflow.yaml` (or `.yml`, `.yaml.j2`, …) is loaded.
2. **Packed archives** — any `agents/*.kdeps` file is extracted and its `workflow.yaml` is loaded.

When the `agents:` list is **provided**, only the listed entries are loaded (directories
or `.kdeps` archives). All listed paths are resolved relative to the agency directory.

## Running an Agency

```bash
# Run from a directory containing agency.yaml
kdeps run my-agency/

# Run from an explicit manifest path
kdeps run my-agency/agency.yaml
```

## Inter-Agent Calls (`run.agent:`)

Resources within one agent can delegate work to another agent in the same agency using
the `agent` resource type.

```yaml
run:
  agent:
    name: summariser-agent   # metadata.name of the target agent's workflow
    params:
      text: "{{ get('body') }}"
```

- `name:` — resolves to the target agent by `metadata.name` in its `workflow.yaml`.
  The legacy `agent:` key is also accepted for backward compatibility.
- `params:` — key-value pairs forwarded as input to the target agent (accessible via
  `get('key')` inside the target).
- The return value is the first `apiResponse.response` produced by the target, accessible
  via `output('actionId')` in the calling resource.

## Packaging an Agency (`.kagency`)

An entire agency — `agency.yaml` plus all `agents/` sub-trees — can be packed into a
single portable **`.kagency`** archive (a gzip-compressed tar).

```bash
# Pack the agency → produces my-agency-1.0.0.kagency
kdeps package my-agency/

# Custom name / output directory
kdeps package my-agency/ --name my-agency-1.0.0 --output dist/
```

The resulting `.kagency` archive can then be used just like a directory:

```bash
kdeps run     my-agency-1.0.0.kagency
kdeps build   my-agency-1.0.0.kagency   # build Docker image
kdeps export iso my-agency-1.0.0.kagency # export bootable ISO
```

## Running as Docker

```bash
# Build a Docker image from the entry-point agent (greeter-agent in this example)
kdeps build my-agency/

# Or from a packed archive
kdeps build my-agency-1.0.0.kagency --tag myregistry/my-agency:latest
```

The generated Docker image runs the entry-point agent (`targetAgentId`) inside a
minimal Alpine/Ubuntu container with all dependencies pre-installed.

## Exporting as a Bootable ISO

```bash
# Export to a bootable EFI ISO
kdeps export iso my-agency/

# Export from a packed archive
kdeps export iso my-agency-1.0.0.kagency --output my-agency.iso
```

The ISO boots a minimal LinuxKit system that runs the agency's entry-point agent as a
containerised service.

## Creating a Self-Contained Binary

A `.kagency` archive (or a plain `.kdeps` workflow archive) can be embedded directly
into the kdeps binary, producing a **zero-dependency single binary**:

```bash
kdeps prepackage my-agency-1.0.0.kagency --output my-agency-binary

# The binary auto-detects the embedded archive and runs it
./my-agency-binary
```

When executed, the binary inspects its own bytes, extracts the embedded archive to a
temp directory, then runs it exactly as `kdeps run` would.

## Example: Two-Agent Greeter

The `examples/agency/` directory ships a minimal two-agent example:

```
examples/agency/
├── agency.yaml
└── agents/
    ├── greeter/workflow.yaml    # API server, calls responder
    └── responder/workflow.yaml  # Builds the greeting string
```

```bash
# Run the example
kdeps run examples/agency/agency.yaml

# Query the API
curl "http://localhost:17100/api/v1/greet?name=Alice"
# → {"success":true,"data":"Hello, Alice! (from responder-agent)"}
```

## See Also

- [Agent resource](../resources/overview.md#agent) — `run.agent:` reference
- [`examples/agency/`](https://github.com/kdeps/kdeps/tree/main/examples/agency) — runnable example
- [Packaging workflows](../getting-started/cli-reference.md) — `.kdeps` and `.kagency` formats
- [Docker deployment](../deployment/docker.md) — building Docker images
- [Standalone executables](../deployment/prepackage.md) — exporting self-contained binaries
