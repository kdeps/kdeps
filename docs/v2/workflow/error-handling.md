# Error handling (onError)

`onError` sets what happens when a resource fails: retry, use a fallback value, or log and continue. Without it, any error stops the workflow.

```yaml
# resources/example.yaml
onError:
  action: continue    # "continue" (use fallback), "retry", or "fail" (default)

  maxRetries: 3       # for action: retry -- total attempts after the first
  retryDelay: "1s"    # wait between retries

  fallback:           # for action: continue -- what get('resourceId') returns on failure
    status: "error"
    message: "Service unavailable"

  expr:               # expressions that run when an error is caught
    - set('errorMessage', error.message)
    - set('errorLogged', true)

  when:               # only apply onError if one of these is true
    - error.type == 'TIMEOUT'          # otherwise the error propagates normally
    - error.message contains 'connection refused'
```

## Read next

- [Error handling (onError) reference](/reference/workflow/error-handling) - complete reference, basic usage, advanced usage, use cases, error object reference, best practices
- [Validation and control flow](/workflow/validation) - Input validation
- [Expression helpers](/workflow/expression-helpers) - Helper functions
- [Resources overview](/workflow/resources) - Resource types
