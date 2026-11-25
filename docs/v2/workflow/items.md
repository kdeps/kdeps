# Items iteration

`items:` runs a resource one time for each list entry. Each iteration is a full resource execution with its own output.

<div v-pre>

```yaml
# resources/process-items.yaml
flow:
  items: ["Item 1", "Item 2"]
chat:
  context:
    prompt: "Process: {{ get('current') }}"
```

</div>

## Read next

- [Items iteration reference](/reference/workflow/items) - every key, option and detail
- [Resources overview](/workflow/resources) - Resource configuration
- [Expressions](/workflow/expressions) - Expression syntax
