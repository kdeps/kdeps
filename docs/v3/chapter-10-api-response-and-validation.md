# Chapter 10: API Response and Validation

Every workflow needs two things to be production-ready: a way to return structured responses to callers, and a way to guard against bad inputs. The `apiResponse:` resource and the `validations:` block are how kdeps does both.

## The API Response Resource

`apiResponse:` is the terminal resource in every workflow. It builds the HTTP response that goes back to the caller. There must be exactly one resource with `apiResponse:` that is reachable from `targetActionId`.

### Basic Structure

```yaml
# resources/respond.yaml
actionId: respond
requires: [lastStep]
apiResponse:
  success: true
  response:
    result: get('lastStep')
```

This returns:

```json
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "response": {
    "result": "..."
  }
}
```

### Full Configuration

```yaml
apiResponse:
  success: true             # boolean; included in response body
  statusCode: 200           # HTTP status code; defaults to 200
  headers:                  # additional response headers
    X-Request-ID: "&#123;&#123; info('ID') &#125;&#125;"
    X-Workflow-Version: "1.0.0"
  response:                 # response body; can be any shape
    key: value
    nested:
      data: get('someResource')
      count: len(get('results'))
```

### Dynamic Response Bodies

The `response:` block supports any YAML structure. Values can be:

- Literal values: `"static string"`, `42`, `true`
- Expression calls: `get('resource')`, `len(get('list'))`, `get('resource').field`
- Computed values set earlier via `set()`

```yaml
apiResponse:
  success: true
  response:
    # Direct resource output
    answer: get('llm')
    
    # Nested structure
    metadata:
      request_id: info('ID')
      processing_time_ms: get('timing')
      model: "llama3.2:1b"
    
    # Arrays
    results: get('searchResults')
    top_result: get('searchResults')[0]
    
    # Computed
    word_count: len(split(get('llm'), ' '))
    truncated: get('llm')[0:500]
```

### Error Responses

For controlled error cases (validation failures that you handle explicitly), return a non-200 status:

```yaml
# resources/not-found.yaml
actionId: notFound
validations:
  skip:
    - get('record') != null    # skip this resource if record was found
apiResponse:
  success: false
  statusCode: 404
  response:
    error: "Record not found"
    id: get('requestedId')
```

### Choosing the Terminal Resource

`targetActionId` in `workflow.yaml` points to the `apiResponse:` resource. Resources not in the dependency path of `targetActionId` are not executed on a given request, even if they exist in `resources/`.

This is how you structure multi-route workflows: each route has its own terminal resource, and the resources filter themselves with `validations.routes`. `targetActionId` can point to one that handles all routes, or you can use a "gateway" resource that requires all possible terminal resources and lets validation determine which one actually runs.

## Validations

`validations:` is a first-class feature, not an afterthought. It runs before any resource action executes. When validation fails, the pipeline stops immediately — no downstream resources run, no LLM tokens are spent.

### Where Validations Live

Every resource can have a `validations:` block. Validations are evaluated for each resource independently before that resource's action runs:

```yaml
# resources/process.yaml
actionId: process
validations:
  methods: [POST]                           # only fires on POST
  routes: [/api/v1/process]                 # only fires on this path
  check:
    - get('text') != ''                     # text must be present
    - len(get('text')) <= 10000             # text must not be too long
    - get('format') in ['json', 'text', 'markdown']   # format must be valid
  skip:
    - get('result') != ''                   # skip if already computed
  error:
    code: 422
    message: "validation failed"
chat:
  # ...
```

### Validation Execution Order

Validations run in a fixed sequence. Later stages never run if an earlier stage skips or fails:

```
1. headers / params   — filter: resource skipped silently if required header/param absent
2. skip               — OR logic: skipped silently if any expression is true
3. methods / routes   — filter: skipped silently if no match
4. check + error      — AND logic: aborted with error if any expression is false
5. required / rules   — schema: aborted with 422 if field is missing or invalid type
6. Execute resource action
```

### methods and routes

`methods:` and `routes:` act as filters. If the incoming request does not match, the resource is silently skipped — not an error, just not executed:

