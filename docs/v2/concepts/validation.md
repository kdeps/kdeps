# Validation and Control Flow

KDeps provides powerful validation and control flow features to handle request filtering, input validation, and conditional execution.

## Request Filtering

### Allowed Headers

Restrict which HTTP headers are accepted by a resource:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: secureEndpoint
run:
  allowedHeaders:
    - Authorization
    - Content-Type
    - X-Request-ID
    - X-API-Key

  chat:
    prompt: "{{ get('q') }}"
```

</div>

When `allowedHeaders` is set:
- Only listed headers are accessible via `get('HeaderName')`
- Other headers are ignored/filtered
- Useful for security and documentation

### Allowed Parameters

Restrict which query/body parameters are accepted:

<div v-pre>

```yaml
run:
  allowedParams:
    - q
    - limit
    - offset
    - sort
    - filter

  sql:
    queries:
      - query: "SELECT * FROM items WHERE name LIKE ? LIMIT ? OFFSET ?"
        params:
          - "%{{ get('q') }}%"
          - "{{ default(get('limit'), 10) }}"
          - "{{ default(get('offset'), 0) }}"
```

</div>

Benefits:
- Prevents unexpected parameters from being processed
- Documents expected inputs
- Protects against parameter pollution attacks

### Combined Example

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: searchEndpoint
run:
  # Only accept these headers
  allowedHeaders:
    - Authorization
    - Content-Type

  # Only accept these parameters
  allowedParams:
    - query
    - page
    - pageSize

  expr:
    - set('auth', get('Authorization'))
    - set('searchQuery', get('query'))
    - set('pageNum', default(get('page'), 1))
    - set('size', default(get('pageSize'), 20))

  httpClient:
    url: "https://api.example.com/search"
    method: GET
    headers:
      Authorization: "{{ get('auth') }}"
    params:
      q: "{{ get('searchQuery') }}"
      page: "{{ get('pageNum') }}"
      limit: "{{ get('size') }}"
```

</div>

## Preflight Checks

Preflight checks run **before** the primary action (LLM, HTTP, SQL, etc.) of a resource. They are used to validate inputs and state, and can return custom error responses.

<div v-pre>

```yaml
run:
  preflightCheck:
    validations:
      - get('q') != ''
      - len(get('q')) > 3
      - get('Authorization', 'header') != ''
    error:
      code: 400
      message: "Query 'q' must be at least 4 characters, and Authorization is required."

  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
```

</div>

### Preflight Properties

| Property | Type | Description |
|----------|------|-------------|
| `validations` | array | List of expressions that must all evaluate to `true`. |
| `error` | object | Custom error to return if any validation fails. |

### Error Properties

| Property | Type | Description |
|----------|------|-------------|
| `code` | integer | The HTTP status code or application error code. |
| `message` | string | The error message returned to the user. Supports interpolation. |

## Input Validation

### Using the validation Block

Define validation rules for incoming data:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: createUser
run:
  validation:
    required:
      - username
      - email
      - password
    properties:
      username:
        type: string
        minLength: 3
        maxLength: 50
        pattern: "^[a-zA-Z0-9_]+$"
      email:
        type: string
        format: email
      password:
        type: string
        minLength: 8
              age:
                type: integer
                minimum: 0
                maximum: 150
      ```
      
      <div v-pre>
      
      ```yaml
        sql:
          queries:
            - query: "INSERT INTO users (username, email, password, age) VALUES (?, ?, ?, ?)"
              params:
                - "{{ get('username') }}"
                - "{{ get('email') }}"
                - "{{ get('password') }}"
                - "{{ get('age') }}"
      ```
      
      </div>
      
      ### Validation Properties
| Property | Type | Description |
|----------|------|-------------|
| `required` | array | List of required fields |
| `properties` | object | Field-specific validation rules |

### Supported Types

| Type | Description | Validation |
|------|-------------|------------|
| `string` | Text values | minLength, maxLength, pattern, enum |
| `integer` | Whole numbers | min/minimum, max/maximum |
| `number` | Decimal numbers | min/minimum, max/maximum |
| `boolean` | true/false | Type check only |
| `array` | Lists | minItems, maxItems |
| `object` | Key-value maps | Type check only |
| `email` | Email addresses | RFC-compliant email format |
| `url` | HTTP/HTTPS URLs | Must start with http:// or https:// |
| `uuid` | UUID strings | Standard UUID format |
| `date` | Date strings | RFC3339 or YYYY-MM-DD format |

### Property Rules

| Rule | Description | Example |
|------|-------------|---------|
| `type` | Data type | `string`, `integer`, `number`, `email`, etc. |
| `minLength` | Minimum string length | `minLength: 3` |
| `maxLength` | Maximum string length | `maxLength: 100` |
| `min` / `minimum` | Minimum number value | `min: 0` or `minimum: 0` |
| `max` / `maximum` | Maximum number value | `max: 100` or `maximum: 100` |
| `pattern` | Regex pattern | `pattern: "^[a-z]+$"` |
| `enum` | Allowed values | `enum: ["a", "b", "c"]` |
| `minItems` | Minimum array length | `minItems: 1` |
| `maxItems` | Maximum array length | `maxItems: 10` |
| `message` | Custom error message | `message: "Invalid value"` |

### Alternative Syntax: Rules Array

You can also use the `rules` array format instead of `properties`:

```yaml
run:
  validation:
    required:
      - username
      - email
    rules:
      - field: username
        type: string
        minLength: 3
        maxLength: 50
        pattern: "^[a-zA-Z0-9_]+$"
      - field: email
        type: email
      - field: age
        type: integer
        min: 0
        max: 150
