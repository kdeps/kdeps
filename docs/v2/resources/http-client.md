# HTTP Client Resource

The `httpClient:` resource makes an outbound HTTP request and stores the parsed response body as its output. JSON responses are parsed automatically; other content types are stored as a string.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Global Named Connections

Authentication credentials and proxy settings belong in `~/.kdeps/config.yaml` as named connections, not inline in resource files or `workflow.yaml`. Resources reference them by name.

```yaml
# ~/.kdeps/config.yaml
http_connections:
  stripe:
    auth:
      type: bearer
      token: "${STRIPE_SECRET_KEY}"
  internal-api:
    auth:
      type: basic
      username: "${API_USER}"
      password: "${API_PASS}"
  corporate-proxy:
    proxy: "http://${PROXY_HOST}:${PROXY_PORT}"
  github:
    auth:
      type: api_key
      key: Authorization       # header name
      value: "token ${GITHUB_TOKEN}"
```

Reference a connection in a resource:

```yaml
httpClient:
  method: GET
  url: "https://api.stripe.com/v1/charges"
  connectionName: stripe   # references http_connections.stripe in ~/.kdeps/config.yaml
```

## Complete reference

<div v-pre>

```yaml
# resources/http-client.yaml
httpClient:
  method: GET                    # GET, POST, PUT, PATCH, DELETE
  url: "https://api.example.com/{{ get('id') }}"
  headers:
    Content-Type: application/json
  data:                          # request body -- serialised as JSON
    key: value
  timeout: 30s                   # hard stop -- returns error, does not retry

  connectionName: myapi          # named connection (auth + proxy) from settings.httpConnections

  # Retry on transient failures
  retry:
    maxAttempts: 3               # total attempts including the first
    backoff: 1s                  # initial wait; doubles on each retry
    maxBackoff: 30s              # ceiling on the retry wait
    retryOn: [500, 502, 503, 504]

  # Response caching -- presence of the cache: block enables it
  cache:
    ttl: 5m                      # cache lifetime; key defaults to the URL
    key: "custom-cache-key"

  followRedirects: true          # set false to stop at the first 3xx
  tls:
    insecureSkipVerify: false    # never set true in production
    certFile: "/path/to/cert.pem"
    keyFile: "/path/to/key.pem"
    caFile: "/path/to/ca.pem"
```

</div>

## HTTP Methods

<div v-pre>

```yaml
# GET request
httpClient:
  method: GET
  url: "https://api.example.com/users/{{ get('id') }}"

# POST request with body
httpClient:
  method: POST
  url: "https://api.example.com/users"
  headers:
    Content-Type: application/json
  data:
    name: "{{ get('name') }}"
    email: "{{ get('email') }}"

# PUT request
httpClient:
  method: PUT
  url: "https://api.example.com/users/{{ get('id') }}"
  data:
    name: "{{ get('name') }}"

# DELETE request
httpClient:
  method: DELETE
  url: "https://api.example.com/users/{{ get('id') }}"
```

</div>

## Authentication

All auth types are defined in `settings.httpConnections` and referenced via `connectionName:`.

### Bearer Token

```yaml
# ~/.kdeps/config.yaml
http_connections:
  myapi:
    auth:
      type: bearer
      token: "${API_TOKEN}"
```

```yaml
# resources/fetch.yaml
httpClient:
  method: GET
  url: "https://api.example.com/protected"
  connectionName: myapi
```

### Basic Auth

```yaml
# ~/.kdeps/config.yaml
http_connections:
  legacy:
    auth:
      type: basic
      username: "${API_USER}"
      password: "${API_PASS}"
```

### API Key

```yaml
# ~/.kdeps/config.yaml
http_connections:
  analytics:
    auth:
      type: api_key
      key: X-API-Key        # header name
      value: "${ANALYTICS_KEY}"
```

### OAuth2

```yaml
# ~/.kdeps/config.yaml
http_connections:
  oauth:
    auth:
      type: oauth2
      token: "${OAUTH2_ACCESS_TOKEN}"
```

### Proxy with Auth