```yaml
validations:
  methods: [GET]         # only activates on GET requests
  routes: [/api/v1/read] # only activates on this route
```

This is how you design workflows that handle multiple routes. Each resource declares which routes and methods it participates in. For a given request, only the matching resources execute.

### check

`check:` is a list of boolean expressions. **All** must be true for the resource to proceed:

```yaml
validations:
  check:
    - get('email') != ''
    - get('email') matches '^[^@]+@[^@]+\\.[^@]+$'
    - get('age') >= 18
    - get('country') in ['NL', 'DE', 'BE', 'FR']
```

If any expression is false, execution stops and the `error:` block is returned to the caller.

### skip

`skip:` is a list of boolean expressions. If **any** is true, the resource is silently skipped:

```yaml
validations:
  skip:
    - get('cached') != ''          # already have a result, skip expensive computation
    - get('type') != 'premium'     # only process premium content
```

`skip` is for conditional execution, not for errors. When a resource is skipped, execution continues through the DAG. Any downstream resources that depended on this one also skip (since the dependency was never satisfied).

### error

`error:` defines what to return when a `check:` expression fails:

```yaml
validations:
  check:
    - get('q') != ''
  error:
    code: 400
    message: "the 'q' field is required"
```

The response body when this fires:

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "the 'q' field is required"
  }
}
```

You can include dynamic values in the error message:

```yaml
error:
  code: 422
  message: "Invalid format '&#123;&#123; get('format') &#125;&#125;'. Allowed: json, text, markdown."
```

### Validation Placement Strategy

**Early validation** — put broad input validation on the first resource in the request path. This catches malformed requests before any downstream resources run.

**Resource-local validation** — put domain-specific checks on the resource that uses the data. A SQL resource that looks up a user can validate the user exists before running the query.

**Skip for caching** — put cache-check skips on expensive resources (LLM calls, browser automation) so they do not re-execute when results are already available.

### headers and params

`headers:` skips the resource if the specified request headers are absent. `params:` skips if the specified query parameters are absent. Both are silent filters — no error, just skip:

```yaml
validations:
  headers: [Authorization]        # skip if Authorization header is missing
  params: [q, limit]              # skip if ?q or ?limit are missing from query string
```

Use these to scope resources to specific callers or request shapes without writing check expressions.

### Input Schema Validation

`required:`, `rules:`, and `properties:` provide a declarative schema system. This is more expressive than `check:` expressions for field-level validation — it produces specific per-field error messages and handles type coercion automatically:

```yaml
validations:
  required: [username, email, age]    # these fields must be present and non-empty
  
  rules:                              # typed field validation
    - field: email
      type: email                     # RFC-compliant email format
    - field: username
      type: string
      minLength: 3
      maxLength: 50
      pattern: "^[a-zA-Z0-9_]+$"     # alphanumeric + underscore only
    - field: age
      type: integer
      min: 18
      max: 120
    - field: role
      type: string
      enum: [admin, editor, viewer]   # must be one of these values
    - field: website
      type: url                        # must be http:// or https://
    - field: user_id
      type: uuid                       # standard UUID format
    - field: tags
      type: array
      minItems: 1
      maxItems: 10
  
  expr:                              # custom expression rules (AND logic)
    - "get('password') == get('confirmPassword')"
    - "len(get('password')) >= 8"
```

**Supported field types:**

| Type | Validates |
|---|---|
| `string` | Any string; `minLength`, `maxLength`, `pattern` (regex), `enum` |
| `integer` | Whole number; `min`/`minimum`, `max`/`maximum` |
| `number` | Decimal; `min`/`minimum`, `max`/`maximum` |
| `boolean` | `true` or `false` |
| `array` | List; `minItems`, `maxItems` |
| `object` | Key-value map |
| `email` | RFC-compliant email address |
| `url` | Must start with `http://` or `https://` |
| `uuid` | Standard UUID format |
| `date` | RFC3339 or YYYY-MM-DD format |

When schema validation fails, kdeps returns a 422 with a field-specific error message identifying which field failed and why. The error is automatic — you do not need to write a custom `error:` block for schema failures.

