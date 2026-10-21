# http_request component

The `http_request` component makes one outbound HTTP request and returns the response: status code, status line, headers, raw body, and the parsed JSON body under `data` when the response is JSON. It is a kdeps component from the `components` asset set: call it with a `component:` resource, no install step. Its call is the kdeps API's `http_request` tool, which runs the native HTTP executor. A workflow that uses `httpClient:` directly - primary or in a `before:`/`after:` step - is rejected.

It works in both modes: as a DAG step in workflow mode, and inside any workflow the LLM calls as a tool in agent mode.

```d2
direction: right
call: "component: http_request" {shape: oval}
api: "POST /tools/http_request/call\nkdeps API"
conn: "~/.kdeps/config.yaml\nhttp_connections / identity.accounts"
exec: "HTTP executor\nretry, cache, TLS, redirects"
target: "outbound HTTP request"
out: "result map\nstatusCode, headers, body, data" {shape: oval}
call -> api
conn -> api: "connectionName / accountName"
api -> exec
exec -> target
exec -> out
```

## Configuration

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
name: Fetch User
component:
  name: http_request               # from the components asset set; a components/http_request/ of your own wins
  with:
    method: GET                    # GET, POST, PUT, PATCH, DELETE (default GET)
    url: "https://api.example.com/users/{{ get('id') }}"
    headers:
      Accept: application/json
    data:                          # request body, serialized as JSON (POST, PUT, PATCH)
      name: "{{ get('name') }}"
    timeout: 30s                   # hard stop; returns an error, does not retry
    failOnError: false             # true: a 4xx/5xx fails the resource instead of returning a result
```

</div>

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | yes | - | URL to request |
| `method` | string | no | `GET` | `GET`, `POST`, `PUT`, `PATCH`, or `DELETE` |
| `headers` | object | no | - | Request headers as key-value pairs |
| `data` | object | no | - | Request body, serialized as JSON |
| `files` | object | no | - | Multipart form uploads: each key is a form field, each value a file path |
| `timeout` | string | no | executor default | Hard stop for the request, e.g. `30s` |
| `failOnError` | bool | no | `false` | A non-2xx response becomes a resource error instead of a result |
| `saveTo` | string | no | - | Stream a 2xx body into this file; the result reports `savedTo` and `bytes` instead of the body |
| `retry` | object | no | - | `{maxAttempts, backoff, maxBackoff, retryOn}` - retry on the listed status codes |
| `cache` | object | no | - | `{ttl, key}` - presence enables response caching, key defaults to the method and URL |
| `connectionName` | string | no | - | Named connection in `~/.kdeps/config.yaml` supplying auth and proxy |
| `accountName` | string | no | - | Named account under the agent identity; resolved as Basic Auth |
| `followRedirects` | bool | no | follow | `false` stops at the first 3xx |
| `tls` | object | no | - | `{insecureSkipVerify, certFile, keyFile, caFile}` |

## HTTP methods

<div v-pre>

```yaml
# GET
component:
  name: http_request
  with:
    method: GET
    url: "https://api.example.com/users/{{ get('id') }}"

# POST with a JSON body
component:
  name: http_request
  with:
    method: POST
    url: "https://api.example.com/users"
    headers:
      Content-Type: application/json
    data:
      name: "{{ get('name') }}"
      email: "{{ get('email') }}"

# PUT
component:
  name: http_request
  with:
    method: PUT
    url: "https://api.example.com/users/{{ get('id') }}"
    data:
      name: "{{ get('name') }}"

# DELETE
component:
  name: http_request
  with:
    method: DELETE
    url: "https://api.example.com/users/{{ get('id') }}"
```

</div>

## Authentication

Credentials belong in `~/.kdeps/config.yaml` as named connections, not inline in resource files or `workflow.yaml`. The component references them with `connectionName`.

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
  analytics:
    auth:
      type: api_key
      key: X-API-Key              # header name
      value: "${ANALYTICS_KEY}"
  oauth:
    auth:
      type: oauth2
      token: "${OAUTH2_ACCESS_TOKEN}"
  via-proxy:
    proxy: "http://${PROXY_USER}:${PROXY_PASS}@proxy.corp:8080"
    auth:
      type: bearer
      token: "${API_TOKEN}"
```

<div v-pre>

```yaml
component:
  name: http_request
  with:
    method: GET
    url: "https://api.stripe.com/v1/charges"
    connectionName: stripe        # auth + proxy from http_connections.stripe
```

</div>

`accountName` is a lighter alternative for services the agent authenticates with as itself, using a named account from the agent's [identity](/reference/advanced-config#agent-identity). It resolves as Basic Auth; when both are set, `connectionName` wins.

```yaml
# ~/.kdeps/config.yaml
identity:
  accounts:
    crm:
      username: "${CRM_USER}"
      password: "${CRM_PASS}"
```

<div v-pre>

```yaml
component:
  name: http_request
  with:
    method: GET
    url: "https://crm.example.com/contacts"
    accountName: crm
```

</div>

### `http_connections` fields (in `~/.kdeps/config.yaml`)

