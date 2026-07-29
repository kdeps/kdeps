# Chapter 23: Error Handling with onError

By default, any resource failure stops the workflow immediately and returns an error to the caller. `onError:` changes this. It gives you control over what happens when a resource fails: retry, substitute a fallback value, or log and continue.

## The Default: Fail Loud

Without `onError:`, a failed resource propagates the error immediately:

```
[httpClient] GET https://api.example.com/data → connection refused
Workflow stopped. Error returned to caller.
```

Downstream resources never run. The caller gets a 500 or the error from the failed resource. This is the right behavior for most cases — fail early, fail clearly.

## onError: Syntax

```yaml
# resources/example.yaml
actionId: example

httpClient:
  url: "https://api.example.com/data"

onError:
  action: continue      # "continue", "retry", or "fail" (default)
  
  maxRetries: 3         # for action: retry — attempts after the first
  retryDelay: "1s"      # wait between retries (exponential backoff from this base)
  
  fallback:             # for action: continue — what get('example') returns on failure
    data: []
    error: true
    message: "service unavailable"
  
  expr:                 # expressions that run when an error is caught
    - set('errorMessage', error.message)
    - set('errorType', error.type)
  
  when:                 # only apply onError if one of these conditions is true
    - error.type == 'TIMEOUT'
    - error.message contains 'connection refused'
    # if the error doesn't match, it propagates normally
```

## The Three Actions

### action: fail (default)

Stops the workflow and returns the error. This is the default — you do not need to write `onError` to get this behavior.

```yaml
onError:
  action: fail    # same as having no onError block
```

Explicit `fail` is useful combined with `when:` to fail on specific errors while handling others:

```yaml
onError:
  action: fail
  when:
    - error.type == 'AUTH_ERROR'    # always fail on auth errors; handle others elsewhere
```

### action: continue

The resource is treated as if it succeeded with the `fallback:` value. Downstream resources run normally. `get('resourceId')` returns the fallback instead of the real output.

```yaml
# resources/fetch-enrichment.yaml
actionId: fetchEnrichment
httpClient:
  url: "https://enrichment-service.example.com/enrich/&#123;&#123; get('companyName') &#125;&#125;"
  timeout: 10s

onError:
  action: continue
  fallback:
    score: null
    industry: "unknown"
    enriched: false
```

If the enrichment service is down, `get('fetchEnrichment')` returns `{"score": null, "industry": "unknown", "enriched": false}`. The rest of the workflow continues. The downstream response resource might include `enriched: false` to signal to the caller that enrichment was not available.

**continue without fallback:**

If you omit `fallback:`, the resource returns an error info object on failure:

```json
{"error": true, "message": "connection refused", "type": "NETWORK_ERROR"}
```

Downstream resources can check for this:

```yaml
validations:
  skip:
    - get('fetchEnrichment').error == true    # skip if enrichment failed
```

### action: retry

Retries the resource action up to `maxRetries` times before failing:

```yaml
# resources/fetch-api.yaml
actionId: fetchApi
httpClient:
  url: "https://unreliable-api.example.com/data"
  timeout: 30s

onError:
  action: retry
  maxRetries: 3
  retryDelay: "2s"
  when:
    - error.type == 'TIMEOUT'
    - error.type == 'NETWORK_ERROR'
    - error.message contains '503'
```

Retry sequence:
- Attempt 1: fails (TIMEOUT)
- Wait 2 seconds
- Attempt 2: fails (TIMEOUT)
- Wait 4 seconds (exponential)
- Attempt 3: fails (TIMEOUT)
- Wait 8 seconds
- Attempt 4 (maxRetries=3 means 3 retries after the first attempt = 4 total): fails → workflow stops with error

If the resource succeeds on any retry, execution continues normally.

**Retry with final fallback:**

```yaml
onError:
  action: retry
  maxRetries: 2
  retryDelay: "1s"
  fallback:           # used only after all retries are exhausted
    data: []
    from_cache: true
```

After all retries fail, the fallback kicks in instead of failing the workflow.

## The error Object

Inside `expr:` blocks and `when:` conditions, the `error` object is available:

| Field | Description |
|---|---|
| `error.message` | Human-readable error message |
| `error.type` | Machine-readable error type (e.g., `TIMEOUT`, `NETWORK_ERROR`, `AUTH_ERROR`, `VALIDATION_ERROR`) |
| `error.code` | HTTP status code (for `httpClient:` failures) |
| `error.resource` | `actionId` of the failed resource |

```yaml
onError:
  expr:
    - set('last_error', json({
        "resource": error.resource,
        "type": error.type,
        "message": error.message,
        "timestamp": info('timestamp')
      }))
  
  when:
    - error.type != 'AUTH_ERROR'    # don't retry on auth failures
  
  action: retry
  maxRetries: 3
```

