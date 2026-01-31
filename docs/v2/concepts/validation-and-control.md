# Validation and Control Flow

KDeps provides multiple mechanisms to control resource execution and validate inputs before processing.

## Overview

Resources can be controlled through:
- **Skip Conditions** - Skip execution based on runtime conditions
- **Preflight Checks** - Validate inputs before execution
- **Route/Method Restrictions** - Limit which requests trigger the resource
- **Input Validation** - Validate request data structure

## Skip Conditions

Skip conditions allow you to conditionally skip resource execution based on runtime values.

### Basic Usage



<div v-pre>



```yaml

apiVersion: kdeps.io/v1

kind: Resource



metadata:

  actionId: conditionalResource

  name: Conditional Resource



run:

  skipCondition:

    - get('skip') == true

    - get('mode') == 'dry-run'

  chat:

    model: llama3.2:1b

    prompt: "{{ get('q') }}"

```



</div>

### How It Works

- If **any** condition evaluates to `true`, the resource is skipped
- Skipped resources don't execute and produce no output
- Dependencies are still resolved (other resources can still reference it)
- Use `get()` to access any data source

### Common Patterns

```yaml
# Skip if flag is set
skipCondition:
  - get('skip') == true

# Skip for certain routes
skipCondition:
  - get('route') == '/api/v1/admin'

# Skip if no query parameter
skipCondition:
  - get('q') == '' || get('q') == null

# Skip based on item value (in items iteration)
skipCondition:
  - get('current') == 'skip_this'

# Skip if previous resource failed
skipCondition:
  - get('previousResource') == null
```

## Preflight Checks

Preflight checks validate inputs **before** resource execution begins. If validation fails, execution is aborted with a custom error.

### Basic Usage



<div v-pre>



```yaml

apiVersion: kdeps.io/v1

kind: Resource



metadata:

  actionId: validatedResource

  name: Validated Resource



run:

  preflightCheck:

    validations:

      - get('q') != ''

      - get('userId') != null

      - len(get('q')) > 3

    error:

      code: 400

      message: Query parameter 'q' is required and must be at least 3 characters

  chat:

    model: llama3.2:1b

    prompt: "{{ get('q') }}"

```



</div>

### How It Works

- All validations must pass (AND logic)
- If any validation fails, execution stops
- Custom error is returned with specified code and message
- Validations run before any resource action executes

### Validation Expressions

```yaml
preflightCheck:
  validations:
    # Check existence
    - get('q') != ''
    - get('userId') != null
    
    # Check type
    - typeof(get('age')) == 'number'
    
    # Check range
    - get('age') >= 18
    - get('age') <= 120
    
    # Check length
    - len(get('email')) > 5
    - len(get('password')) >= 8
    
    # Check format (using regex-like checks)
    - get('email').includes('@')
    
    # Check multiple conditions
    - get('status') == 'active' || get('status') == 'pending'
    
    # Check resource outputs
    - get('previousResource') != null
```

### Error Response

When preflight validation fails:

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

apiVersion: kdeps.io/v1

kind: Resource



metadata:

  actionId: apiResource

  name: API Resource



run:

  restrictToHttpMethods: [GET, POST]

  restrictToRoutes: ["/api/v1/data", "/api/v1/query"]

  chat:

    model: llama3.2:1b

    prompt: "{{ get('q') }}"

```



</div>

### How It Works

- Resource only executes if **both** conditions match:
  - HTTP method is in `restrictToHttpMethods`
  - Route path matches one in `restrictToRoutes`
- If restrictions don't match, resource is skipped
- Empty arrays mean "allow all"

### Method Restrictions

```yaml
# Only GET requests
restrictToHttpMethods: [GET]

# GET and POST only
restrictToHttpMethods: [GET, POST]

# All methods (default)
restrictToHttpMethods: []  # or omit
```

### Route Restrictions

```yaml
# Single route
restrictToRoutes: ["/api/v1/users"]

# Multiple routes
restrictToRoutes:
  - "/api/v1/users"
  - "/api/v1/profiles"

