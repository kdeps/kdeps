# component-input-source

Demonstrates the **`component` input source** - a workflow designed to be invoked exclusively via `run.component` from a parent workflow.

When `sources: [component]` is declared, kdeps starts **no HTTP server, bot listener, or file reader**. The workflow is driven entirely by component calls from a parent.

## Structure

```
component-input-source/
├── workflow.yaml          # sources: [component]
└── resources/
    └── transform.yaml     # transforms text via Python
```

## What It Does

Takes a `text` input and an optional `style` (`uppercase` | `lowercase` | `title`) and returns the transformed text as JSON.

## Using It From a Parent Workflow

```yaml
# In your parent workflow's resource:
run:
  component:
    name: component-input-source
    with:
      text: "hello world"
      style: title
```

Then access the result:
```yaml
"{{ output('myActionId') }}"  # {"original":"hello world","style":"title","result":"Hello World"}
```

## The `component` Source

```yaml
settings:
  input:
    sources: [component]
    component:
      description: "Transforms text to uppercase, lowercase, or title case."
```

- `sources: [component]` - no listener is started; the workflow only runs when called via `run.component`
- `component.description` - surfaced by `kdeps component list` and `kdeps component info`

## Difference from `sources: [api]`

| | `api` | `component` |
|---|---|---|
| HTTP server started | Yes | No |
| Callable via `run.component` | Yes | Yes |
| Callable via HTTP request | Yes | No |
| Listed by `kdeps component list` | Yes | Yes |

## Validate

```bash
kdeps validate examples/component-input-source
```
