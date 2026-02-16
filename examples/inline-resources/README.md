# Inline Resources Example

This example demonstrates the **inline resources** feature in KDeps, which allows you to configure multiple LLM, HTTP, Exec, SQL, and Python resources to run before or after the main resource execution.

## Overview

Inline resources provide a way to:
- Execute preparatory tasks before the main resource runs
- Perform cleanup or post-processing tasks after the main resource completes
- Keep related operations organized within a single resource definition
- Avoid creating separate resource definitions for small auxiliary tasks

## Features Demonstrated

### Before Inline Resources
Resources configured in the `before` section execute **before** the main resource:
- HTTP call to fetch configuration
- Shell command to prepare environment

### Main Resource
The primary resource that performs the core business logic:
- LLM chat processing with Ollama

### After Inline Resources
Resources configured in the `after` section execute **after** the main resource:
- SQL query to store results
- Python script for post-processing
- HTTP call to send notifications

## Execution Order

The execution order for a resource with inline resources is:

1. **ExprBefore** expressions (if any)
2. **Before** inline resources (executed sequentially)
3. **Main resource** (chat, httpClient, sql, python, or exec)
4. **After** inline resources (executed sequentially)
5. **Expr/ExprAfter** expressions (if any)
6. **APIResponse** formatting (if configured)

## Configuration

### Inline Resource Structure

```yaml
run:
  # Inline resources to run before main resource
  before:
    - httpClient:
        method: GET
        url: "https://api.example.com/data"
    - exec:
        command: "echo 'Preparing...'"
  
  # Main resource
  chat:
    model: llama3.2:1b
    prompt: "Process this"
  
  # Inline resources to run after main resource
  after:
    - sql:
        connection: "sqlite3://./db.sqlite"
        query: "INSERT INTO logs VALUES (?)"
    - python:
        script: "print('Done')"
```

### Supported Inline Resource Types

Each inline resource can be one of:
- **chat**: LLM interaction
- **httpClient**: HTTP requests
- **sql**: Database queries
- **python**: Python script execution
- **exec**: Shell command execution

### Error Handling

If an inline resource fails:
- Execution stops immediately
- The error is propagated up
- Subsequent inline resources and main resource are not executed
- Resource-level `onError` configuration can be used to handle errors

## Running the Example

1. Start Ollama:
   ```bash
   ollama pull llama3.2:1b
   ```

2. Create the SQLite database:
   ```bash
   sqlite3 results.db "CREATE TABLE IF NOT EXISTS results (data TEXT, timestamp TEXT)"
   ```

3. Run the workflow:
   ```bash
   kdeps run workflow.yaml --input '{"data": "test data"}'
   ```

## Use Cases

Inline resources are useful for:

1. **Data Enrichment**: Fetch additional data before processing
2. **Environment Setup**: Prepare files or environment variables
3. **Logging and Auditing**: Record operations in a database
4. **Notifications**: Send alerts or updates after completion
5. **Cleanup**: Remove temporary files or reset state
6. **Caching**: Store results for future use

## Best Practices

1. **Keep inline resources focused**: Each should perform a single, well-defined task
2. **Handle errors appropriately**: Use `onError` configuration when needed
3. **Consider timeouts**: Set appropriate timeouts for each inline resource
4. **Use expressions**: Access context data with `{{get('variable')}}`
5. **Order matters**: Inline resources execute sequentially in the order defined

## Comparison with Separate Resources

### Without Inline Resources (Traditional Approach)
```yaml
# workflow.yaml with 4 separate resource files
resources:
  - fetch-config.yaml      # Before
  - prepare-env.yaml       # Before
  - main-processing.yaml   # Main
  - store-results.yaml     # After
  - send-notification.yaml # After
```

### With Inline Resources (New Approach)
```yaml
# Single resource file with inline resources
run:
  before:
    - httpClient: ...    # Fetch config
    - exec: ...          # Prepare env
  chat: ...              # Main processing
  after:
    - sql: ...           # Store results
    - httpClient: ...    # Send notification
```

**Benefits:**
- Fewer files to manage
- Related operations stay together
- Clearer execution flow
- Reduced boilerplate

## Advanced Features

### Combining with ExprBefore/ExprAfter

You can combine inline resources with expression blocks:

```yaml
run:
  exprBefore:
    - set('timestamp', now())
  
  before:
    - httpClient: ...
  
  chat: ...
  
  after:
    - sql: ...
  
  expr:
    - set('duration', now() - get('timestamp'))
```

### Accessing Results

Inline resource results are stored in the execution context and can be accessed by subsequent resources or expressions.

## Notes

- Inline resources are optional - you can use `before`, `after`, both, or neither
- A resource can have inline resources without a main resource type
- Inline resources respect the resource's error handling configuration
- Each inline resource has access to the full execution context
