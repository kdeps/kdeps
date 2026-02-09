# HTTP Client Resource

The HTTP Client resource enables making external API calls with support for authentication, retries, caching, and advanced TLS configuration.

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: httpResource
  name: API Call

run:
  httpClient:
    method: GET
    url: "https://api.example.com/data"
    timeoutDuration: 30s
```

</div>

## Configuration Options

### Complete Reference

```yaml
run:
  httpClient:
    # Request Configuration
    method: GET                      # GET, POST, PUT, PATCH, DELETE
    url: <span v-pre>"https://api.example.com/{{ get('id') }}"</span>
    headers:
      Authorization: <span v-pre>"Bearer {{ get('token') }}"</span>
      Content-Type: application/json
    data:                            # Request body
      key: value
    timeoutDuration: 30s

    # Authentication
    auth:
      type: bearer                   # basic, bearer, api_key, oauth2
      token: <span v-pre>"{{ get('api_token') }}"</span>

    # Retry Configuration
    retry:
      maxAttempts: 3
      backoff: 1s
      maxBackoff: 30s
      retryOn: [500, 502, 503, 504]

    # Caching
    cache:
      enabled: true
      ttl: 5m
      key: "custom-cache-key"

    # Advanced Options
    followRedirects: true
    proxy: "http://proxy:16395"
    tls:
      insecureSkipVerify: false
      certFile: "/path/to/cert.pem"
      keyFile: "/path/to/key.pem"
      caFile: "/path/to/ca.pem"
```

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

### Bearer Token

<div v-pre>

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/protected"
  auth:
    type: bearer
    token: "{{ get('api_token') }}"
```

</div>

### Basic Auth

<div v-pre>

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/protected"
  auth:
    type: basic
    username: "{{ get('username') }}"
    password: "{{ get('password') }}"
```

</div>

### API Key

<div v-pre>

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/data"
  auth:
    type: api_key
    key: X-API-Key           # Header name
    value: "{{ get('api_key') }}"
```

</div>

### OAuth2

<div v-pre>

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/oauth/data"
  auth:
    type: oauth2
    token: "{{ get('access_token') }}"
```

</div>

## Retry Configuration

Automatic retries with exponential backoff:

```yaml
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

Cache responses to reduce API calls:

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/static-data"
  cache:
    enabled: true
    ttl: 5m                  # Cache for 5 minutes
    key: "static-data-cache" # Optional: custom cache key
```

Cache key defaults to the URL if not specified.

## TLS Configuration

### Skip Certificate Verification (Development Only)

```yaml
httpClient:
  method: GET
  url: "https://self-signed.example.com"
  tls:
    insecureSkipVerify: true
```

> **Warning**: Never use `insecureSkipVerify: true` in production.

### Custom Certificates

```yaml
httpClient:
  method: GET
  url: "https://internal.example.com"
  tls:
    certFile: "/certs/client.pem"
    keyFile: "/certs/client-key.pem"
    caFile: "/certs/ca.pem"
```

## Proxy Configuration

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/data"
  proxy: "http://proxy.internal:16395"
```

## Redirect Handling

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/redirect"
  followRedirects: true      # Default: true
```

Set to `false` to prevent following redirects.

## Examples

### Fetch Data and Process

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: fetchData

run:
  httpClient:
    method: GET
    url: "https://api.github.com/repos/{{ get('owner') }}/{{ get('repo') }}"
    headers:
      Accept: application/vnd.github.v3+json
      Authorization: "Bearer {{ get('GITHUB_TOKEN', 'env') }}"
    timeoutDuration: 30s

---
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: analyzeRepo
  requires: [fetchData]

run:
  chat:
    model: llama3.2:1b
    prompt: "Analyze this GitHub repo: {{ get('fetchData') }}"
```

</div>

### Authenticated API with Retry

<div v-pre>

```yaml
run:
  httpClient:
    method: POST
    url: "https://api.stripe.com/v1/charges"
    auth:
      type: bearer
      token: "{{ get('STRIPE_SECRET_KEY', 'env') }}"
    headers:
      Content-Type: application/x-www-form-urlencoded
    data:
      amount: "{{ get('amount') }}"
      currency: usd
      source: "{{ get('token') }}"
    retry:
      maxAttempts: 3
      backoff: 2s
      retryOn: [500, 502, 503]
    timeoutDuration: 30s
```

</div>

### Webhook Call

<div v-pre>

```yaml
run:
  httpClient:
    method: POST
    url: "{{ get('webhook_url') }}"
    headers:
      Content-Type: application/json
      X-Webhook-Secret: "{{ get('WEBHOOK_SECRET', 'env') }}"
    data:
      event: order_completed
      order_id: "{{ get('order_id') }}"
      timestamp: "{{ info('timestamp') }}"
    timeoutDuration: 10s
```

</div>

### Cached External API

```yaml
run:
  httpClient:
    method: GET
    url: "https://api.exchangerate.host/latest"
    cache:
      enabled: true
      ttl: 1h
      key: "exchange-rates"
    timeoutDuration: 30s
```

## Accessing Response

The HTTP client response includes:

```yaml
# In another resource
metadata:
  requires: [httpResource]

run:
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
run:
  expr:
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
run:
  preflightCheck:
    validations:
      - get('api_token') != ''
      - get('user_id') != ''
    error:
      code: 400
      message: "API token and user ID are required"

  httpClient:
    method: GET
    url: "https://api.example.com/users/{{ get('user_id') }}"
    auth:
      type: bearer
      token: "{{ get('api_token') }}"
```

</div>

## Best Practices

1. **Always set timeouts** - Prevent hanging requests
2. **Use retries for external APIs** - Handle transient failures
3. **Cache static data** - Reduce API calls and latency
4. **Store secrets in environment** - Never hardcode API keys
5. **Use appropriate authentication** - Match the API's requirements
6. **Validate inputs** - Check required parameters before calling

## Next Steps

- [SQL Resource](sql.md) - Database operations
- [LLM Resource](llm.md) - AI model integration
- [Unified API](../concepts/unified-api.md) - Data access patterns
