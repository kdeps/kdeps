# Resources overview

A resource is a single step in a workflow. It has an ID, the steps it needs, and one action.

<div v-pre>

```yaml
# resources/my-resource.yaml
actionId: myResource
flow:
  requires: [otherResource]   # runs after otherResource
chat:                         # the one action
  context:
    prompt: "{{ get('q') }}"
```

</div>

## Read next

- [Resources overview reference](/reference/workflow/resources) - every key, option and detail
- [Validation and control flow](/workflow/validation) - the `validations:` block
- [Expressions](/workflow/expressions) - `before:`/`after:`, `get()`, `set()`
