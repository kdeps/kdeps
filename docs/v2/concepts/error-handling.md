# Error Handling (onError)

`onError` defines what happens when a resource fails. Without it, any error stops the workflow immediately. With it, you can retry, substitute a fallback value, or log the error and continue.

## Complete reference

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

| action | what happens |
|--------|----------|
| `continue` | downstream resources run; `get('resourceId')` returns the fallback |
| `fail` | workflow stops and returns the error (default when no onError block) |
| `retry` | resource is retried up to `maxRetries` times; fails after that |

## Basic Usage

### Continue with Fallback

Continue execution even if the resource fails, using a fallback value:

```yaml
# resources/fetch-data.yaml
actionId: fetchData
httpClient:
  url: "https://api.example.com/data"
  method: GET

onError:
  action: continue
  fallback:
    data: []
    fromCache: false
    error: true
```

When the HTTP request fails, the resource returns the fallback value instead of stopping the workflow.

### Continue without Fallback

If no fallback is provided, the resource returns an error info object:

```yaml
# resources/example.yaml
onError:
  action: continue
```

The output will be:
```json
{
  "_error": {
    "message": "connection refused",
    "handled": true
  }
}
```

### Retry with Backoff

Automatically retry failed operations:

```yaml
# resources/unreliable-api.yaml
actionId: unreliableApi
httpClient:
  url: "https://flaky-api.example.com/data"
  method: GET

onError:
  action: retry
  maxRetries: 3
  retryDelay: "1s"
```

This will:
1. Execute the resource
2. On failure, wait 1 second
3. Retry up to 3 times total
4. If all retries fail, return an error

### Explicit Fail

Explicitly mark that errors should stop execution (useful for documentation):

```yaml
# resources/example.yaml
onError:
  action: fail
```

This is the default behavior when no `onError` is configured.

## Advanced Usage

### Error-Specific Handling with `when`

Handle only specific types of errors:

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

If the error doesn't match any `when` condition, the error is NOT handled and propagates normally.

### Execute Expressions on Error (`expr`)

Run expressions when an error occurs (useful for logging, metrics, etc.):

```yaml
# resources/example.yaml
onError:
  action: continue
  expr:
    - set('lastError', error.message, 'session')
    - set('errorCount', get('errorCount', 'session') + 1, 'session')
    - set('errorTimestamp', info('timestamp'))
  fallback:
    error: true
    retryLater: true
```

The expressions have access to the `error` object:
- `error.message` - The error message string
- `error.type` - Error type/code (e.g., "TIMEOUT", "VALIDATION_ERROR")
- `error.code` - Error code (same as type)
- `error.statusCode` - HTTP status code (if applicable)
- `error.details` - Additional error details (if available)

### Dynamic Fallback Values

Fallback values can include expressions:

<div v-pre>

```yaml
# resources/example.yaml
onError:
  action: continue
  fallback:
    data: "{{ get('cachedData', 'session') }}"
    timestamp: "{{ info('timestamp') }}"
    error: true
```

</div>

## Use Cases

### Resilient API Calls

<div v-pre>

```yaml
# resources/fetch-user-data.yaml
actionId: fetchUserData
httpClient:
  url: "https://api.example.com/users/{{ get('userId') }}"
  method: GET
  timeout: 5s

onError:
  action: retry
  maxRetries: 3
  retryDelay: "500ms"
```

</div>

### Graceful Degradation

<div v-pre>

```yaml
# resources/llm-enhancement.yaml
actionId: llmEnhancement
chat:
  prompt: "Enhance this text: {{ get('text') }}"

onError:
  action: continue
  fallback: "{{ get('text') }}"  # Return original text on failure
```

</div>

### Circuit Breaker Pattern

```yaml
# resources/external-service.yaml
actionId: externalService
# Check circuit breaker state first
validations:
  skip:
  - get('circuitOpen', 'session') == true

httpClient:
  url: "https://api.example.com/data"
  method: GET

onError:
  action: continue
  expr:
    # Increment failure count
    - set('failCount', default(get('failCount', 'session'), 0) + 1, 'session')
    # Open circuit after 5 failures
    - set('circuitOpen', get('failCount', 'session') >= 5, 'session')
  fallback:
    error: true
    circuitBreaker: "open"
```

### Database Fallback

<div v-pre>

```yaml
# resources/query-primary.yaml
actionId: queryPrimary
sql:
  connection: primary
  query: "SELECT * FROM users WHERE id = ?"
  params:
    - "{{ get('userId') }}"

onError:
  action: continue
  fallback: null

---
actionId: queryReplica
requires:
  - queryPrimary
# Only query replica if primary failed
validations:
  skip:
  - get('queryPrimary') != null
  - safe(get('queryPrimary'), '_error') == nil

sql:
  connection: replica
  query: "SELECT * FROM users WHERE id = ?"
  params:
    - "{{ get('userId') }}"
```

</div>

### LLM with Model Fallback

<div v-pre>

```yaml
# resources/primary-l-l-m.yaml
actionId: primaryLLM
chat:
  prompt: "{{ get('q') }}"

onError:
  action: continue
  fallback: null

---
actionId: fallbackLLM
requires:
  - primaryLLM
validations:
  skip:
  - get('primaryLLM') != null
  - safe(get('primaryLLM'), '_error') == nil

chat:
  prompt: "{{ get('q') }}"

---
actionId: response
requires:
  - primaryLLM
  - fallbackLLM
apiResponse:
  success: true
  response:
    answer: "{{ default(get('primaryLLM'), get('fallbackLLM')) }}"
```

</div>

## Error Object Reference

In `onError.expr` and `onError.when` expressions, the `error` object is available:

| Property | Type | Description |
|----------|------|-------------|
| `error.message` | string | Human-readable error message |
| `error.type` | string | Error type code |
| `error.code` | string | Same as type |
| `error.statusCode` | number | HTTP status code (if applicable) |
| `error.details` | object | Additional error context |

### Common Error Types

| Type | Description |
|------|-------------|
| `execution_error` | General execution failure |
| `TIMEOUT` | Request timed out |
| `VALIDATION_ERROR` | Input validation failed |
| `NOT_FOUND` | Resource not found |
| `UNAUTHORIZED` | Authentication required |
| `RESOURCE_FAILED` | Resource execution failed |

## Best Practices

1. **Use retries for transient failures** - Network issues, rate limits, temporary unavailability
2. **Use continue for non-critical operations** - Enhancements, optional data, caching
3. **Use when conditions** - Handle specific errors differently
4. **Log errors with expr** - Store error info for debugging/monitoring
5. **Provide meaningful fallbacks** - Return useful data even on failure
6. **Combine with validations.skip** - Create fallback resource chains

## See Also

- [Validation](/concepts/validation) - Input validation
- [Expression Helpers](/concepts/expression-helpers) - Helper functions
- [Resources Overview](../resources/overview) - Resource types