```

Both formats are equivalent and can be used interchangeably based on preference.

### Custom Validation Rules

Add custom validation logic with expressions:

```yaml
run:
  validation:
    required:
      - password
      - confirmPassword
    customRules:
      - expr: get('password') == get('confirmPassword')
        message: "Passwords must match"
      - expr: len(get('password')) >= 8
        message: "Password must be at least 8 characters"
      - expr: get('age') >= 18 || get('parentConsent') == true
        message: "Must be 18 or older, or have parent consent"
```

### Complex Validation Example

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: orderValidation
run:
  validation:
    required:
      - items
      - shippingAddress
      - paymentMethod
    properties:
      items:
        type: array
        minItems: 1
        maxItems: 100
      shippingAddress:
        type: object
      paymentMethod:
        type: string
        enum:
          - credit_card
          - debit_card
          - paypal
          - bank_transfer
      couponCode:
        type: string
        pattern: "^[A-Z0-9]{6,10}$"
    customRules:
      - expr: |
          get('items') != nil &&
          all(get('items'), .quantity > 0 && .quantity <= 99)
        message: "Item quantities must be between 1 and 99"
      - expr: |
          get('shippingAddress').country != nil &&
          get('shippingAddress').postalCode != nil
        message: "Shipping address must include country and postal code"
```

## Control Flow

### Skip Conditions

Skip resource execution based on conditions:

<div v-pre>

```yaml
run:
  skipCondition:
    - get('skipProcessing') == true
    - get('status') == 'completed'

  chat:
    prompt: "Process this"
```

</div>

Multiple conditions use OR logic - resource is skipped if ANY condition is true.

### Skip with Expression

<div v-pre>

```yaml
run:
  # Skip if no query provided
  skipCondition:
    - get('q') == nil || get('q') == ''

  # Skip if user is not authenticated
  skipCondition:
    - get('Authorization') == nil

  # Skip if already cached
  skipCondition:
    - get('cachedResult', 'session') != nil
```

</div>

### Conditional Resource Selection

Use skip conditions to implement routing:

<div v-pre>

```yaml
# Resource: Handle text queries
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: handleText
run:
  skipCondition:
    - get('type') != 'text'
  chat:
    prompt: "{{ get('q') }}"

---
# Resource: Handle image queries
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: handleImage
run:
  skipCondition:
    - get('type') != 'image'
  chat:
    model: llama3.2-vision
    prompt: "Analyze this image"
    files:
      - "{{ get('imagePath') }}"

---
# Resource: Router
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: router
  requires:
    - handleText
    - handleImage
run:
  apiResponse:
    response:
      result: "{{ get('type') == 'text' ? get('handleText') : get('handleImage') }}"
```

</div>

### Early Exit Pattern

Stop processing on validation failure:

<div v-pre>

```yaml
# Validate first
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: validateRequest
run:
  expr:
    - set('isValid', get('apiKey') != nil && len(get('apiKey')) == 32)
    - set('errorMessage', get('isValid') ? nil : 'Invalid API key')

---
# Process only if valid
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: processRequest
  requires:
    - validateRequest
run:
  skipCondition:
    - get('validateRequest').isValid == false
  chat:
    prompt: "{{ get('q') }}"

---
# Return appropriate response
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: response
  requires:
    - validateRequest
    - processRequest
run:
  apiResponse:
    success: get('validateRequest').isValid
    response:
      data: "{{ get('validateRequest').isValid ? get('processRequest') : nil }}"
      error: "{{ get('validateRequest').errorMessage }}"
```

</div>

## Error Handling

### Try-Catch Pattern

Handle errors gracefully:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: safeApiCall
run:
  expr:
    - set('hasError', false)
    - set('errorMessage', nil)

  httpClient:
    url: "https://api.example.com/data"
    onError:
      - set('hasError', true)
      - set('errorMessage', error.message)

---
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: handleResult
  requires:
    - safeApiCall
run:
  expr:
    - set('result', get('safeApiCall').hasError ?
        {"error": get('safeApiCall').errorMessage} :
        {"data": get('safeApiCall')})
  apiResponse:
    success: "!get('safeApiCall').hasError"
    response:
      result: get('result')
```

### Fallback Values

Use `default()` for fallback handling:

<div v-pre>

```yaml
run:
  expr:
    # Primary data source with fallback
    - set('userData', default(
        get('primaryDb'),
        default(
          get('cacheDb'),
          {"name": "Guest", "role": "anonymous"}
        )
      ))

    # Configuration with defaults
    - set('config', {
        "timeout": default(get('TIMEOUT', 'env'), 30),
        "retries": default(get('RETRIES', 'env'), 3),
        "debug": default(get('DEBUG', 'env'), false)
      })
```

</div>

## Best Practices

1. **Validate Early**: Put validation at the start of your workflow
2. **Fail Fast**: Use skip conditions to avoid unnecessary processing
3. **Clear Error Messages**: Provide helpful messages in customRules
4. **Whitelist, Don't Blacklist**: Use allowedHeaders/allowedParams to whitelist acceptable inputs
5. **Layer Validations**: Combine schema validation with custom rules for comprehensive checks

## See Also

- [Resources Overview](../resources/overview) - Resource configuration
- [Error Handling](error-handling) - onError with retries and fallbacks
- [Expressions](expressions) - Expression syntax
- [Expression Helpers](expression-helpers) - Helper functions like default()
