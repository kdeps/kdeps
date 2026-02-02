# Expression Blocks (expr)

Expression blocks (`expr`) allow you to execute expressions before the main resource action runs. This is useful for pre-processing, data transformation, and side effects like storing values in memory or session.

## Overview

The `expr` block executes expressions **before** the resource's main action (chat, httpClient, sql, etc.). Expressions run in sequence and can:

- Transform data with `set()`
- Store values in memory or session
- Perform calculations
- Prepare data for the main action

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: preProcessor
  name: Pre-Processor

run:
  expr:
    - set('normalized_input', get('q').toLowerCase())
    - set('timestamp', info('now'))
    - set('user_id', get('userId', 'session'))
  
  chat:
    model: llama3.2:1b
    prompt: "Process: {{ get('normalized_input') }}"
```

</div>

## When to Use expr Blocks

### 1. Data Transformation

Transform data before using it:

```yaml
run:
  expr:
    - set('cleaned_data', get('rawData').trim())
    - set('formatted_date', formatDate(get('date'), 'YYYY-MM-DD'))
  
  httpClient:
    url: "https://api.example.com/process"
    data:
      input: get('cleaned_data')
      date: get('formatted_date')
```

### 2. Memory/Session Operations

Store values for later use:

<div v-pre>

```yaml
run:
  expr:
    - set('last_query', get('q'), 'memory')
    - set('query_count', get('query_count', 'memory', 0) + 1, 'memory')
    - set('user_preferences', get('prefs'), 'session')
  
  chat:
    model: llama3.2:1b
    prompt: "User asked {{ get('query_count') }} times. Query: {{ get('q') }}"
```

</div>

### 3. Calculations

Perform calculations before execution:

```yaml
run:
  expr:
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
run:
  expr:
    - set('mode', get('env') == 'production' ? 'strict' : 'debug')
    - set('cache_ttl', get('mode') == 'strict' ? '1h' : '5m')
  
  httpClient:
    url: "https://api.example.com/data"
    cache:
      enabled: true
      ttl: get('cache_ttl')
```

## Execution Order

Expressions execute in this order:

1. **expr block** - Pre-processing expressions
2. **Main action** - chat, httpClient, sql, python, exec, or apiResponse

```
Request
    ↓
┌─────────────┐
│ expr Block  │ → Execute expressions in sequence
└──────┬──────┘
       ↓
┌─────────────┐
│ Main Action │ → Execute resource action
└─────────────┘
```

## Common Patterns

### Pattern 1: Data Normalization

```yaml
run:
  expr:
    - set('normalized_email', get('email').toLowerCase().trim())
    - set('normalized_name', get('name').trim())
  
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
run:
  expr:
    - set('request_id', generateUUID())
    - set('request_count', get('request_count', 'memory', 0) + 1, 'memory')
    - set('last_request_time', info('now'), 'memory')
  
  chat:
    model: llama3.2:1b
    prompt: "Request #{{ get('request_count') }}: {{ get('q') }}"
```

</div>

### Pattern 3: Data Aggregation

```yaml
run:
  expr:
    - set('items', get('previousResource'))
    - set('total_items', len(get('items')))
    - set('total_value', sum(get('items').map(item => item.price)))
  
  apiResponse:
    response:
      summary:
        count: get('total_items')
        total: get('total_value')
```

### Pattern 4: Error Handling Preparation

```yaml
run:
  expr:
    - set('fallback_value', get('default', 'memory', 'N/A'))
    - set('retry_count', get('retry_count', 'memory', 0))
  
  httpClient:
    url: "https://api.example.com/data"
    retry:
      maxAttempts: 3
```

## Expression-Only Resources

You can create resources that only execute expressions (no main action):

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: setupContext
  name: Setup Context

run:
  expr:
    - set('session_id', generateUUID())
    - set('start_time', info('now'))
    - set('environment', get('ENV', 'env', 'development'))
    - set('user_context', get('user', 'session', {}))
```

This resource returns `{"status": "expressions_executed"}` and can be used as a dependency for other resources.

## Accessing expr Results

Values set in `expr` blocks are immediately available via `get()`:

<div v-pre>

```yaml
run:
  expr:
    - set('processed_data', processData(get('raw_data')))
  
  # Use the processed data
  chat:
    prompt: "Analyze: {{ get('processed_data') }}"
```

</div>

## Best Practices

### 1. Keep Expressions Simple

```yaml
# Good: Simple, clear expressions
expr:
  - set('normalized', get('input').trim())
  - set('count', len(get('items')))

# Avoid: Complex logic (use Python resource instead)
expr:
  - set('result', complexCalculation(get('data')))
```

### 2. Use for Side Effects

```yaml
# Good: Storing state
expr:
  - set('last_action', get('action'), 'memory')
  - set('timestamp', info('now'), 'session')

# Avoid: Complex data processing (use Python)
```

### 3. Order Matters

Expressions execute in sequence:

```yaml
# Correct order
expr:
  - set('step1', process(get('input')))
  - set('step2', process(get('step1')))
  - set('final', process(get('step2')))

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
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: loggedRequest
  name: Logged Request

run:
  expr:
    - set('request_id', generateUUID())
    - set('request_time', info('now'))
    - set('request_log', {
        'id': get('request_id'),
        'time': get('request_time'),
        'query': get('q')
      }, 'memory')
  
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
```

</div>

### Example 2: Data Validation and Transformation

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: validatedInput
  name: Validated Input

run:
  expr:
    - set('email', get('email').toLowerCase().trim())
    - set('age', parseInt(get('age')))
    - set('is_valid', get('email').includes('@') && get('age') >= 18)
  
  preflightCheck:
    validations:
      - get('is_valid') == true
    error:
      code: 400
      message: Invalid email or age
  
  chat:
    model: llama3.2:1b
    prompt: "Process user: {{ get('email') }}, age {{ get('age') }}"
```

</div>

### Example 3: Session Management

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: sessionHandler
  name: Session Handler

run:
  expr:
    - set('session_id', get('session_id', 'session', generateUUID()), 'session')
    - set('visit_count', get('visit_count', 'session', 0) + 1, 'session')
    - set('last_visit', info('now'), 'session')
  
  apiResponse:
    response:
      session_id: get('session_id')
      visit_count: get('visit_count')
      last_visit: get('last_visit')
```

</div>

## Related Documentation

- [Expressions](expressions.md) - Expression syntax and operators
- [Unified API](unified-api.md) - Using `get()` and `set()`
- [Resources Overview](../resources/overview.md) - Resource structure
- [Validation and Control Flow](validation-and-control.md) - Preflight checks
