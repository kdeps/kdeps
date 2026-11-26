# Error handling (onError)

`onError` sets what happens when a resource fails: retry, use a fallback value, or log and continue. Without it, any error stops the workflow.

```yaml
# resources/example.yaml
onError:
  action: continue
  fallback:
    status: "timeout"
  when:
    - error.type == 'TIMEOUT'
    - error.message contains 'deadline exceeded'
```

## Read next

- [Error handling (onError) reference](/reference/workflow/error-handling) - every key, option and detail
- [Validation and control flow](/workflow/validation) - Input validation
- [Expression helpers](/workflow/expression-helpers) - Helper functions