# All routes (default)
restrictToRoutes: []  # or omit
```

### Combined Example

<div v-pre>

```yaml
run:
  # Only execute for POST requests to specific endpoints
  restrictToHttpMethods: [POST]
  restrictToRoutes:
    - "/api/v1/create"
    - "/api/v1/update"
  chat:
    model: llama3.2:1b
    prompt: "Create: {{ get('data') }}"
```

</div>

## Input Validation

Validate the structure and content of request data using the `validation` block. This includes both schema-based validation and custom expression-based rules.

### Basic Usage



<div v-pre>



```yaml

apiVersion: kdeps.io/v1

kind: Resource



metadata:

  actionId: validatedInput

  name: Validated Input



run:

  validation:

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

    model: llama3.2:1b

    prompt: "{{ get('action') }} user {{ get('userId') }}"

```



</div>

### Validation Syntax

KDeps supports multiple syntaxes for field validation:

**Option 1: Using `properties` (map format)**
```yaml
validation:
  required: [email, name]
  properties:
    email:
      type: string
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    name:
      type: string
      minLength: 1
```

**Option 2: Using `rules` (array format)**
```yaml
validation:
  required: [email, name]
  rules:
    - field: email
      type: string
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    - field: name
      type: string
      minLength: 1
```

**Option 3: Using `fields` (alternative map format)**
```yaml
validation:
  required: [email, name]
  fields:
    email:
      type: string
      pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    name:
      type: string
      minLength: 1
```

### Validation Rules

| Rule | Type | Description |
|------|------|-------------|
| `required` | array | List of required fields |
| `properties` / `fields` / `rules` | object/array | Field-specific validation rules |
| `type` | string | Data type: `string`, `number`, `integer`, `boolean`, `object`, `array`, `email`, `url`, `uuid`, `date` |
| `minLength` | number | Minimum string length |
| `maxLength` | number | Maximum string length |
| `minimum` / `min` | number | Minimum numeric value |
| `maximum` / `max` | number | Maximum numeric value |
| `enum` | array | Allowed values |
| `pattern` | string | Regex pattern (for strings) |
| `minItems` | number | Minimum array items |
| `maxItems` | number | Maximum array items |
| `message` | string | Custom error message for this field |

### Custom Validation Rules



In addition to schema validation, you can define custom expression-based validation rules:



<div v-pre>



```yaml

run:

  validation:

    required:

      - email

      - password

      - confirmPassword

    properties:

      email:

        type: string

        pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

      password:

        type: string

        minLength: 8

      confirmPassword:

        type: string

    customRules:

      - expr: get('password') == get('confirmPassword')

        message: "Passwords do not match"

      - expr: get('password').length >= 8

        message: "Password must be at least 8 characters"

      - expr: get('email').includes('@')

        message: "Email must contain @ symbol"

  chat:

    model: llama3.2:1b

    prompt: "Process user: {{ get('email') }}"

```



</div>

### Example: Complete Validation



<div v-pre>



```yaml

run:

  validation:

    required:

      - email

      - name

    properties:

      email:

        type: string

        pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

      name:

        type: string

        minLength: 1

        maxLength: 100

      age:

        type: integer

        minimum: 18

        maximum: 120

      role:

        type: string

        enum: [user, admin, moderator]

      tags:

        type: array

        minItems: 1

        maxItems: 10

    customRules:

      - expr: get('password') == get('confirmPassword')

        message: "Passwords must match"

      - expr: get('age') >= 18

        message: "Must be 18 or older"

  chat:

    model: llama3.2:1b

    prompt: "Process user: {{ get('name') }}"

```



</div>

## Allowed Headers and Parameters

Restrict which headers and query parameters are allowed in requests.

### Allowed Headers



<div v-pre>



```yaml

run:

  allowedHeaders:

    - Authorization

    - Content-Type

    - X-API-Key

  chat:

    model: llama3.2:1b

    prompt: "{{ get('q') }}"

```



</div>

If a request contains headers not in this list, the resource is skipped.

### Allowed Parameters



<div v-pre>



```yaml

run:

  allowedParams:

    - q

    - userId

    - action

  chat:

    model: llama3.2:1b

    prompt: "{{ get('q') }}"

