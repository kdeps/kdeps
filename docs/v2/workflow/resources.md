# Resources overview

A resource is a single step in a workflow. It has an ID, optional dependencies, optional validation, and exactly one action. kdeps builds a dependency graph from all resources and runs them in order.

<div v-pre>

```yaml
# resources/my-resource.yaml
actionId: myResource        # unique ID, used by requires: and get()
name: My Resource
flow:
  requires: [otherResource]  # runs after otherResource
validations:
  check:
    - get('q') != ''         # fail with a 400 if q is missing
chat:                        # the one action: chat, component, apiResponse, ...
  context:
    prompt: "{{ get('q') }}"
```

</div>

## Read next

- [Resources overview reference](/reference/workflow/resources) - the shape of a resource, the one action, actionId and requires, resource types, AI & language, web, data & system, messaging
- [Validation and control flow](/workflow/validation) - the `validations:` block
- [Expressions](/workflow/expressions) - `before:`/`after:`, `get()`, `set()`
- [Items iteration](/workflow/items) and [Loop](/workflow/loop)
- [Error handling (onError)](/workflow/error-handling) - retry and fallback
- [Agencies](/agencies/) - multi-agent orchestration
- [Components](/agencies/components) - installable capability extensions
