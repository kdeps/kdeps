# Call an authenticated external API

Build an endpoint that calls a third-party API with a
bearer token, retries on transient failures, and caches the response. You will
also see API-key auth and TLS options.

The `httpClient:` resource makes an outbound request and stores the parsed body
as its output. Credentials come from a named connection in `~/.kdeps/config.yaml`
(`connectionName:`), so the token never sits in the repo. `retry:`, `cache:` and
`tls:` are built in - you do not write retry loops or token headers by hand.

**Needs:** Network access (the example calls `httpbin.org`).

## Step 1: create the project

```bash
mkdir http-auth
cd http-auth
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: http-auth
  version: "1.0.0"
  targetActionId: result

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/call
        methods: [GET]
```

## Step 3: the authenticated call

Create `resources/call.yaml`:

<div v-pre>

```yaml
# resources/call.yaml
actionId: authedCall
name: Authenticated API call
validations:
  methods: [GET]
  routes: [/api/v1/call]
httpClient:
  method: GET
  url: "https://httpbin.org/bearer"
  timeout: 10s
  connectionName: httpbin                           # auth from http_connections.httpbin
  retry:
    maxAttempts: 3
    backoff: 1s
    maxBackoff: 5s
    retryOn: [500, 502, 503, 504]                   # retry only these
  cache:
    ttl: 5m                                         # reuse the response for 5 minutes
    key: "bearer_call"                              # cache entry name
```

</div>

The `httpbin` connection holds the bearer token, machine-locally:

```yaml
# ~/.kdeps/config.yaml
http_connections:
  httpbin:
    auth:
      type: bearer          # basic | bearer | api_key | oauth2
      token: my-real-token
```

## Step 4: return the result

Create `resources/result.yaml`:

<div v-pre>

```yaml
# resources/result.yaml
actionId: result
name: Result
requires: [authedCall]
apiResponse:
  success: true
  response:
    upstream: "{{ get('authedCall') }}"            # the parsed response body
    status: "{{ get('authedCall').statusCode }}"
    at: "{{ info('current_time') }}"
```

</div>

## Step 5: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
# skip these two if http_connections.httpbin is in config.yaml
export KDEPS_HTTP_CONNECTIONS_HTTPBIN_AUTH_TYPE=bearer
export KDEPS_HTTP_CONNECTIONS_HTTPBIN_AUTH_TOKEN=my-real-token
kdeps run .
```

```bash
curl http://localhost:16395/api/v1/call \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

The upstream accepted the token:

```json
{
  "success": true,
  "data": {
    "status": 200,
    "upstream": {
      "statusCode": 200,
      "data": {"authenticated": true, "token": "my-real-token"},
      "headers": {"Content-Type": "application/json", "...": "..."}
    },
    "at": "..."
  }
}
```

A `"statusCode": 401` here means the connection has no token: check the two
`KDEPS_HTTP_CONNECTIONS_HTTPBIN_*` variables or `http_connections.httpbin`.

The second identical call within 5 minutes returns the cached body without
hitting `httpbin.org`.

## Other auth types

Other auth types are set on the named connection in `~/.kdeps/config.yaml`:

```yaml
# ~/.kdeps/config.yaml
http_connections:
  partner_api:
    auth:
      type: api_key
      key: "X-API-Key"      # header name
      value: "sk-..."       # header value
  legacy_api:
    auth:
      type: basic
      username: svc-user
      password: "..."
```

Skip TLS verification (test environments only):

```yaml
tls:
  insecureSkipVerify: true
```

## Next steps

- [HTTP client resource](/workflow/resources/http-client) - all auth types, proxy, named connections
- [HTTP client examples](/reference/http-client-examples) - pagination, file downloads
- [Error handling (onError)](/workflow/error-handling) - fallback when retries are exhausted
- [Global config](/reference/advanced-config) - named HTTP connections with shared credentials
