# Validation and Control Flow

The `validations:` block controls whether a resource runs and what it accepts. It fires before the action -- before any LLM call, HTTP request, or script execution.

```yaml
validations:
  methods: [POST]          # skip unless method matches
  routes: [/api/v1/data]  # skip unless route matches
  headers: [Authorization] # skip unless header present
  params: [q]              # skip unless param present
  skip:
    - get('mode') == 'dry-run'  # skip silently (no error)
  check:
    - get('q') != ''            # fail with error below if false
  error:
    code: 400
    message: "q is required"
```

Fields summary:

| Field | Logic | Behavior when condition triggers |
|-------|-------|--------------------------------|
| `methods` / `routes` | match | skip silently if no match |
| `headers` / `params` | match | skip silently if no match |
| `skip` | OR -- any true | skip silently |
| `check` | AND -- all must be true | return error to caller |

## Skip Conditions

Skip conditions allow you to conditionally skip resource execution based on runtime values.

### Basic Usage

<div v-pre>

```yaml
actionId: conditionalResource
name: Conditional Resource
validations:
  skip:
    - get('skip') == true
    - get('mode') == 'dry-run'
chat:
  prompt: "{{ get('q') }}"
```

</div>

### How It Works

Any true condition skips the resource silently -- it produces no output but its slot in the dependency graph still exists so downstream resources can reference it. Use `get()` to access any data source in skip conditions.

### Common Patterns

```yaml
validations:
  skip:
    # Skip if flag is set
    - get('skip') == true
    # Skip if no query parameter
    - get('q') == '' || get('q') == null
    # Skip based on item value (in items iteration)
    - get('current') == 'skip_this'
    # Skip if previous resource failed
    - get('previousResource') == null
```

## Preflight Checks

Preflight checks validate inputs **before** resource execution begins. If any condition fails, execution is aborted with a custom error.

### Basic Usage

<div v-pre>

```yaml
actionId: validatedResource
name: Validated Resource
validations:
  check:
    - get('q') != ''
    - get('userId') != null
    - len(get('q')) > 3
  error:
    code: 400
    message: Query parameter 'q' is required and must be at least 3 characters
chat:
  prompt: "{{ get('q') }}"
```

</div>

### How It Works

All `check` conditions must pass (AND logic). If any one fails, execution stops immediately and the configured `error` is returned to the caller -- no LLM call, no HTTP request, no work done.

### Check Expressions

<div v-pre>

```yaml
validations:
  check:
    - get('q') != ''
    - get('userId') != null
    - typeof(get('age')) == 'number'
    - get('age') >= 18
    - len(get('email')) > 5
    - get('email').includes('@')
    - get('status') == 'active' || get('status') == 'pending'
    - get('previousResource') != null
  error:
    code: 400
    message: "Validation failed"
```

</div>

### Error Response

When a `check` validation fails:

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Query parameter 'q' is required and must be at least 3 characters"
  }
}
```

## Route and Method Restrictions

Limit which HTTP requests can trigger a resource.

### Basic Usage

<div v-pre>

```yaml
actionId: apiResource
name: API Resource
validations:
  methods: [GET, POST]
  routes: [/api/v1/data, /api/v1/query]
chat:
  prompt: "{{ get('q') }}"
```

</div>

### How It Works

- Resource only executes if **both** conditions match
- If restrictions don't match, resource is skipped silently
- Empty arrays mean "allow all"

### Method Restrictions

```yaml
validations:
  methods: [GET]         # only GET
  methods: [GET, POST]   # GET and POST
  # omit to allow all methods
```

### Route Restrictions

```yaml
validations:
  routes: [/api/v1/users]                   # single route
  routes: [/api/v1/users, /api/v1/profiles] # multiple routes
  # omit to allow all routes
```

### Combined Example

<div v-pre>

```yaml
validations:
  methods: [POST]
  routes:
    - /api/v1/create
    - /api/v1/update
chat:
  prompt: "Create: {{ get('data') }}"
```

</div>

## Input Validation

Validate the structure and content of request data using `required`, `rules`, and `expr`.

### Basic Usage

<div v-pre>

```yaml
actionId: validatedInput
name: Validated Input
validations:
  required:
    - userId
    - action
  properties:
    userId:
      type: string
      minLength: 1
    action:
      type: string
      enum: [create, update, delete]
    age:
      type: number
      minimum: 18
      maximum: 120
chat:
  prompt: "{{ get('action') }} user {{ get('userId') }}"
```

</div>

### Validation Syntax

KDeps supports multiple syntaxes for field validation:

**`properties` (map format)**
```yaml
validations:
  required: [email, name]
  properties:
    email:
      type: string
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    name:
      type: string
      minLength: 1
