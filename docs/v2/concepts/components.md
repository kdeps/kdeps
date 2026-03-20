# Components

Components are reusable, shareable subsets of a workflow that expose a clean interface (named inputs) and encapsulate resources and UI assets. They enable modular architecture and code reuse across multiple KDeps projects.

## Overview

A component is defined by a `component.yaml` manifest and lives in a `components/<name>/` directory alongside your workflow. When the workflow runs, all component resources are automatically loaded and merged with the workflow's own resources.

### Key Benefits

- **Reusability**: Use the same component across multiple workflows
- **Encapsulation**: Hide implementation details; expose only inputs
- **Shareable**: Package as `.komponent` archives for distribution
- **Auto-discovery**: No need to declare components in `workflow.yaml`; just place them in `components/`

## Component Structure

```
my-workflow/
├── workflow.yaml
├── resources/
└── components/                  ← auto-discovered
    └── greeter/
        ├── component.yaml       ← component manifest
        ├── resources/           ← component-specific resources
        │   └── greet.yaml
        └── template.html        ← optional UI template
```

## component.yaml Reference

```yaml
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter                # Required: component name
  description: "A greeting component"
  version: "1.0.0"
  targetActionId: greet        # Optional: default action to invoke
interface:
  inputs:
    - name: message            # Required: input parameter name
      type: string             # Required: string, integer, number, boolean
      required: true           # Optional (default: false)
      description: "Message to greet with"
      default: "Hello"         # Optional: default value if not required
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: greet
    # ... resource definition
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | yes | Must be `kdeps.io/v1` |
| `kind` | string | yes | Must be `Component` |
| `metadata.name` | string | yes | Unique component name within the workflow |
| `metadata.version` | string | no | Component version (used for packaging) |
| `metadata.targetActionId` | string | no | Default resource `actionId` when component is invoked |
| `interface.inputs[].name` | string | yes | Input parameter name |
| `interface.inputs[].type` | enum | yes | Data type: `string`, `integer`, `number`, `boolean` |
| `interface.inputs[].required` | bool | no | Whether input is required (default: false) |
| `interface.inputs[].description` | string | no | Human-readable description |
| `interface.inputs[].default` | any | no | Default value (only meaningful when required=false) |
| `resources` | array | no | Inline resource definitions (same as workflow resources) |

## Interface and Inputs

The `interface` section defines the component's public contract — the named parameters that parent workflows can provide. Inputs behave like function arguments:

```yaml
interface:
  inputs:
    - name: user_query
      type: string
      required: true
    - name: temperature
      type: number
      required: false
      default: 0.7
```

When the parent workflow calls the component's target action, it supplies these inputs. The component's resources can reference them via expressions:

```yaml
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: greet
    run:
      chat:
        model: "llama3.2:latest"
        prompt: "{{ inputs.message }}"
```

## Resources

Component resources can be defined inline in `component.yaml` or placed in the `resources/` subdirectory. The `actionId` **must be unique** across the entire workflow (component resources take precedence on conflict, but avoid collisions).

> **Note**: Components cannot contain `settings` (no server modes, port bindings, etc.). They are purely resource bundles.

## Auto-Discovery

When a workflow is parsed, KDeps automatically scans for `components/<name>/component.yaml` files in the same directory as the workflow. All resources from each component are prepended to the workflow's resource list, making them available as if they were defined locally.

- No explicit declaration in `workflow.yaml` is needed
- Local workflow resources override component resources on `actionId` collision
- Component loading happens recursively (imported workflows can also have components)

### Resolution Order

1. Workflow's inline resources
2. Resources from imported workflows (`metadata.workflows`)
3. Resources from auto-discovered components (alphabetical by component name)
4. Local workflow `resources/` directory

## Packaging Components

Use `kdeps package` to create a portable `.komponent` archive:

```bash
# From a component directory (containing component.yaml)
kdeps package ./my-component

# Output: my-component-1.0.0.komponent
```

The resulting archive can be:
- Shared with other developers
- Stored in a component registry
- Used as a dependency in agency workflows (extracted at runtime)

### Archive Contents

```
my-component-1.0.0.komponent
├── component.yaml
├── resources/
│   └── greet.yaml
├── template.html
└── (other data files, scripts, etc.)
```

Hidden files and `.kdepsignore` patterns are respected.

## Environment Variable Auto-Derivation

After component resources are loaded, KDeps automatically scans for environment variable references:

- String fields containing `{{ env('VAR_NAME') }}`
- `ChatConfig.APIKey` with `env:` prefix
- `MCPConfig.Env` and `PythonConfig.Env` maps
- Any `env('VAR')` expression in resource fields

Detected variables are tracked as required by the component. The parent workflow must provide them via `agentSettings.env` or the runtime environment. This ensures all external dependencies are explicit.

## Example: Greeter Component

**`components/greeter/component.yaml`**
```yaml
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter
  version: "1.0.0"
  targetActionId: greet
interface:
  inputs:
    - name: message
      type: string
      required: true
      description: Greeting message
    - name: recipient
      type: string
      required: false
      default: "World"
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: greet
    run:
      exec:
        command: "echo '{{ inputs.message }}, {{ inputs.recipient }}!'"
```

**`my-workflow/workflow.yaml`**
```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-workflow
  version: "1.0.0"
  targetActionId: main
settings:
  agentSettings:
    timezone: "UTC"
    offlineMode: true
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: main
    run:
      exec:
        command: "kdeps run greet --message 'Hello'"
```

Running `kdeps run my-workflow/` will automatically load the `greeter` component and make its `greet` action available.

## Best Practices

1. **Keep components focused**: Each component should have a single responsibility
2. **Version your components**: Use semantic versioning in `metadata.version`
3. **Document inputs**: Provide clear `description` fields for each input
4. **Use targetActionId**: Set it to the primary action for easy invocation
5. **Avoid actionId collisions**: Prefix actionIds with the component name (e.g., `greeter-greet`)
6. **Test in isolation**: Package and validate your component before using it in workflows
7. **Minimize env vars**: Declare all required environment dependencies; let auto-derivation detect them
