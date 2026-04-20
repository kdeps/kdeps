# config-namespace

Demonstrates the **config namespace** feature — reading global settings from
`~/.kdeps/config.yaml`, workflow metadata, and resource data directly inside
kdeps expressions.

## Namespace objects

Five namespaces are injected into every expression environment:

| Namespace | Source |
|-----------|--------|
| `config`  | `~/.kdeps/config.yaml` |
| `workflow`| current `workflow.yaml` metadata |
| `resource`| resources loaded into the workflow |
| `component` | components loaded into the workflow |
| `agency`  | current agency (when running inside an agency) |

## Usage in YAML

```yaml
# Direct property access
model: "{{ config.llm.model }}"
name:  "{{ workflow.metadata.name }}"

# get() with namespace path — supports a fallback default
model: "{{ get('config.llm.model', 'llama3.2') }}"

# set() mutates the value at runtime
run:
  exec:
    command: "echo '{{ set(\"config.llm.model\", \"gpt-4o\") }}'"
```

## Run

```bash
kdeps run examples/config-namespace
curl http://localhost:16395/api/v1/config
```

Expected response:

```json
{
  "workflow_name": "config-namespace",
  "workflow_version": "1.0.0",
  "model": "llama3.2",
  "timezone": "UTC",
  "model_from_exec": "llama3.2"
}
```

## Configuration

Values read from `~/.kdeps/config.yaml`. If the file is absent or a field is
unset, the `get()` fallback default is used instead.
