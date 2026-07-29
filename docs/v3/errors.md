# Errors (`onError`)

Default: any resource failure **stops the workflow** and returns an error.

`onError:` changes that — continue with a fallback, retry, or fail only for some cases.

```yaml
actionId: fetchApi
httpClient:
  url: "https://unreliable.example.com/data"
  timeout: 30s

onError:
  action: retry              # continue | retry | fail (default)
  maxRetries: 3
  retryDelay: "1s"           # base; exponential backoff
  when:
    - error.type == 'TIMEOUT'
    - error.type == 'NETWORK_ERROR'
  fallback:                  # used with action: continue
    data: []
    error: true
  expr:
    - set('errorMessage', error.message)
```

## Actions

| `action` | Behavior |
|----------|----------|
| `fail` | Stop workflow (default) |
| `continue` | Use `fallback` (or an error object) as `get('actionId')`; keep going |
| `retry` | Retry up to `maxRetries`, then fail (or fall through if combined carefully) |

### continue without fallback

`get('resourceId')` becomes roughly:

```json
{"error": true, "message": "connection refused", "type": "NETWORK_ERROR"}
```

Downstream can `skip:` when `get('fetchApi').error == true`.

### when:

Only apply this `onError` if a condition matches; otherwise the error propagates normally.

```yaml
onError:
  action: fail
  when:
    - error.type == 'AUTH_ERROR'
```

## Validation vs onError

- **`validations.check`** — fail *before* the action (bad input)  
- **`onError`** — handle *action* failure (network, model, SQL, …)

[Validation](/resources) · [Iteration](/iteration) · [Debug](/debug).
