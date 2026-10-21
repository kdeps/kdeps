# Inline resources

Inline resources are full resource actions (chat, component, codeIntelligence) placed directly inside a resource's `before:` or `after:` block. They run as part of that resource instead of requiring separate files.

In workflow mode they are a DAG feature. In agent mode they run inside the workflow when the LLM calls that workflow as a tool.

## Basic syntax

```yaml
# resources/example.yaml
actionId: example
name: Example Resource
# Inline resources to run BEFORE the main resource
before:
  - component:                     # HTTP steps go through the http_request component
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/config"
  - component:                     # shell steps go through the exec component
      name: exec
      with:
        command: "echo 'Preparing environment...'"

# Main resource (chat, component, codeIntelligence, or file)
chat:
  role: user
  prompt: "Process this data"

# Inline resources to run AFTER the main resource
after:
  - component:
      name: sql
      with:
        connectionName: main  # DSN defined in ~/.kdeps/config.yaml sql_connections.main.connection
        query: "INSERT INTO logs VALUES (?)"
  - component:
      name: python
      with:
        script: "print('Post-processing complete')"
```

## Supported resource types

Each inline resource can be any supported execution block, including `chat`, `agent`, `component`, `codeIntelligence`, `loader`, `vectorStore`, or `apiResponse`. Every execution type works as either the resource primary block or as an inline step in `before:`/`after:`.

`exec:`, `python:`, `embedding:`, and `httpClient:` are not available inline: shell steps call the `exec` component (`component: {name: exec, with: {command: ...}}`), scripts call the `python` component, embedding steps call the `embedding` component, and HTTP steps call the `http_request` component, the same as anywhere else in a workflow.

When `apiResponse` is the only primary block and the resource has `before:` or `after:` inline steps, the response is evaluated exactly once - after all `before:` steps have run - so API clients always receive a single response object.

## Execution order

```d2
direction: down

A: "expressions in before:\nset() / get() statements"
B: "inline resources in before:\ncomponent, chat, etc."
C: "main action\nchat, sql, component..."
D: "inline resources in after:\ncomponent, codeIntelligence, etc."
E: "expressions in after:\nset() / get() statements"

A -> B -> C -> D -> E
```

## Common use cases

### 1. Data enrichment

Fetch additional data before processing:

```yaml
# resources/example.yaml
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/user/{{get('user_id')}}"
        timeout: 5s

chat:
  prompt: "Analyze user: {{get('_output')}}"
```

### 2. Logging and auditing

Record operations in a database:

```yaml
# resources/example.yaml
chat:
  prompt: "{{get('prompt')}}"

after:
  - component:
      name: sql
      with:
        connectionName: logs_db  # DSN defined in ~/.kdeps/config.yaml sql_connections.logs_db.connection
        query: "INSERT INTO audit_log (action, timestamp) VALUES (?, NOW())"
        params: ["chat_completion"]
```

### 3. Notifications

Send alerts after completion:

```yaml
# resources/example.yaml
component:
  name: python
  with:
    script: "process_data.py"

after:
  - component:
      name: http_request
      with:
        method: POST
        url: "https://api.example.com/notify"
        data:
          status: "completed"
          timestamp: "{{now()}}"
```

### 4. Environment setup

Prepare files or environment before execution:

```yaml
# resources/example.yaml
before:
  - component:                     # the exec component runs each shell step
      name: exec
      with:
        command: "mkdir -p /tmp/workspace"
  - component:
      name: exec
      with:
        command: "cp config.json /tmp/workspace/"

component:
  name: python
  with:
    script: "process_with_config.py"

after:
  - component:
      name: exec
      with:
        command: "rm -rf /tmp/workspace"
```

### 5. Caching

Store results for future use:

```yaml
# resources/example.yaml
chat:
  prompt: "{{get('query')}}"

after:
  - component:
      name: sql
      with:
        connectionName: cache_db  # DSN defined in ~/.kdeps/config.yaml sql_connections.cache_db.connection
        query: "INSERT INTO response_cache (query_hash, response) VALUES (?, ?) ON CONFLICT (query_hash) DO UPDATE SET response = excluded.response"
        params: ["{{get('query_hash')}}", "{{get('_output')}}"]
```

## Multiple inline resources

You can have multiple inline resources of the same or different types:

