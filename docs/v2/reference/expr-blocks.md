# Inline Resource Blocks (before / after)

`before:` and `after:` are expression lists that run around a resource's main action -- `before:` prepares data before the action runs, `after:` processes output after it completes. Think of them like setup/teardown blocks around a function call.

- `before:` runs before the main action (chat, httpClient, sql, file, etc.)
- `after:` runs after the main action

Both accept bare scalar expressions. Each item executes in sequence and can call `set()` to store values, perform calculations, or read from memory and session.

## Basic Usage

<div v-pre>

```yaml

actionId: preProcessor
name: Pre-Processor
after:
  - set('normalized_input', lower(get('q')))
  - set('timestamp', info('timestamp'))
  - set('user_id', get('userId', 'session'))

chat:
  prompt: "Process: {{ get('normalized_input') }}"
```

</div>

## When to Use expr Blocks

### 1. Data Transformation

Transform data before using it:

```yaml
# resources/example.yaml
after:
  - set('cleaned_data', trim(get('rawData')))
  - set('request_time', info('timestamp'))

httpClient:
  url: "https://api.example.com/process"
  data:
    input: get('cleaned_data')
    timestamp: get('request_time')
```

### 2. Memory/Session Operations

Store values for later use:

<div v-pre>

```yaml
# resources/example.yaml
after:
  - set('last_query', get('q'), 'memory')
  - set('query_count', (get('query_count', 'memory') ?? 0) + 1, 'memory')
  - set('user_preferences', get('prefs'), 'session')

chat:
  prompt: "User asked {{ get('query_count') }} times. Query: {{ get('q') }}"
```

</div>

### 3. Calculations

Perform calculations before execution:

```yaml
# resources/example.yaml
after:
  - set('total', get('price') * get('quantity'))
  - set('discount', get('total') * 0.1)
  - set('final_price', get('total') - get('discount'))

apiResponse:
  response:
    total: get('total')
    discount: get('discount')
    final_price: get('final_price')
```

### 4. Conditional Logic

Set values based on conditions:

```yaml
# resources/example.yaml
after:
  - set('mode', get('env') == 'production' ? 'strict' : 'debug')
  - set('cache_ttl', get('mode') == 'strict' ? '1h' : '5m')

httpClient:
  url: "https://api.example.com/data"
  cache:
    ttl: get('cache_ttl')
```

## Execution Order

Expressions in each block run top to bottom. The full per-resource order is:

```d2
direction: down

A: Request {shape: oval}
B: "before:\nexpressions run in order\nvalues stored via set()"
C: "Main Action\nchat, httpClient, sql, python, exec, file, git, codeIntelligence, apiResponse"
D: "after:\nexpressions run in order\noutput accessible via get()"
E: response {shape: oval}

A -> B -> C -> D -> E
```

## Common Patterns

### Pattern 1: Data Normalization

```yaml
# resources/example.yaml
after:
  - set('normalized_email', lower(trim(get('email'))))
  - set('normalized_name', trim(get('name')))

sql:
  query: |
    INSERT INTO users (email, name) 
    VALUES ($1, $2)
  params:
    - get('normalized_email')
    - get('normalized_name')
```

### Pattern 2: Counter/State Management

<div v-pre>

```yaml
# resources/example.yaml
after:
  - set('request_id', info('ID'))
  - set('request_count', (get('request_count', 'memory') ?? 0) + 1, 'memory')
  - set('last_request_time', info('timestamp'), 'memory')

chat:
  prompt: "Request #{{ get('request_count') }}: {{ get('q') }}"
```

</div>

### Pattern 3: Data Aggregation

```yaml
# resources/example.yaml
after:
  - set('items', get('previousResource'))
  - set('total_items', len(get('items')))
  - set('total_value', sum(map(get('items'), .price)))

apiResponse:
  response:
    summary:
      count: get('total_items')
      total: get('total_value')
```

