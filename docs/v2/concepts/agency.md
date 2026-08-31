# AI agencies

An AI agency is a collection of kdeps agents that cooperate to handle complex tasks without human-in-the-loop intervention. This is a workflow mode concept: each agent has its own `workflow.yaml` (its own resources, routes, and settings), bundled under a single `agency.yaml` manifest that makes them operate as one system.

An agency runs as a workflow. In agent mode, an agency is registered as an LLM tool and executes its full pipeline only when the model calls it.

## Use cases

Agencies are the natural evolution from single agents. Where a single agent handles one workflow, an agency coordinates many specialized agents - each one a reproducible, repeatable process - into a unified system that tackles multi-step problems.

| Single AI Agent | Autonomous AI Agency |
|---|---|
| One workflow file, one port | Multiple specialized agents coordinated within one agency (optionally runnable standalone) |
| All resources coupled together | Each agent is independently deployable and testable |
| Hard to reuse logic across projects | Agents can be packaged as `.kdeps` archives and reused |
| No inter-agent delegation | Agents delegate work to each other via `agent:` resource |
| Limited scope | Self-governing system handles complex end-to-end tasks autonomously |

## Directory structure

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

## Agency manifest (agency.yaml)

```yaml
# workflow-agency.yaml
apiVersion: kdeps.io/v1
kind: Agency

metadata:
  name: my-agency
  version: "1.0.0"
  description: "A multi-agent pipeline"
  # Entry-point agent - resolved by metadata.name in an agent's workflow.yaml.
  # If omitted, the first discovered agent is used.
  targetAgentId: greeter-agent

# Optional: explicit agent list.
# If omitted, all agents/ sub-directories and agents/*.kdeps are auto-discovered.
agents:
  - agents/greeter            # directory-based agent
  - agents/summariser         # directory-based agent
  - agents/packed-1.0.0.kdeps # packed agent archive
```

### Agent discovery

When the `agents:` list is **omitted**, kdeps auto-discovers agents in two ways:

1. **Directory-based** - any `agents/**/workflow.yaml` (or `.yml`, `.yaml.j2`, ...) is loaded.
2. **Packed archives** - any `agents/*.kdeps` file is extracted and its `workflow.yaml` is loaded.

When the `agents:` list is **provided**, only the listed entries are loaded (directories
or `.kdeps` archives). All listed paths are resolved relative to the agency directory.

## Running an agency

```bash
# Run from a directory containing agency.yaml
kdeps run my-agency/

# Run from an explicit manifest path
kdeps run my-agency/agency.yaml
```

## Inter-agent calls (agent:)

The `agent:` resource type is like calling a function where the function is an entire workflow. kdeps runs the target agent's full pipeline and returns its `apiResponse.response` as the output of the calling resource.

```d2
direction: right

A: "calling agent\nactionId: draft\nagent: name: summariser\nparams: text: ..."
B: "target agent\nworkflow.yaml\nname: summariser\nresources/...\napiResponse: ..."

A -> B: params
B -> A: output
```

```yaml
# resources/example.yaml
agent:
  name: summariser-agent   # matches metadata.name in the target's workflow.yaml
  params:
    text: "{{ get('body') }}"   # becomes get('text') inside the target agent
```

- `name:` resolves to the target agent by `metadata.name` in its `workflow.yaml`.
- `params:` are key-value pairs the target reads via `get('key')`.
- The caller reads the result via `output('actionId')` or `get('actionId')`.

## Packaging an agency (.kagency)

An entire agency - `agency.yaml` plus all `agents/` sub-trees - can be packed into a
single portable **`.kagency`** archive (a gzip-compressed tar).

```bash
# Pack the agency → produces my-agency-1.0.0.kagency
kdeps bundle package my-agency/

# Custom name / output directory
kdeps bundle package my-agency/ --name my-agency-1.0.0 --output dist/
```

The resulting `.kagency` archive can then be used just like a directory:

```bash
kdeps run     my-agency-1.0.0.kagency
kdeps bundle build   my-agency-1.0.0.kagency   # build Docker image
kdeps export iso my-agency-1.0.0.kagency # export bootable ISO
```

## Running as Docker

```bash
# Build a Docker image from the entry-point agent (greeter-agent in this example)
kdeps bundle build my-agency/

# Or from a packed archive
kdeps bundle build my-agency-1.0.0.kagency --tag myregistry/my-agency:latest
```

The generated Docker image runs the entry-point agent (`targetAgentId`) inside a
minimal Alpine/Ubuntu container with all dependencies pre-installed.

## Exporting as a bootable ISO

```bash
# Export to a bootable EFI ISO
kdeps export iso my-agency/

# Export from a packed archive
kdeps export iso my-agency-1.0.0.kagency --output my-agency.iso
```

The ISO boots a minimal LinuxKit system that runs the agency's entry-point agent as a
containerised service.

## Creating a self-contained binary

A `.kagency` archive (or a plain `.kdeps` workflow archive) can be embedded directly
into the kdeps binary, producing a **zero-dependency single binary**:

```bash
kdeps bundle prepackage my-agency-1.0.0.kagency --output dist/

# The binary auto-detects the embedded archive and runs it
./dist/my-agency-linux-amd64
```

When executed, the binary inspects its own bytes, extracts the embedded archive to a
temp directory, then runs it exactly as `kdeps run` would.

## Example: two-agent greeter

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
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run examples/agency/agency.yaml

# Query the API
curl "http://localhost:17100/api/v1/greet?name=Alice" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
# → {"success":true,"data":"Hello, Alice! (from responder-agent)"}
```

## See also

- [Agent resource](/resources/delegation/agent) - `agent:` resource reference
- [`examples/agency/`](https://github.com/kdeps/kdeps/tree/main/examples/agency) - runnable example
- [Packaging commands](/reference/cli/packaging) - `.kdeps` and `.kagency` formats
- [Docker deployment](../deployment/docker.md) - building Docker images
- [Standalone executables](../deployment/prepackage.md) - exporting self-contained binaries
