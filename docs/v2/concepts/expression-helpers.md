# Expression Helper Functions

KDeps provides several built-in helper functions beyond `get()` and `set()` for data manipulation and debugging in expressions.

## Overview

| Function | Purpose |
|----------|---------|
| `json(data)` | Convert data to JSON string |
| `safe(obj, path)` | Safely access nested properties |
| `debug(obj)` | Format data for debugging |
| `default(value, fallback)` | Null coalescing |
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: formatData
run:
  expr:
    - set('user', {"name": "Alice", "age": 30})
    - set('userJson', json(get('user')))
    # Result: '{"name":"Alice","age":30}'
```

**Preparing API payload:**
<div v-pre>

```yaml
run:
  expr:
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
run:
  expr:
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
run:
  expr:
    # If user is {"profile": {"address": {"city": "NYC"}}}
    - set('city', safe(get('user'), 'profile.address.city'))
    # Result: "NYC"

    # If path doesn't exist, returns nil instead of error
    - set('country', safe(get('user'), 'profile.address.country'))
    # Result: nil (no error thrown)
```

**With fallback using default():**
```yaml
run:
  expr:
    - set('city', default(safe(get('user'), 'profile.address.city'), 'Unknown'))
```

**Accessing API response data:**
```yaml
run:
  expr:
    - set('errorMessage', safe(get('apiResponse'), 'error.details.message'))
    - set('items', safe(get('apiResponse'), 'data.items'))
```

**Handling optional configuration:**
```yaml
run:
  expr:
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
run:
  expr:
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
run:
  expr:
    - set('requestDebug', debug({
        "query": get('q'),
        "headers": get('Authorization'),
        "session": get('userId', 'session')
      }))
```

**Include in API response for debugging:**
<div v-pre>

```yaml
run:
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
run:
  expr:
    - set('name', default(get('userName'), 'Guest'))
    - set('limit', default(get('pageSize'), 10))
```

**Chained defaults:**
```yaml
run:
  expr:
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
run:
  expr:
    - set('email', default(safe(get('user'), 'contact.email'), 'no-email@example.com'))
```

**Configuration defaults:**
```yaml
run:
  expr:
    - set('config', {
        "timeout": default(get('timeout'), 30),
        "retries": default(get('retries'), 3),
        "model": default(get('model'), 'llama3.2:1b')
      })
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
run:
  expr:
    - set('query', input('q'))
    - set('limit', input('limit'))
```

**Property access style:**
```yaml
run:
  expr:
    - set('items', input.items)
    - set('userId', input.userId)
```

**Equivalent to get():**
```yaml
run:
  expr:
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
run:
  expr:
    - set('llmResult', output('llmResource'))
    - set('sqlData', output('databaseQuery'))
```

**Equivalent to get():**
```yaml
run:
  expr:
    # These are equivalent:
    - set('result1', get('llmResource'))
    - set('result2', output('llmResource'))
```

## Combining Helper Functions

### Complex Data Transformation

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: processData
  requires:
    - fetchData
run:
  expr:
    # Safely extract nested data with defaults
    - set('items', default(safe(get('fetchData'), 'response.data.items'), []))

    # Format for logging
    - set('log', json({
        "itemCount": len(get('items')),
        "query": get('q'),
        "timestamp": info('request.id')
      }))

    # Debug complex structures during development
    - set('debugOutput', debug({
        "rawResponse": get('fetchData'),
        "processedItems": get('items')
      }))
```

### Error Handling Pattern

```yaml
run:
  expr:
    # Check for error in response
    - set('error', safe(get('apiResponse'), 'error.message'))
    - set('hasError', get('error') != nil)

    # Get data or error message
    - set('result', default(
        safe(get('apiResponse'), 'data'),
        {"error": default(get('error'), 'Unknown error')}
      ))
```

### Configuration Loading

```yaml
run:
  expr:
    # Load config with multiple fallback sources
    - set('dbHost', default(
        get('DB_HOST', 'env'),
        default(safe(get('config'), 'database.host'), 'localhost')
      ))

    - set('dbPort', default(
        get('DB_PORT', 'env'),
        default(safe(get('config'), 'database.port'), 5432)
      ))
```

## Best Practices

1. **Use safe() for external data** - Always use `safe()` when accessing data from external APIs or user input where structure may vary.

2. **Combine safe() with default()** - Provide sensible defaults for optional values:
   ```yaml
   set('value', default(safe(get('data'), 'optional.path'), 'default'))
   ```

3. **Use debug() during development** - Remove or disable debug output in production.

4. **Prefer json() for API payloads** - Ensures proper formatting of complex objects.

5. **Use default() for configuration** - Make your workflows more robust by providing fallbacks.

## See Also

- [Unified API](unified-api) - Core get() and set() functions
- [Expressions](expressions) - Expression syntax and operators
- [Expression Functions Reference](expression-functions-reference) - Complete function reference
- [Advanced Expressions](advanced-expressions) - Advanced expression features
- [Items Iteration](items) - Processing multiple items
