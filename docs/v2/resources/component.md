# Component Resource

The `component:` resource calls a reusable resource bundle -- a registry component installed with `kdeps registry install`, or a custom component in your project's `components/` directory. Think of it like calling a function: you pass typed inputs via `with:`, the component's own resources run, and you get its result back.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). See [Components](/concepts/components) for the difference between registry and custom components, and how to build one.

## Basic Usage

```yaml
# resources/scrape.yaml
actionId: scrape
name: Extract Page Content
component:
  name: scraper                 # installed via `kdeps registry install scraper`
  with:
    url: "https://example.com"
    selector: ".article"
```

```yaml
# resources/reply.yaml
actionId: reply
name: Send Bot Reply
component:
  name: botreply
  with:
    platform: telegram
    message: "Hello!"
```

## Configuration Options

| Option | Description |
|---|---|
| `name` | Component name -- either the registry install name, or a directory under `components/` (required) |
| `version` | Pin a specific installed version (registry components only) |
| `with` | Key-value inputs passed to the component. Each entry is injected as `set('<callerActionId>.<key>', value)` before the component's resources run, so the same component can be called multiple times with different inputs in one workflow |

## Output

Whatever the component's own result resource returns -- the shape depends on the component. Check the component's own documentation or `component.yaml` for its output contract.

## See Also

- [Components](/concepts/components) -- registry vs. custom components, how to build and publish one
- [Agent Resource](agent) -- call a full sibling agent instead of a component
- [Resources Overview](overview) -- all resource types