```



</div>

If a request contains query parameters not in this list, the resource is skipped.

### Combined Example

<div v-pre>

```yaml
run:
  restrictToHttpMethods: [POST]
  restrictToRoutes: ["/api/v1/secure"]
  allowedHeaders:
    - Authorization
    - Content-Type
  allowedParams:
    - action
  chat:
    model: llama3.2:1b
    prompt: "Secure action: {{ get('action') }}"
```

</div>

## Execution Order

Resources are checked in this order:

1. **Route/Method Restrictions** - Skip if doesn't match
2. **Skip Conditions** - Skip if condition is true
3. **Preflight Checks** - Error if validation fails
4. **Input Validation** - Error if structure invalid
5. **Execute Resource** - Run the action

```
Request
    ↓
┌─────────────────────┐
│ Route/Method Check │ → Skip if not matching
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│ Skip Conditions     │ → Skip if condition true
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│ Preflight Check     │ → Error if validation fails
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│ Input Validation    │ → Error if structure invalid
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│ Execute Resource    │
└─────────────────────┘
```

## Best Practices

### 1. Use Skip Conditions for Optional Logic

```yaml
# Good: Skip optional processing
skipCondition:
  - get('enableCache') != true
run:
  python:
    script: |
      # Expensive caching operation
```

### 2. Validate Early with Preflight Checks

<div v-pre>

```yaml
# Good: Catch errors before expensive operations
preflightCheck:
  validations:
    - get('userId') != null
    - get('apiKey') != ''
run:
  httpClient:
    url: "https://api.example.com/users/{{ get('userId') }}"
```

</div>

### 3. Restrict Routes for Security

<div v-pre>

```yaml
# Good: Limit resource to specific endpoints
restrictToRoutes: ["/api/v1/admin"]
restrictToHttpMethods: [POST]
run:
  chat:
    prompt: "Admin action: {{ get('action') }}"
```

</div>

### 4. Combine Multiple Controls

<div v-pre>

```yaml
run:
  # Security: Only POST to admin routes
  restrictToHttpMethods: [POST]
  restrictToRoutes: ["/api/v1/admin"]
  allowedHeaders: [Authorization]
  
  # Validation: Check inputs
  preflightCheck:
    validations:
      - get('adminToken') != null
    error:
      code: 401
      message: Admin token required
  
  # Logic: Skip if dry-run
  skipCondition:
    - get('dryRun') == true
  
  # Execute
  chat:
    model: llama3.2:1b
    prompt: "Admin: {{ get('action') }}"
```

</div>

## Error Handling

### Preflight Errors

Preflight errors return the custom error you define:

```yaml
preflightCheck:
  validations:
    - get('q') != ''
  error:
    code: 400
    message: Query parameter 'q' is required
```

Response:
```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Query parameter 'q' is required"
  }
}
```

### Validation Errors

Input validation errors return structured error information:

```json
{
  "success": false,
  "error": {
    "code": 422,
    "message": "Validation failed",
    "details": {
      "email": "Invalid email format",
      "age": "Must be between 18 and 120"
    }
  }
}
```

## Examples

### Example 1: Conditional Processing

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: smartProcessor
  name: Smart Processor

run:
  # Skip if not needed
  skipCondition:
    - get('process') != true
  
  # Validate inputs
  preflightCheck:
    validations:
      - get('data') != null
      - len(get('data')) > 0
    error:
      code: 400
      message: Data is required
  
  # Process
  python:
    script: |
      data = get('data')
      return process(data)
```

### Example 2: Secure Endpoint

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: secureEndpoint
  name: Secure Endpoint

run:
  # Security restrictions
  restrictToHttpMethods: [POST]
  restrictToRoutes: ["/api/v1/secure"]
  allowedHeaders: [Authorization, Content-Type]
  
  # Validate authentication
  preflightCheck:
    validations:
      - get('Authorization') != null
      - get('Authorization').startsWith('Bearer ')
    error:
      code: 401
      message: Valid authorization token required
  
  # Process
  chat:
    model: llama3.2:1b
    prompt: "Secure: {{ get('q') }}"
```

</div>

## Related Documentation

- [Expressions](expressions.md) - Expression syntax for conditions
- [Resources Overview](../resources/overview.md) - Resource structure
- [Unified API](unified-api.md) - Using `get()` in validations
- [Workflow Configuration](../configuration/workflow.md) - Route configuration