| Field | Type | Description |
|---|---|---|
| `auth.type` | string | `basic`, `bearer`, `api_key`, or `oauth2` |
| `auth.username` | string | Basic auth username |
| `auth.password` | string | Basic auth password |
| `auth.token` | string | Bearer / OAuth2 token |
| `auth.key` | string | API key header name |
| `auth.value` | string | API key header value |
| `proxy` | string | Proxy URL (may include `user:pass@`) |

## File upload, download and failing on errors

`files:` sends a `multipart/form-data` body: each key is a form field, each value the path of the file to upload; `data:` holds the text fields of the same form. `saveTo:` streams a 2xx response into a file instead of memory. `failOnError: true` turns a non-2xx response into a resource error instead of a result to inspect.

<div v-pre>

```yaml
# resources/upload.yaml - multipart upload (e.g. a Whisper API)
component:
  name: http_request
  with:
    method: POST
    url: https://api.openai.com/v1/audio/transcriptions
    headers:
      Authorization: "Bearer {{ env('OPENAI_API_KEY') }}"
    files:
      file: "{{ get('audioPath') }}"   # field name -> file path, uploaded with its base name
    data:
      model: whisper-1                 # text fields of the same form
      language: "{{ get('lang') }}"
    failOnError: true                  # 4xx/5xx fails the resource with the status and body
```

```yaml
# resources/download.yaml - fetch a file once
validations:
  skip:
    - "fileExists('/data/models/model.bin')"
component:
  name: http_request
  with:
    method: GET
    url: https://example.com/model.bin
    saveTo: /data/models/model.bin     # parent dirs created; written to .part then renamed
    timeout: 900s
```

</div>

With `saveTo`, the result is `{statusCode, status, headers, savedTo, bytes}` instead of `body`/`data`. A non-2xx response is not saved and comes back as a normal result, or an error with `failOnError`.

## Retry

Retries use exponential backoff: attempt 1 immediately, then `backoff`, `2x backoff`, up to `maxBackoff`.

<div v-pre>

```yaml
component:
  name: http_request
  with:
    method: GET
    url: "https://api.example.com/data"
    retry:
      maxAttempts: 3          # total attempts, including the first
      backoff: 1s             # initial delay; doubles each retry
      maxBackoff: 30s         # ceiling on the delay
      retryOn: [429, 500, 502, 503, 504]   # status codes to retry
```

</div>

## Response caching

Presence of the `cache:` block enables caching in the process memory store, shared across component calls.

<div v-pre>

```yaml
component:
  name: http_request
  with:
    method: GET
    url: "https://api.example.com/static-data"
    cache:
      ttl: 5m                    # cache lifetime
      key: "static-data-cache"   # optional; defaults to the method and URL
```

</div>

## TLS

<div v-pre>

```yaml
# Skip certificate verification - development only
component:
  name: http_request
  with:
    method: GET
    url: "https://self-signed.example.com"
    tls:
      insecureSkipVerify: true

# Client certificates
component:
  name: http_request
  with:
    method: GET
    url: "https://internal.example.com"
    tls:
      certFile: "/certs/client.pem"
      keyFile: "/certs/client-key.pem"
      caFile: "/certs/ca.pem"
```

</div>

## Redirects

Redirects are followed by default. Set `followRedirects: false` to stop at the first 3xx and return it as the result.

<div v-pre>

```yaml
component:
  name: http_request
  with:
    method: GET
    url: "https://api.example.com/redirect"
    followRedirects: false
```

</div>

## Output

```json
{
  "statusCode": 200,
  "status": "200 OK",
  "headers": {"Content-Type": "application/json"},
  "body": "{\"name\":\"Ada\"}",
  "data": {"name": "Ada"}
}
```

| Key | Type | Description |
|-----|------|-------------|
| `statusCode` | integer | HTTP status code |
| `status` | string | Status line, e.g. `200 OK` |
| `headers` | object | Response headers |
| `body` | string | Raw response body |
| `data` | any | Parsed body when it is JSON; absent otherwise |
| `savedTo` | string | File the body was streamed into (`saveTo`) |
| `bytes` | integer | Bytes written (`saveTo`) |

Read the parsed body with `get('fetch').data`, the status with `get('fetch').statusCode`.

### Response accessors

Resource-specific accessors read the same result:

<div v-pre>

```yaml
# resources/fetch.yaml runs first; this resource reads it
after:
  - set('response_body', http.responseBody('fetch'))            # body only
  - set('content_type', http.responseHeader('fetch', 'Content-Type'))
  - set('is_success', get('fetch').statusCode >= 200 && get('fetch').statusCode < 300)
```

</div>

See [Unified API](/workflow/data-access#resource-specific-accessors) for details.

## Error handling

Use preflight checks to validate before making requests:

<div v-pre>

```yaml
validations:
  check:
    - get('user_id') != ''
  error:
    code: 400
    message: "user_id is required"

component:
  name: http_request
  with:
    method: GET
    url: "https://api.example.com/users/{{ get('user_id') }}"
    connectionName: myapi
```

</div>

## See also

- [HTTP client examples](/reference/http-client-examples) - GitHub, Stripe, webhook, cached API examples
- [Scraper component](/workflow/resources/scraper) - fetch and extract readable text from a page
- [SQL resource](/workflow/resources/sql) - database operations
- [Unified API](/workflow/data-access) - data access patterns
