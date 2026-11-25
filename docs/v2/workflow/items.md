# Items iteration

`items:` runs a resource one time for each list entry. Each iteration is a full resource execution with its own output.

<div v-pre>

```yaml
# resources/process-items.yaml

actionId: processItems
flow:
  items:
    - "Item 1"
    - "Item 2"
    - "Item 3"

chat:
  context:
    prompt: "Process: {{ get('current') }}"
```

</div>

## Read next

- [Items iteration reference](/reference/workflow/items) - basic usage, item context, the `item` object, accessing all item values, examples, use cases
- [Items reference](/reference/items) - dynamic items, performance, best practices
- [Resources overview](/workflow/resources) - Resource configuration
- [Expressions](/workflow/expressions) - Expression syntax
- [Expression functions reference](/reference/expression-functions) - Complete function reference
- [Python resource](/workflow/resources/python) - For complex batch processing