`expr:` runs before `action` takes effect. Use it to log, record metrics, or set downstream-visible values about the failure.

## Conditional Error Handling with when:

`when:` limits `onError:` to specific error conditions. If none of the `when:` expressions match, the error propagates normally — as if there were no `onError:` block:

```yaml
onError:
  action: continue
  fallback:
    result: null
  when:
    - error.type == 'TIMEOUT'
    - error.code == 503
    # On any other error (404, 401, network error), fail normally
```

This gives you precise control: handle transient failures gracefully, let permanent failures propagate.

## Practical Patterns

### Graceful Degradation

```yaml
# resources/get-recommendations.yaml
actionId: getRecommendations
httpClient:
  url: "https://ml-service.internal/recommend/&#123;&#123; get('userId') &#125;&#125;"
  timeout: 5s

onError:
  action: continue
  fallback:
    items: []
    fallback_used: true
  when:
    - error.type in ['TIMEOUT', 'NETWORK_ERROR']
    - error.code in [503, 502, 504]
```

```yaml
# resources/respond.yaml
apiResponse:
  success: true
  response:
    recommendations: get('getRecommendations').items
    personalized: get('getRecommendations').fallback_used != true
```

The API always returns a response. If the ML service is unavailable, it returns an empty recommendations list with `personalized: false`.

### Retry with Logging

```yaml
# resources/call-external-api.yaml
actionId: callExternalApi
httpClient:
  url: "https://partner-api.example.com/data"
  timeout: 30s

onError:
  action: retry
  maxRetries: 3
  retryDelay: "2s"
  expr:
    - set('retry_count', loop.count() or 1)
    - set('error_log', get('error_log', 'session') or [])
    - set('error_log', get('error_log') + [{
        "attempt": get('retry_count'),
        "error": error.message,
        "timestamp": info('timestamp')
      }], 'session')
  when:
    - error.type != 'AUTH_ERROR'
```

Each retry attempt appends to an error log in the session store. After the workflow completes (or fails), the log records every failed attempt with timestamps.

### Circuit-Breaker Pattern

For high-traffic workflows, you can simulate a circuit breaker using session state:

```yaml
# resources/check-circuit.yaml
actionId: checkCircuit
before:
  - set('failures', int(get('api_failures', 'session')) or 0)
validations:
  check:
    - get('failures') < 5    # circuit open if 5+ recent failures
  error:
    code: 503
    message: "Service temporarily unavailable (circuit open)"
exec:
  command: "echo ok"

# resources/call-api.yaml
actionId: callApi
requires: [checkCircuit]
httpClient:
  url: "https://fragile-service.example.com/data"

onError:
  action: continue
  fallback:
    data: null
  expr:
    - set('api_failures', int(get('api_failures', 'session') or '0') + 1, 'session')
  when:
    - error.type in ['TIMEOUT', 'NETWORK_ERROR']
```

Failures increment a session counter. When the counter reaches 5, the `checkCircuit` validation rejects requests immediately without trying the API. A real circuit breaker would also need a reset mechanism (TTL on the counter), but this illustrates the pattern.

## Error Handling vs. Validation

These two features look similar but serve different purposes:

| `validations.check:` | `onError:` |
|---|---|
| Checks conditions *before* the resource action runs | Catches failures *during or after* execution |
| For input validation — "is this request valid?" | For resilience — "what do we do when the service is down?" |
| Fires when data is wrong | Fires when execution fails |
| Always defined behavior | Exception handling |

Use `validations:` to enforce correctness. Use `onError:` to handle reality.

X> ## Exercise
X>
X> Build a workflow that calls an external weather API and degrades gracefully when it is unavailable.
X>
X> The endpoint accepts `POST /api/v1/weather` with a `city` field and normally returns a full weather report. When the API is down or slow, it should return a cached result or a friendly fallback — never a raw error.
X>
X> Requirements:
X>
X> 1. Create a `fetchWeather` resource with `httpClient:` pointing at a weather API (or mock one with `exec: "sleep 10 && exit 1"` to simulate timeouts). Add `onError: { action: retry, maxRetries: 2, retryDelay: "500ms" }`.
X> 2. After the retries fail, fall through to a `cachedWeather` resource that reads a session key `lastKnownWeather`. Use `validations.skip` to skip this resource if `fetchWeather` succeeded.
X> 3. If neither source has data, return a graceful response: `{ "success": false, "error": "weather data temporarily unavailable", "city": "..." }` with HTTP 503.
X> 4. Add `onError.expr` to the `fetchWeather` resource that logs the error type and timestamp to a session key `errorLog`.
X>
X> Test all three paths: success, retry-then-cache-hit, and retry-then-no-cache.
X>
X> **Stretch goal:** Implement the circuit breaker pattern from the chapter: after 5 consecutive failures, open the circuit (set a session flag) so subsequent requests skip the API call entirely for 60 seconds, returning the cached value immediately.
