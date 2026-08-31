# Delegation resources

Two resources that call another unit of work and return its output. Each has
its own reference page.

*Applies to both workflow mode and agent mode.*

| Resource | Calls | Inputs | Reference |
| :--- | :--- | :--- | :--- |
| `agent:` | A full sibling agent's workflow (same agency) | `params:` | [Agent](/resources/delegation/agent) |
| `component:` | A reusable resource bundle (registry or local) | `with:` | [Component](/resources/delegation/component) |

## See also

- [AI agencies](/concepts/agency) - multi-agent orchestration
- [Components](/concepts/components) - registry vs. custom components
- [Two-agent agency tutorial](/examples/agency)
- [Reusable component tutorial](/examples/custom-component)
- [Resources overview](/resources/overview) - all resource types