### Pattern 4: Error Handling Preparation

```yaml
# resources/example.yaml
after:
  - set('fallback_value', get('default', 'memory') ?? 'N/A')
  - set('retry_count', get('retry_count', 'memory') ?? 0)

httpClient:
  url: "https://api.example.com/data"
  retry:
    maxAttempts: 3
```

## Expression-Only Resources

You can create resources that only execute expressions (no main action):

```yaml

actionId: setupContext
name: Setup Context
after:
  - set('session_id', info('ID'))
  - set('start_time', info('timestamp'))
  - set('environment', get('ENV', 'env') ?? 'development')
  - set('user_context', get('user', 'session') ?? {})
```

This resource returns `{"status": "expressions_executed"}` and can be used as a dependency for other resources.

## Accessing expr Results

Values set in `expr` blocks are immediately available via `get()`:

<div v-pre>

```yaml
# resources/example.yaml
after:
  - set('processed_data', lower(trim(get('raw_data'))))

# Use the processed data
chat:
  prompt: "Analyze: {{ get('processed_data') }}"
```

</div>

## Best Practices

### 1. Keep Expressions Simple

```yaml
# Good: Simple, clear expressions
after:
  - set('normalized', trim(get('input')))
  - set('count', len(get('items')))

# Avoid: Complex logic (use Python resource instead)
# after:
#   - set('result', get('a') * get('b') / get('c') + get('d') - ...)
```

### 2. Use for Side Effects

```yaml
# Good: Storing state
after:
  - set('last_action', get('action'), 'memory')
  - set('timestamp', info('timestamp'), 'session')

# Avoid: Complex data processing (use Python)
```

### 3. Order Matters

Expressions execute in sequence:

```yaml
# Correct order
after:
  - set('step1', trim(get('input')))
  - set('step2', lower(get('step1')))
  - set('final', replace(get('step2'), ' ', '_'))

# Wrong: step2 depends on step1, must come after
```

## Limitations

- **No return values**: Expressions don't return values (use `set()` to store results)
- **Sequential execution**: Expressions run one after another
- **Error handling**: If an expression fails, the resource fails
- **Complex logic**: For complex operations, use Python resources

## Examples

### Example 1: Request Logging

<div v-pre>

```yaml

actionId: loggedRequest
name: Logged Request
after:
  - set('request_id', info('ID'))
  - set('request_time', info('timestamp'))
  - set('request_log', {
      'id': get('request_id'),
      'time': get('request_time'),
      'query': get('q')
    }, 'memory')

chat:
  prompt: "{{ get('q') }}"
```

</div>

### Example 2: Data Validation and Transformation

<div v-pre>

```yaml

actionId: validatedInput
name: Validated Input
after:
  - set('email', lower(trim(get('email'))))
  - set('age', int(get('age')))
  - set('is_valid', get('email') contains '@' && get('age') >= 18)

validations:
  check:
    - get('is_valid') == true
  error:
    code: 400
    message: Invalid email or age

chat:
  prompt: "Process user: {{ get('email') }}, age {{ get('age') }}"
```

</div>

### Example 3: Session Management

<div v-pre>

```yaml

actionId: sessionHandler
name: Session Handler
after:
  - set('session_id', get('session_id', 'session') ?? info('sessionId'), 'session')
  - set('visit_count', (get('visit_count', 'session') ?? 0) + 1, 'session')
  - set('last_visit', info('timestamp'), 'session')

apiResponse:
  response:
    session_id: get('session_id')
    visit_count: get('visit_count')
    last_visit: get('last_visit')
```

</div>

## See Also

- [Expressions](/concepts/expressions) - Expression syntax and operators
- [Unified API](/concepts/unified-api) - Using `get()` and `set()`
- [Resources Overview](../resources/overview.md) - Resource structure
- [Validation and Control Flow](/concepts/validation-and-control) - Preflight checks
