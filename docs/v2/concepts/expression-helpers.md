# Expression Helper Functions

Expression helper functions give you safe nested-property access, null coalescing, JSON serialization, URL encoding, and conditionals -- the utilities you reach for after `get()` and `set()` are not enough.

## Overview

| Function | Purpose |
|----------|---------|
| `json(data)` | Convert data to JSON string |
| `toJSON(data)` | Alias for `json()` |
| `safe(obj, path)` | Safely access nested properties |
| `debug(obj)` | Format data for debugging |
| `default(value, fallback)` | Null coalescing |
| `urlencode(str)` | URL-encode a string |
| `ternary(cond, trueVal, falseVal)` | Conditional expression |
| `input(key)` | Access request input |
| `output(resourceId)` | Access resource output |

## json()

Converts any data structure to a JSON string. Useful for logging, debugging, or preparing data for external APIs.

### Syntax

```yaml
json(data)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `data` | any | Data to convert to JSON |

### Examples

**Basic usage:**
```yaml
actionId: formatData
after:
  - set('user', {"name": "Alice", "age": 30})
  - set('userJson', json(get('user')))
  # Result: '{"name":"Alice","age":30}'
```

**Preparing API payload:**
<div v-pre>

```yaml
after:
  - set('payload', json({
      "query": get('q'),
      "timestamp": info('request.id'),
      "metadata": get('sessionData')
    }))
httpClient:
  url: "https://api.example.com/data"
  method: POST
  body: "{{ get('payload') }}"
```

</div>

**Logging complex objects:**
```yaml
after:
  - set('debugLog', json({
      "request": get('q'),
      "response": get('llmResource'),
      "duration": info('request.id')
    }))
```

## safe()

Safely accesses nested properties without throwing errors. Returns `nil` if any part of the path is missing.

### Syntax

```yaml
safe(object, path)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `object` | object | The object to traverse |
| `path` | string | Dot-separated path to the property |

### Examples

**Basic nested access:**
```yaml
after:
  # If user is {"profile": {"address": {"city": "NYC"}}}
  - set('city', safe(get('user'), 'profile.address.city'))
  # Result: "NYC"

  # If path doesn't exist, returns nil instead of error
  - set('country', safe(get('user'), 'profile.address.country'))
  # Result: nil (no error thrown)
```

**With fallback using default():**
```yaml
after:
  - set('city', default(safe(get('user'), 'profile.address.city'), 'Unknown'))
```

**Accessing API response data:**
```yaml
after:
  - set('errorMessage', safe(get('apiResponse'), 'error.details.message'))
  - set('items', safe(get('apiResponse'), 'data.items'))
```

**Handling optional configuration:**
```yaml
after:
  - set('timeout', default(safe(get('config'), 'http.timeout'), 30))
  - set('retries', default(safe(get('config'), 'http.retries'), 3))
```

## debug()

Returns a formatted, indented JSON string representation of any data. Useful for development and troubleshooting.

### Syntax

```yaml
debug(data)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `data` | any | Data to format for debugging |

### Examples

**Debug LLM response:**
```yaml
after:
  - set('debugInfo', debug(get('llmResource')))
  # Result (formatted with indentation):
  # {
  #   "response": "...",
  #   "model": "llama3.2:1b",
  #   "tokens": 150
  # }
```

**Debug request context:**
```yaml
after:
  - set('requestDebug', debug({
      "query": get('q'),
      "headers": get('Authorization'),
      "session": get('userId', 'session')
    }))
```

**Include in API response for debugging:**
<div v-pre>

```yaml
apiResponse:
  success: true
  response:
    data: get('result')
    _debug: "{{ get('debugInfo') }}"  # Remove in production
```

</div>

## default()

Null coalescing operator. Returns the fallback value if the primary value is `nil` or an empty string.

### Syntax

```yaml
default(value, fallback)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `value` | any | Primary value to check |
| `fallback` | any | Value to return if primary is nil/empty |

### Examples

**Basic fallback:**
```yaml
after:
  - set('name', default(get('userName'), 'Guest'))
  - set('limit', default(get('pageSize'), 10))
```

**Chained defaults:**
```yaml
after:
  # Try multiple sources in order
  - set('apiKey', default(
      get('API_KEY', 'env'),
      default(
        get('apiKey', 'session'),
        'default-key'
      )
    ))
```

**With safe() for nested access:**
```yaml
after:
  - set('email', default(safe(get('user'), 'contact.email'), 'no-email@example.com'))
```

**Configuration defaults:**
```yaml
after:
  - set('config', {
      "timeout": default(get('timeout'), 30),
      "retries": default(get('retries'), 3),
      "model": default(get('model'), 'llama3.2:1b')
    })
```

## toJSON()

Alias for `json()`. Converts any data structure to a JSON string.

### Syntax

```yaml
toJSON(data)
```

## urlencode()

URL-encodes a string. Useful for building query parameters or form-encoded values.

### Syntax

```yaml
urlencode(str)
```

### Examples

<div v-pre>

```yaml
component:
  name: browser
  with:
    url: "https://www.example.com/search?q={{ urlencode(get('query')) }}"
    action: getText
```

</div>

```yaml
after:
  - set('encoded', urlencode(get('searchTerm')))
  # "hello world" -> "hello+world"
```

## ternary()

Returns `trueVal` if `cond` is `true`, otherwise returns `falseVal`. Equivalent to a conditional expression.

### Syntax

```yaml
ternary(condition, trueVal, falseVal)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `condition` | bool | Boolean condition to test |
| `trueVal` | any | Value returned when condition is true |
| `falseVal` | any | Value returned when condition is false |

### Examples

<div v-pre>

```yaml
component:
  name: browser
  with:
    url: "https://example.com/jobs?remote={{ get('remote_only') == 'true' | ternary('2', '') }}"
    action: getText
```

</div>

```yaml
after:
  - set('label', ternary(get('isAdmin'), 'Admin', 'User'))
```

## input()

Alternative way to access request input parameters. Equivalent to `get(key)` for request data.

### Syntax

```yaml
input(key)
# or
input.property
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `key` | string | The input parameter name |

### Examples

**Function style:**
```yaml
after:
  - set('query', input('q'))
  - set('limit', input('limit'))
```

**Property access style:**
```yaml
after:
  - set('items', input.items)
  - set('userId', input.userId)
```

**Equivalent to get():**
```yaml
after:
  # These are equivalent:
  - set('query1', get('q'))
  - set('query2', input('q'))
```

## output()

Alternative way to access resource outputs. Equivalent to `get(resourceId)`.

### Syntax

```yaml
output(resourceId)
```

### Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `resourceId` | string | The actionId of the resource |

### Examples

**Basic usage:**
```yaml
after:
  - set('llmResult', output('llmResource'))
  - set('sqlData', output('databaseQuery'))
```

**Equivalent to get():**
```yaml
after:
  # These are equivalent:
  - set('result1', get('llmResource'))
  - set('result2', output('llmResource'))
```

## See Also

- [Unified API](/concepts/unified-api) - Core get() and set() functions
- [Expressions](/concepts/expressions) - Expression syntax and operators
- [Expression Functions Reference](/reference/expression-functions-reference) - Complete function reference with array, string, and type operations
- [Items Iteration](items) - Processing multiple items
