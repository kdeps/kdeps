# Inline Resources

Inline resources are full resource actions (chat, httpClient, sql, python, exec, file) placed directly inside a resource's `before:` or `after:` block. They run as part of that resource instead of requiring separate files.

## Basic Syntax

```yaml
# resources/example.yaml
actionId: example
name: Example Resource
# Inline resources to run BEFORE the main resource
before:
  - httpClient:
      method: GET
      url: "https://api.example.com/config"
  - exec:
      command: "echo 'Preparing environment...'"

# Main resource (chat, httpClient, sql, python, exec, or file)
chat:
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

Each inline resource can be any supported execution block, including `chat`, `httpClient`, `sql`, `python`, `exec`, `agent`, `component`, `scraper`, `embedding`, `searchLocal`, `searchWeb`, `telephony`, `browser`, `botReply`, `email`, `file`, or `apiResponse`. Every execution type works as either the resource primary block or as an inline step in `before:`/`after:`.

When `apiResponse` is the only primary block and the resource has `before:` or `after:` inline steps, the response is evaluated exactly once - after all `before:` steps have run - so API clients always receive a single response object.

## Execution Order

```d2
direction: down

A: "expressions in before:\nset() / get() statements"
B: "inline resources in before:\nhttpClient, exec, etc."
C: "main action\nchat, sql, python, exec..."
D: "inline resources in after:\nhttpClient, sql, etc."
E: "expressions in after:\nset() / get() statements"

A -> B -> C -> D -> E
```

## Common Use Cases

### 1. Data Enrichment

Fetch additional data before processing:

```yaml
# resources/example.yaml
before:
  - httpClient:
      method: GET
      url: "https://api.example.com/user/{{get('user_id')}}"
      timeout: 5s

chat:
  prompt: "Analyze user: {{get('_output')}}"
```

### 2. Logging and Auditing

Record operations in a database:

```yaml
# resources/example.yaml
chat:
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
# resources/example.yaml
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
# resources/example.yaml
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
# resources/example.yaml
chat:
  prompt: "{{get('query')}}"

after:
  - sql:
      connection: "redis://localhost"
      query: "SET cache:{{get('query_hash')}} {{get('_output')}}"
```

## Multiple Inline Resources

You can have multiple inline resources of the same or different types:

```yaml
# resources/example.yaml
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
# resources/example.yaml
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
# resources/example.yaml
before:
  - httpClient:
      method: GET
      url: "https://api.example.com/config"

chat:
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
# resources/example.yaml
before:
  - set('user_id', get('input.user_id'))

before:
  # Access variables set in before
  - httpClient:
      method: GET
      url: "https://api.example.com/user/{{get('user_id')}}"

chat:
  # Access results from previous steps
  prompt: "User data: {{get('_output')}}"
```

## Configuration Options

Each inline resource supports the same configuration options as the standalone resource:

### HTTP Client
```yaml
# resources/example.yaml
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
# resources/example.yaml
- sql:
    connectionName: main  # DSN defined in ~/.kdeps/config.yaml sql_connections.main.connection
    query: "SELECT * FROM users WHERE id = ?"
    params:
      - "{{get('user_id')}}"
    timeout: 5s
```

### Python
```yaml
# resources/example.yaml
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
# resources/example.yaml
- exec:
    command: "process_file.sh"
    args:
      - "{{get('filename')}}"
    timeout: 60s
```

### Chat (LLM)
```yaml
# resources/example.yaml
- chat:
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
6. **Use expressions**: Access context data with <span v-pre>`{{get('variable')}}`</span>
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
# resources/example.yaml
before:
  - set('should_notify', get('input.notify') == true)

chat:
  prompt: "{{get('prompt')}}"

after:
  - set('notification_sent', get('should_notify') == true)
```

### Combining with Items

Inline resources work with the `items` feature:

```yaml
# resources/example.yaml
items:
  - item1
  - item2

before:
  - httpClient:
      url: "https://api.example.com/prepare/{{item.current()}}"

chat:
  prompt: "Process {{item.current()}}"

after:
  - sql:
      query: "INSERT INTO results VALUES (?)"
      params: ["{{item.current()}}"]
```

## See Also

- [Expression Blocks](/reference/expr-blocks) - Using `before` and `after`
- [Error Handling](error-handling.md) - Handling errors in resources
- [Items](items.md) - Iterating over collections
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples/inline-resources) - Complete example with inline resources
