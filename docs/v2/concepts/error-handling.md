# Error Handling (onError)

KDeps provides built-in error handling for all resource types through the `onError` configuration. This allows you to gracefully handle failures with retries, fallback values, and custom error processing.

## Overview

The `onError` block can be added to any resource to define how errors should be handled:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: myResource
run:
  httpClient:
    url: "https://api.example.com/data"
    method: GET

  onError:
    action: continue
    fallback:
      status: "error"
      message: "Service unavailable"
```

## Configuration Options

### Complete Reference

```yaml
onError:
  # Action to take on error
  action: continue    # "continue", "fail", "retry"

  # Retry configuration (for action: retry)
  maxRetries: 3       # Number of retry attempts
  retryDelay: "1s"    # Delay between retries

  # Fallback value (for action: continue)
  fallback:
    default: "value"

  # Expressions to execute on error (has access to 'error' object)
  expr:
    - set('errorMessage', error.message)
    - set('errorLogged', true)

  # Conditions for when to apply error handling
  when:
    - error.type == 'timeout'
    - contains(error.message, 'connection refused')
```

### Action Types

| Action | Behavior |
|--------|----------|
| `continue` | Continue workflow execution with fallback value or error info |
| `fail` | Stop execution and return the error (default behavior) |
| `retry` | Retry the resource execution with backoff |

## Basic Usage

### Continue with Fallback

Continue execution even if the resource fails, using a fallback value:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: fetchData
run:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: unreliableApi
run:
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
onError:
  action: fail
```

This is the default behavior when no `onError` is configured.

## Advanced Usage

### Error-Specific Handling with `when`

Handle only specific types of errors:

```yaml
onError:
  action: continue
  fallback:
    status: "timeout"
  when:
    - error.type == 'TIMEOUT'
    - contains(error.message, 'deadline exceeded')
```

If the error doesn't match any `when` condition, the error is NOT handled and propagates normally.

### Execute Expressions on Error

Run expressions when an error occurs (useful for logging, metrics, etc.):

```yaml
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: fetchUserData
run:
  httpClient:
    url: "https://api.example.com/users/{{ get('userId') }}"
    method: GET
    timeoutDuration: 5s

  onError:
    action: retry
    maxRetries: 3
    retryDelay: "500ms"
```

</div>

### Graceful Degradation

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: llmEnhancement
run:
  chat:
    model: gpt-4o
    prompt: "Enhance this text: {{ get('text') }}"

  onError:
    action: continue
    fallback: "{{ get('text') }}"  # Return original text on failure
```

</div>

### Circuit Breaker Pattern

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: externalService
run:
  # Check circuit breaker state first
  skipCondition:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: queryPrimary
run:
  sql:
    connection: primary
    query: "SELECT * FROM users WHERE id = ?"
    params:
      - "{{ get('userId') }}"

  onError:
    action: continue
    fallback: null

---
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: queryReplica
  requires:
    - queryPrimary
run:
  # Only query replica if primary failed
  skipCondition:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: primaryLLM
run:
  chat:
    backend: openai
    model: gpt-4o
    prompt: "{{ get('q') }}"

  onError:
    action: continue
    fallback: null

---
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: fallbackLLM
  requires:
    - primaryLLM
run:
  skipCondition:
    - get('primaryLLM') != null
    - safe(get('primaryLLM'), '_error') == nil

  chat:
    backend: ollama
    model: llama3.2:1b
    prompt: "{{ get('q') }}"

---
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: response
  requires:
    - primaryLLM
    - fallbackLLM
run:
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
6. **Combine with skipCondition** - Create fallback resource chains

## See Also

- [Validation](validation) - Input validation
- [Expression Helpers](expression-helpers) - Helper functions
- [Resources Overview](../resources/overview) - Resource types