```yaml
# resources/example.yaml
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/config"
  - component:
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/user"
  - component:
      name: exec
      with:
        command: "echo 'Starting...'"

chat:
  prompt: "{{get('prompt')}}"

after:
  - component:
      name: sql
      with:
        connectionName: main  # DSN defined in ~/.kdeps/config.yaml sql_connections.main.connection
        query: "INSERT INTO results VALUES (?)"
  - component:
      name: python
      with:
        script: "send_metrics.py"
  - component:
      name: http_request
      with:
        method: POST
        url: "https://api.example.com/complete"
```

## Resources without main type

You can have a resource with only inline resources and no main resource type:

```yaml
# resources/example.yaml
before:
  - component:
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/data"

after:
  - component:
      name: sql
      with:
        connectionName: main  # DSN defined in ~/.kdeps/config.yaml sql_connections.main.connection
        query: "INSERT INTO cache VALUES (?)"
```

This is useful for orchestration tasks where you need to coordinate multiple operations.

## Error handling

If an inline resource fails:
- Execution stops immediately
- The error is propagated to the resource level
- Subsequent inline resources are not executed
- The main resource is not executed (if the failure occurred in `before`)

You can use the resource's `onError` configuration to handle errors:

```yaml
# resources/example.yaml
before:
  - component:
      name: http_request
      with:
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

## Accessing context

Inline resources have access to the full execution context:

```yaml
# resources/example.yaml
before:
  - set('user_id', get('input.user_id'))

before:
  # Access variables set in before
  - component:
      name: http_request
      with:
        method: GET
        url: "https://api.example.com/user/{{get('user_id')}}"

chat:
  # Access results from previous steps
  prompt: "User data: {{get('_output')}}"
```

## Configuration options

Each inline resource supports the same configuration options as the standalone resource:

### HTTP request (component)
```yaml
# resources/example.yaml
- component:
    name: http_request
    with:
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
- component:
    name: sql
    with:
      connectionName: main  # DSN defined in ~/.kdeps/config.yaml sql_connections.main.connection
      query: "SELECT * FROM users WHERE id = ?"
      params:
        - "{{get('user_id')}}"
      timeout: 5s
```

### Python (component)
```yaml
# resources/example.yaml
- component:
    name: python
    with:
      script: |
        import json
        result = process_data()
        print(json.dumps(result))
      timeout: 30s
      venvName: "myenv"
```

### Exec (component)
```yaml
# resources/example.yaml
- component:
    name: exec
    with:
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

## Best practices

1. **Keep inline resources focused**: Each should perform a single, well-defined task
2. **Use descriptive configurations**: Make it clear what each inline resource does
3. **Handle errors appropriately**: Consider using `onError` for critical workflows
4. **Set appropriate timeouts**: Prevent hanging on slow operations
5. **Order matters**: Inline resources execute sequentially in the order defined
6. **Use expressions**: Access context data with <span v-pre>`{{get('variable')}}`</span>
7. **Consider alternatives**: For complex workflows, separate resources may be clearer

## Comparison with separate resources

### Traditional approach (separate resources)
```yaml
# 5 separate resource files
- fetch-config.yaml
- prepare-env.yaml
- main-processing.yaml
- store-results.yaml
- send-notification.yaml
```

### With inline resources
```yaml
# Single resource file
before:
  - component: { name: http_request, with: { ... } }  # Fetch config
  - component: { name: exec, with: { ... } }          # Prepare env
chat: { ... }                                         # Main processing
after:
  - component: { name: sql, with: { ... } }           # Store results
  - component: { name: http_request, with: { ... } }  # Send notification
```

**Benefits:**
- Fewer files to manage
- Related operations grouped together
- Clearer execution flow
- Reduced boilerplate
- Easier to understand and maintain

## Advanced patterns

### Conditional inline resources

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

### Combining with items

Inline resources work with the `items` feature:

```yaml
# resources/example.yaml
items:
  - item1
  - item2

before:
  - component:
      name: http_request
      with:
        url: "https://api.example.com/prepare/{{item.current()}}"

chat:
  prompt: "Process {{item.current()}}"

after:
  - component:
      name: sql
      with:
        query: "INSERT INTO results VALUES (?)"
        params: ["{{item.current()}}"]
```

## See also

- [Expression blocks](/reference/expr-blocks) - Using `before` and `after`
- [Error handling](error-handling.md) - Handling errors in resources
- [Items](items.md) - Iterating over collections
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples/inline-resources) - Complete example with inline resources