```yaml
# ~/.kdeps/config.yaml
http_connections:
  via-proxy:
    proxy: "http://${PROXY_USER}:${PROXY_PASS}@proxy.corp:8080"
    auth:
      type: bearer
      token: "${API_TOKEN}"
```

## Retry Configuration

Automatic retries with exponential backoff:

```yaml
# resources/example.yaml
httpClient:
  method: GET
  url: "https://api.example.com/data"
  retry:
    maxAttempts: 3           # Total attempts (including first)
    backoff: 1s              # Initial delay between retries
    maxBackoff: 30s          # Maximum delay
    retryOn:                 # Status codes to retry on
      - 500
      - 502
      - 503
      - 504
      - 429  # Rate limited
```

Retry timing example:
- Attempt 1: immediate
- Attempt 2: wait 1s
- Attempt 3: wait 2s (exponential backoff)

## Response Caching

Cache responses to reduce API calls. Presence of the `cache:` block enables caching.

```yaml
# resources/example.yaml
httpClient:
  method: GET
  url: "https://api.example.com/static-data"
  cache:
    ttl: 5m                  # Cache for 5 minutes
    key: "static-data-cache" # Optional: custom cache key
```

Cache key defaults to the URL if not specified.

## TLS Configuration

### Skip Certificate Verification (Development Only)

```yaml
# resources/example.yaml
httpClient:
  method: GET
  url: "https://self-signed.example.com"
  tls:
    insecureSkipVerify: true
```

> **Warning**: Never use `insecureSkipVerify: true` in production.

### Custom Certificates

```yaml
# resources/example.yaml
httpClient:
  method: GET
  url: "https://internal.example.com"
  tls:
    certFile: "/certs/client.pem"
    keyFile: "/certs/client-key.pem"
    caFile: "/certs/ca.pem"
```

## Redirect Handling

```yaml
# resources/example.yaml
httpClient:
  method: GET
  url: "https://api.example.com/redirect"
  followRedirects: true      # Default: true
```

Set to `false` to prevent following redirects.

## Accessing Response

The HTTP client response includes:

```yaml
# In another resource
requires: [httpResource]
apiResponse:
  response:
    # Full response body (parsed JSON or raw string)
    data: get('httpResource')

    # If JSON response, access fields directly
    user_name: get('httpResource').name
    user_email: get('httpResource').email

    # Access response details
    status_code: get('httpResource').statusCode
    headers: get('httpResource').headers
```

### Advanced Response Access

Use resource-specific accessors for detailed response information:

```yaml
# resources/example.yaml
after:
  # Get response body only
  - set('response_body', http.responseBody('httpResource'))

  # Get specific header
  - set('content_type', http.responseHeader('httpResource', 'Content-Type'))

  # Check status
  - set('is_success', get('httpResource').statusCode >= 200 && get('httpResource').statusCode < 300)

apiResponse:
  response:
    body: get('response_body')
    content_type: get('content_type')
    success: get('is_success')
```

See [Unified API](../concepts/unified-api.md#resource-specific-accessors) for details.

## Error Handling

Use preflight checks to validate before making requests:

<div v-pre>

```yaml
# resources/example.yaml
validations:
  check:
    - get('user_id') != ''
  error:
    code: 400
    message: "user_id is required"

httpClient:
  method: GET
  url: "https://api.example.com/users/{{ get('user_id') }}"
  connectionName: myapi
```

</div>

## `http_connections` fields (in `~/.kdeps/config.yaml`)

| Field | Type | Description |
|---|---|---|
| `auth.type` | string | `basic`, `bearer`, `api_key`, or `oauth2` |
| `auth.username` | string | Basic auth username |
| `auth.password` | string | Basic auth password |
| `auth.token` | string | Bearer / OAuth2 token |
| `auth.key` | string | API key header name |
| `auth.value` | string | API key header value |
| `proxy` | string | Proxy URL (may include `user:pass@`) |

## See Also

- [HTTP Client Examples](/reference/http-client-examples) - GitHub, Stripe, webhook, cached API examples
- [SQL Resource](sql.md) -- database operations
- [LLM Resource](llm.md) -- AI model integration
- [Unified API](../concepts/unified-api.md) -- data access patterns