**Alternative syntax — `properties:`** (map format, equivalent to `rules:`):

```yaml
validations:
  required: [email, name]
  properties:
    email:
      type: email
    name:
      type: string
      minLength: 1
      maxLength: 100
    age:
      type: integer
      min: 0
```

Use `rules:` when order matters (checked top-to-bottom). Use `properties:` when the map format reads more clearly for your team.

### Multi-Field Validation Example

```yaml
# resources/validate-order.yaml
actionId: validateOrder
validations:
  methods: [POST]
  routes: [/api/v1/orders]
  check:
    - get('customer_id') != ''
    - get('items') != null
    - len(get('items')) > 0
    - get('total') > 0
    - get('currency') in ['USD', 'EUR', 'GBP']
    - get('shipping_address') != null
    - get('shipping_address').country != ''
  error:
    code: 422
    message: "order validation failed: check customer_id, items, total, currency, and shipping_address"
exec:
  command: "echo 'validated'"
```

This is a validation gateway — an `exec:` resource that just echoes a dummy command, used purely for its `validations:` side effect. All subsequent resources in the order pipeline `require: [validateOrder]`.

## The onError Block

Resources can also define conditional error handling with `onError:`:

```yaml
# resources/fetch.yaml
actionId: fetch
httpClient:
  url: "&#123;&#123; get('url') &#125;&#125;"
  
onError:
  when:
    - get('fetch').statusCode == 404
  response:
    code: 404
    message: "The requested resource was not found at &#123;&#123; get('url') &#125;&#125;"
```

`onError.when` is evaluated after execution. If any expression is true, the workflow stops and the `onError.response` is returned. This is for handling errors that only become apparent after the resource runs (like an upstream API returning 404).

## Practical: A Complete API with Proper Validation

Putting it all together — a user lookup API with full validation:

```yaml
# workflow.yaml
metadata:
  name: user-lookup
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServer:
    routes:
      - path: /api/v1/users
        methods: [GET]
```

```yaml
# resources/validate.yaml
actionId: validate
validations:
  methods: [GET]
  routes: [/api/v1/users]
  check:
    - get('id') != ''
  error:
    code: 400
    message: "query parameter 'id' is required"
exec:
  command: "echo 'ok'"
```

```yaml
# resources/lookup.yaml
actionId: lookup
requires: [validate]
sql:
  connectionName: main
  query: "SELECT id, name, email, created_at FROM users WHERE id = $1"
  params:
    - get('id')
```

```yaml
# resources/not-found-check.yaml
actionId: notFoundCheck
requires: [lookup]
validations:
  check:
    - len(get('lookup')) > 0
  error:
    code: 404
    message: "user not found"
exec:
  command: "echo 'found'"
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [notFoundCheck]
apiResponse:
  success: true
  response:
    user: get('lookup')[0]
```

This workflow validates that `id` is provided, looks up the user, checks whether the lookup returned a row, and returns the user — or appropriate errors at each step. The logic is explicit, testable, and independently deployable.

X> ## Exercise
X>
X> Harden the chatbot from Chapter 2 by adding a complete validation layer. The endpoint accepts `POST /api/v1/chat` with a JSON body containing `q` (the question) and optionally `lang` (a two-letter ISO language code like `"en"` or `"nl"`).
X>
X> Add these validation rules and confirm each one triggers with a curl test:
X>
X> 1. `q` must be present and non-empty → `400 "question is required"`
X> 2. `q` must be under 500 characters → `400 "question too long"`
X> 3. If `lang` is provided, it must be exactly two lowercase letters → `400 "lang must be a 2-letter ISO code"`
X> 4. If `lang` is not provided, default it to `"en"` in a `before:` expression
X>
X> After validation passes, include `lang` in the response body alongside the LLM answer so the caller can confirm which language was used.
X>
X> Test all four cases before moving on. Then test that `kdeps validate workflow.yaml` passes cleanly.
X>
X> **Stretch goal:** Add a fifth validation: if `q` contains the substring `"ignore previous instructions"`, reject with `400 "invalid input"`. Discuss in a code comment why this is a shallow defence and what a better approach would be.
