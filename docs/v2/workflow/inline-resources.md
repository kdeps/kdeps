# Inline resources

Inline resources are full resource actions (chat, component, file) placed directly inside a resource's `before:` or `after:` block. They run as part of that resource instead of requiring separate files.

<div v-pre>

```yaml
# resources/example.yaml
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/user/{{get('user_id')}}"
        timeout: 5s

chat:
  context:
    prompt: "Analyze user: {{get('_output')}}"
```

</div>

## Read next

- [Inline resources reference](/reference/workflow/inline-resources) - every key, option and detail
- [Error handling](/workflow/error-handling) - Handling errors in resources
- [Items](/workflow/items) - Iterating over collections
