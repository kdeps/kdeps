# Inline Resources

Inline resources allow you to configure multiple LLM, HTTP, Exec, SQL, and Python resources to execute **before** or **after** the main resource within a single resource definition.

## Overview

Instead of creating separate resource files for preparatory or cleanup tasks, inline resources let you:
- Execute tasks before the main resource runs
- Perform post-processing after the main resource completes
- Keep related operations organized in one place
- Reduce boilerplate and improve readability

## Basic Syntax

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: example
  name: Example Resource
run:
  # Inline resources to run BEFORE the main resource
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/config"
    - exec:
        command: "echo 'Preparing environment...'"
  
  # Main resource (chat, httpClient, sql, python, or exec)
  chat:
    model: llama3.2:1b
    role: user
    prompt: "Process this data"
  
  # Inline resources to run AFTER the main resource
  after:
    - sql:
        connection: "sqlite3://./db.sqlite"
        query: "INSERT INTO logs VALUES (?)"
    - python:
        script: "print('Post-processing complete')"
```

## Supported Resource Types

Each inline resource can be one of:

- **chat**: LLM interaction (Ollama, OpenAI, Anthropic, etc.)
- **httpClient**: HTTP requests
- **sql**: Database queries
- **python**: Python script execution
- **exec**: Shell command execution

## Execution Order

Resources with inline resources execute in the following order:

1. **ExprBefore** expressions (if configured)
2. **Before** inline resources (executed sequentially)
3. **Main resource** (the primary resource type)
4. **After** inline resources (executed sequentially)
5. **Expr/ExprAfter** expressions (if configured)
6. **APIResponse** formatting (if configured)

Example:
```yaml
run:
  exprBefore:
    - set('start_time', now())
  
  before:
    - httpClient: { ... }  # Step 1
    - exec: { ... }        # Step 2
  
  chat: { ... }            # Step 3 (main resource)
  
  after:
    - sql: { ... }         # Step 4
    - python: { ... }      # Step 5
  
  expr:
    - set('duration', now() - get('start_time'))
  
  apiResponse:
    data: { ... }
```

## Common Use Cases

### 1. Data Enrichment

Fetch additional data before processing:

```yaml
run:
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/user/{{get('user_id')}}"
        timeout: 5s
  
  chat:
    model: llama3.2:1b
    prompt: "Analyze user: {{get('_output')}}"
```

### 2. Logging and Auditing

Record operations in a database:

```yaml
run:
  chat:
    model: llama3.2:1b
    prompt: "{{get('prompt')}}"
  
  after:
    - sql:
        connection: "postgresql://localhost/logs"
        query: "INSERT INTO audit_log (action, timestamp) VALUES (?, NOW())"
        params: ["chat_completion"]
```

### 3. Notifications

Send alerts after completion:

```yaml
run:
  python:
    script: "process_data.py"
  
  after:
    - httpClient:
        method: POST
        url: "https://api.example.com/notify"
        data:
          status: "completed"
          timestamp: "{{now()}}"
```

### 4. Environment Setup

Prepare files or environment before execution:

```yaml
run:
  before:
    - exec:
        command: "mkdir -p /tmp/workspace"
    - exec:
        command: "cp config.json /tmp/workspace/"
  
  python:
    script: "process_with_config.py"
  
  after:
    - exec:
        command: "rm -rf /tmp/workspace"
```

### 5. Caching

Store results for future use:

```yaml
run:
  chat:
    model: gpt-4
    prompt: "{{get('query')}}"
  
  after:
    - sql:
        connection: "redis://localhost"
        query: "SET cache:{{get('query_hash')}} {{get('_output')}}"
```

## Multiple Inline Resources

You can have multiple inline resources of the same or different types:

```yaml
run:
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/config"
    - httpClient:
        method: GET
        url: "https://api.example.com/user"
    - exec:
        command: "echo 'Starting...'"
  
  chat:
    model: llama3.2:1b
    prompt: "{{get('prompt')}}"
  
  after:
    - sql:
        connection: "sqlite3://./db.sqlite"
        query: "INSERT INTO results VALUES (?)"
    - python:
        script: "send_metrics.py"
    - httpClient:
        method: POST
        url: "https://api.example.com/complete"
```

## Resources Without Main Type

You can have a resource with only inline resources and no main resource type:

```yaml
run:
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/data"
  
  after:
    - sql:
        connection: "sqlite3://./db.sqlite"
        query: "INSERT INTO cache VALUES (?)"
