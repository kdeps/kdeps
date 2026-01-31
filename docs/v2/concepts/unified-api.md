# Unified API

KDeps v2 simplifies data access with a unified API. Just a few core functions provide access to all data sources, replacing the complex API surface of previous versions.

## Core Functions

These functions are available in all expression contexts (string interpolation <span v-pre>`{{ }}`</span> and `expr` blocks).

### 1. get()
The workhorse function for retrieving data. It uses a smart priority chain to find what you're looking for.

```yaml
# Auto-detect source
query: get('q')
```

**Priority Chain:**
1. **Items** (Iteration context)
2. **Memory** (Request-scoped storage)
3. **Session** (User persistent storage)
4. **Outputs** (Resource execution results)
5. **Query** (URL parameters)
6. **Body** (Request body)
7. **Headers** (HTTP headers)
8. **Files** (Uploaded files)
9. **Metadata** (System info)

**Explicit Source:**
You can bypass the chain by specifying a type hint:

```yaml
get('q', 'param')      # Force query/body param
get('auth', 'header')  # Force header
get('user', 'session') # Force session
```

### 2. set()
Stores data for later use.

```yaml
# Store in memory (current request only)
expr:
  - set('count', 1)

# Store in session (persists across requests)
expr:
  - set('user_id', '123', 'session')
```

### 3. file()
Accesses file content.

```yaml
# Get uploaded file content
content: file('doc.pdf')

# Get by pattern
images: file('*.jpg')
```

### 4. info()
Accesses request and system metadata.

```yaml
id: info('requestId')
ip: info('clientIp')
path: info('path')
```

## Advanced Functions

For more specialized needs, additional helper functions are available:

- `json(data)` - Convert data to JSON string
- `safe(obj, path)` - Safely access nested properties
- `debug(obj)` - Inspect objects
- `default(val, fallback)` - Handle missing values

See the [Expression Functions Reference](expression-functions-reference) for the complete list.

## Resource Accessors

While `get('resourceId')` is the standard way to access outputs, resource-specific accessors provide granular data:

- **Python/Exec**: `python.stdout('id')`, `python.stderr('id')`, `python.exitCode('id')`
- **HTTP**: `http.responseBody('id')`, `http.responseHeader('id', 'Name')`
- **LLM**: `llm.response('id')`, `llm.prompt('id')`

These are typically used in `expr` blocks for conditional logic or error handling.

## Next Steps

- [Request Object](request-object) - HTTP request data and file methods
- [Resources Overview](../resources/overview) - Learn about resource types
- [Tools](tools) - LLM function calling
- [Workflow Configuration](../configuration/workflow) - Session settings