```

**`rules` (array format)**
```yaml
validations:
  required: [email, name]
  rules:
    - field: email
      type: string
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    - field: name
      type: string
      minLength: 1
```

**`fields` (alternative map format)**
```yaml
validations:
  required: [email, name]
  fields:
    email:
      type: string
    name:
      type: string
      minLength: 1
```

### Validation Rules Reference

| Rule | Type | Description |
|------|------|-------------|
| `required` | array | List of required fields |
| `properties` / `fields` / `rules` | object/array | Field-specific validation rules |
| `type` | string | `string`, `number`, `integer`, `boolean`, `object`, `array`, `email`, `url`, `uuid`, `date` |
| `minLength` | number | Minimum string length |
| `maxLength` | number | Maximum string length |
| `minimum` / `min` | number | Minimum numeric value |
| `maximum` / `max` | number | Maximum numeric value |
| `enum` | array | Allowed values |
| `pattern` | string | Regex pattern (for strings) |
| `minItems` | number | Minimum array items |
| `maxItems` | number | Maximum array items |
| `message` | string | Custom error message for this field |

### Custom Expression Rules

<div v-pre>

```yaml
validations:
  required: [email, password, confirmPassword]
  properties:
    email:
      type: string
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    password:
      type: string
      minLength: 8
    confirmPassword:
      type: string
  expr:
    - "get('password') == get('confirmPassword')"
    - "get('email').includes('@')"
chat:
  prompt: "Process user: {{ get('email') }}"
```

</div>

## Allowed Headers and Parameters

Restrict which headers and query parameters are allowed in requests.

### Allowed Headers

<div v-pre>

```yaml
validations:
  headers:
    - Authorization
    - Content-Type
    - X-API-Key
chat:
  prompt: "{{ get('q') }}"
```

</div>

Headers not in this list are inaccessible via `get()`.

### Allowed Parameters

<div v-pre>

```yaml
validations:
  params:
    - q
    - userId
    - action
chat:
  prompt: "{{ get('q') }}"
```

</div>

Parameters not in this list are inaccessible via `get()`.

### Combined Example

<div v-pre>

```yaml
validations:
  methods: [POST]
  routes: [/api/v1/secure]
  headers:
    - Authorization
    - Content-Type
  params:
    - action
chat:
  prompt: "Secure action: {{ get('action') }}"
```

</div>

## Execution Order

```
Request
  ↓ methods / routes     → skip if no match
  ↓ headers / params     → filter inaccessible keys
  ↓ skip conditions      → skip if any true
  ↓ check + error        → abort with error if any false
  ↓ required/rules/expr  → abort with 422 if invalid
  ↓ Execute Resource
```

## Best Practices

### 1. Use `skip` for Optional Logic

```yaml
validations:
  skip:
    - get('enableCache') != true
```

### 2. Validate Early with `check`

<div v-pre>

```yaml
# Good: Catch errors before expensive operations
validations:
  check:
    - get('userId') != null
    - get('apiKey') != ''
  error:
    code: 400
    message: "userId and apiKey are required"
```

</div>

### 3. Restrict Routes for Security

<div v-pre>

```yaml
validations:
  routes: [/api/v1/admin]
  methods: [POST]
```

</div>

### 4. Combine All Controls

<div v-pre>

```yaml
validations:
  methods: [POST]
  routes: [/api/v1/admin]
  headers: [Authorization]
  check:
    - get('adminToken') != null
  error:
    code: 401
    message: Admin token required
  skip:
    - get('dryRun') == true
chat:
  prompt: "Admin: {{ get('action') }}"
```

</div>

## Examples

### Example 1: Conditional Processing

```yaml
actionId: smartProcessor
name: Smart Processor
validations:
  skip:
    - get('process') != true
  check:
    - get('data') != null
    - len(get('data')) > 0
  error:
    code: 400
    message: Data is required
python:
  script: |
    data = get('data')
    return process(data)
```

### Example 2: Secure Endpoint

<div v-pre>

```yaml
actionId: secureEndpoint
name: Secure Endpoint
validations:
  methods: [POST]
  routes: [/api/v1/secure]
  headers: [Authorization, Content-Type]
  check:
    - get('Authorization') != null
    - get('Authorization').startsWith('Bearer ')
  error:
    code: 401
    message: Valid authorization token required
chat:
  prompt: "Secure: {{ get('q') }}"
```

</div>

## See Also

- [Validation](/concepts/validation) — Full `validations:` block reference
- [Expressions](/concepts/expressions) — Expression syntax for conditions
- [Resources Overview](../resources/overview.md) — Resource structure
- [Unified API](/concepts/unified-api) — Using `get()` in validations
- [Workflow Configuration](../configuration/workflow.md) — Route configuration
