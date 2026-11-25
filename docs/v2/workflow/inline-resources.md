# Inline resources

Inline resources are full resource actions (chat, component, file) placed directly inside a resource's `before:` or `after:` block. They run as part of that resource instead of requiring separate files.

```yaml
# resources/example.yaml
actionId: example
name: Example Resource
# Inline resources to run BEFORE the main resource
before:
  - component:                     # HTTP steps go through the http_request component
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/config"
  - component:                     # shell steps go through the exec component
      name: exec
      with:
        command: "echo 'Preparing environment...'"

# Main resource (chat, component, or file)
chat:
  context:
    role: user
    prompt: "Process this data"

# Inline resources to run AFTER the main resource
after:
  - component:
      name: sql
      with:
        connectionName: main  # DSN defined in ~/.kdeps/config.yaml sql_connections.main.connection
        query: "INSERT INTO logs VALUES (?)"
  - component:
      name: python
      with:
        script: "print('Post-processing complete')"
```

## Read next

- [Inline resources reference](/reference/workflow/inline-resources) - basic syntax, supported resource types, execution order, common use cases, multiple inline resources, resources without main type, error handling, accessing context
- [Expression blocks](/reference/expr-blocks) - Using `before` and `after`
- [Error handling](/workflow/error-handling) - Handling errors in resources
- [Items](/workflow/items) - Iterating over collections
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples/inline-resources) - Complete example with inline resources
