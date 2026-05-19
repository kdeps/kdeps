# Validation

All validation, filtering, and control-flow for a resource lives in a single `run.validations:` block. This replaces the former separate fields (`restrictToHttpMethods`, `restrictToRoutes`, `allowedHeaders`, `allowedParams`, `skipCondition`, `preflightCheck`, `validation`).

## Quick Reference

```yaml
validations:
  # HTTP filtering
  methods: [GET, POST]              # allow only these methods
  routes: [/api/v1/chat]            # allow only these routes
  headers: [Authorization]          # allow only these headers
  params: [q, limit]                # allow only these query params

  # Control flow
  skip:                             # OR logic — any true → skip resource
    - get('q') == nil
  check:                            # AND logic — all must be true
    - get('q') != ''
    - len(get('q')) > 3
  error:                            # returned when check fails
    code: 400
    message: "Query 'q' must be at least 4 characters"

  # Input schema
  required: [username, email]
  rules:
    - field: email
      type: email
    - field: age
      type: integer
      min: 18
      max: 120
  expr:                             # custom expression rules (bare scalar strings)
    - "get('password') == get('confirmPassword')"
```

## HTTP Filtering

### `methods` — Restrict HTTP methods

```yaml
validations:
  methods: [GET, POST]
sql:
  query: "SELECT * FROM users"
```

- Allowed values: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`
- Resource is skipped for non-matching methods
- Omit to allow all methods

### `routes` — Restrict URL routes

```yaml
validations:
  routes: [/api/v1/users, /api/v1/profiles]
chat:
  prompt: "{{ get('q') }}"
```

- Routes must start with `/`
- Resource is skipped for non-matching routes
- Omit to allow all routes

### `headers` — Whitelist accepted headers

```yaml
validations:
  headers:
    - Authorization
    - Content-Type
    - X-API-Key
chat:
  prompt: "{{ get('q') }}"
```

Only listed headers are accessible via `get('HeaderName')`. Other headers are ignored.

### `params` — Whitelist accepted query parameters

```yaml
validations:
  params:
    - q
    - limit
    - offset
sql:
  query: "SELECT * FROM items LIMIT ? OFFSET ?"
  params:
    - get('limit')
    - get('offset')
```

Only listed parameters are accessible. Protects against parameter pollution.

## Control Flow

### `skip` — Conditional skip (OR logic)

Resource is skipped when **any** condition is `true`:

<div v-pre>

```yaml
validations:
  skip:
    - get('q') == nil || get('q') == ''
    - get('cachedResult', 'session') != nil
chat:
  prompt: "{{ get('q') }}"
```

</div>

### `check` + `error` — Preflight validation (AND logic)

All conditions must be `true`; if any fail the custom error is returned immediately:

<div v-pre>

```yaml
validations:
  check:
    - get('q') != ''
    - get('Authorization', 'header') != ''
  error:
    code: 400
    message: "Query 'q' and Authorization header are required"
chat:
  prompt: "{{ get('q') }}"
```

</div>

`check` runs before the resource action. Use it to short-circuit expensive operations.

## Input Schema Validation

### `required` — Mandatory fields

```yaml
validations:
  required:
    - username
    - email
    - password
```

### `rules` — Field validation rules (array format)

```yaml
validations:
  required: [email, name]
  rules:
    - field: email
      type: email
    - field: name
      type: string
      minLength: 1
      maxLength: 100
    - field: age
      type: integer
      min: 18
      max: 120
      message: "Must be 18 or older"
```

### `properties` / `fields` — Map format (alternative to `rules`)

```yaml
validations:
  required: [email, name]
  properties:
    email:
      type: email
    name:
      type: string
      minLength: 1
    age:
      type: integer
      min: 18
```

`properties` and `fields` are aliases; `properties` takes precedence over `fields`.

### `expr` — Custom expression rules

<div v-pre>

```yaml
validations:
  required: [password, confirmPassword]
  expr:
    - "get('password') == get('confirmPassword')"
    - "len(get('password')) >= 8"
    - "get('age') >= 18 || get('parentConsent') == true"
```

</div>

## Supported Field Types

| Type | Description | Validation options |
|------|-------------|-------------------|
| `string` | Text | `minLength`, `maxLength`, `pattern`, `enum` |
| `integer` | Whole numbers | `min`/`minimum`, `max`/`maximum` |
| `number` | Decimal numbers | `min`/`minimum`, `max`/`maximum` |
| `boolean` | true/false | type check only |
| `array` | Lists | `minItems`, `maxItems` |
| `object` | Key-value maps | type check only |
| `email` | Email addresses | RFC-compliant format |
| `url` | HTTP/HTTPS URLs | Must start with `http://` or `https://` |
| `uuid` | UUID strings | Standard UUID format |
| `date` | Date strings | RFC3339 or YYYY-MM-DD |

## Execution Order

Validations run in this order:

```
Request
  ↓ headers / params filter    → filter accessible keys
  ↓ skip conditions            → skip if any true
  ↓ methods / routes check     → skip if no match
  ↓ check + error              → abort with error if any false
  ↓ required / rules / expr    → abort with 422 if invalid
  ↓ Execute resource
```

## Complete Example

<div v-pre>

```yaml
actionId: createUser
validations:
  methods: [POST]
  routes: [/api/v1/users]
  headers: [Authorization, Content-Type]
  check:
    - get('Authorization', 'header') != ''
  error:
    code: 401
    message: "Authorization required"
  required: [username, email, password]
  rules:
    - field: username
      type: string
      minLength: 3
      maxLength: 50
      pattern: "^[a-zA-Z0-9_]+$"
    - field: email
      type: email
    - field: password
      type: string
      minLength: 8
  expr:
    - "get('password') == get('confirmPassword')"

sql:
  query: "INSERT INTO users (username, email, password) VALUES (?, ?, ?)"
  params:
    - "{{ get('username') }}"
    - "{{ get('email') }}"
    - "{{ get('password') }}"
```

</div>

## Best Practices

1. **Validate early** — put `check` before expensive LLM or DB operations
2. **Use `skip` for optional resources** — cleaner than error responses for conditional logic
3. **Whitelist with `headers`/`params`** — explicitly list what you accept
4. **Combine `check` and `rules`** — `check` for coarse guards, `rules` for field-level schema

## See Also

- [Resources Overview](../resources/overview) — Resource configuration
- [Expressions](/advanced/expressions) — Expression syntax
- [Expression Helpers](/concepts/expression-helpers) — Helper functions
- [Error Handling](/concepts/error-handling) — `onError` with retries and fallbacks