```

This is useful for orchestration tasks where you need to coordinate multiple operations.

## Error Handling

If an inline resource fails:
- Execution stops immediately
- The error is propagated to the resource level
- Subsequent inline resources are not executed
- The main resource is not executed (if the failure occurred in `before`)

You can use the resource's `onError` configuration to handle errors:

```yaml
run:
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/config"
  
  chat:
    model: llama3.2:1b
    prompt: "{{get('prompt')}}"
  
  onError:
    action: continue
    fallback:
      error: true
      message: "Processing failed"
```

## Accessing Context

Inline resources have access to the full execution context:

```yaml
run:
  exprBefore:
    - set('user_id', get('input.user_id'))
  
  before:
    # Access variables set in exprBefore
    - httpClient:
        method: GET
        url: "https://api.example.com/user/{{get('user_id')}}"
  
  chat:
    model: llama3.2:1b
    # Access results from previous steps
    prompt: "User data: {{get('_output')}}"
```

## Configuration Options

Each inline resource supports the same configuration options as the standalone resource:

### HTTP Client
```yaml
- httpClient:
    method: POST
    url: "https://api.example.com"
    headers:
      Authorization: "Bearer {{get('token')}}"
    data:
      key: "value"
    timeout: 10s
    retry:
      maxAttempts: 3
      backoff: 1s
```

### SQL
```yaml
- sql:
    connection: "postgresql://localhost/db"
    query: "SELECT * FROM users WHERE id = ?"
    params:
      - "{{get('user_id')}}"
    timeout: 5s
```

### Python
```yaml
- python:
    script: |
      import json
      result = process_data()
      print(json.dumps(result))
    timeout: 30s
    venvName: "myenv"
```

### Exec
```yaml
- exec:
    command: "process_file.sh"
    args:
      - "{{get('filename')}}"
    timeout: 60s
```

### Chat (LLM)
```yaml
- chat:
    backend: ollama
    model: llama3.2:1b
    role: user
    prompt: "{{get('prompt')}}"
    timeout: 30s
```

## Best Practices

1. **Keep inline resources focused**: Each should perform a single, well-defined task
2. **Use descriptive configurations**: Make it clear what each inline resource does
3. **Handle errors appropriately**: Consider using `onError` for critical workflows
4. **Set appropriate timeouts**: Prevent hanging on slow operations
5. **Order matters**: Inline resources execute sequentially in the order defined
6. **Use expressions**: Access context data with `{{get('variable')}}`
7. **Consider alternatives**: For complex workflows, separate resources may be clearer

## Comparison with Separate Resources

### Traditional Approach (Separate Resources)
```yaml
# 5 separate resource files
- fetch-config.yaml
- prepare-env.yaml
- main-processing.yaml
- store-results.yaml
- send-notification.yaml
```

### With Inline Resources
```yaml
# Single resource file
run:
  before:
    - httpClient: { ... }  # Fetch config
    - exec: { ... }        # Prepare env
  chat: { ... }            # Main processing
  after:
    - sql: { ... }         # Store results
    - httpClient: { ... }  # Send notification
```

**Benefits:**
- Fewer files to manage
- Related operations grouped together
- Clearer execution flow
- Reduced boilerplate
- Easier to understand and maintain

## Advanced Patterns

### Conditional Inline Resources

Use expressions with inline resources:

```yaml
run:
  exprBefore:
    - set('should_notify', get('input.notify') == true)
  
  chat:
    model: llama3.2:1b
    prompt: "{{get('prompt')}}"
  
  expr:
    - if(get('should_notify'), 
        set('notification_sent', true),
        set('notification_sent', false))
```

### Combining with Items

Inline resources work with the `items` feature:

```yaml
items:
  - item1
  - item2

run:
  before:
    - httpClient:
        url: "https://api.example.com/prepare/{{item()}}"
  
  chat:
    model: llama3.2:1b
    prompt: "Process {{item()}}"
  
  after:
    - sql:
        query: "INSERT INTO results VALUES (?)"
        params: ["{{item()}}"]
```

## See Also

- [Expression Blocks](expr-blocks.md) - Using `exprBefore` and `exprAfter`
- [Error Handling](error-handling.md) - Handling errors in resources
- [Items](items.md) - Iterating over collections
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples/inline-resources) - Complete example with inline resources
