# Call an authenticated external API

*Applies to workflow mode.*

## Overview

In this tutorial you build an endpoint that calls a third-party API with a
bearer token, retries on transient failures, and caches the response. You will
also see API-key auth and TLS options.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- HTTP status codes and auth headers

By the end you will be able to:

- Add bearer or API-key auth to an `httpClient:` request
- Retry on specific status codes with exponential backoff
- Cache a response with a TTL and a cache key

## Background

The `httpClient:` resource makes an outbound request and stores the parsed body
as its output. It has built-in `auth:`, `retry:`, `cache:`, and `tls:` blocks -
you do not write retry loops or token headers by hand.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.
- Network access (the example calls `httpbin.org`).

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
  auth:
    type: bearer
    token: "{{ get('api_token', 'demo-token') }}"   # ?api_token=... or default
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
kdeps run .
```

```bash
curl "http://localhost:16395/api/v1/call?api_token=my-real-token" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

The second identical call within 5 minutes returns the cached body without
hitting `httpbin.org`.

## Other auth types

API key in a header:

```yaml
# resources/call.yaml (auth block)
auth:
  type: api_key
  key: "X-API-Key"
  value: "{{ get('api_key') }}"
```

Basic auth:

```yaml
auth:
  type: basic
  username: "{{ get('user') }}"
  password: "{{ get('pass') }}"
```

Skip TLS verification (test environments only):

```yaml
tls:
  insecureSkipVerify: true
```

## Summary

You built an endpoint that:

- Adds a bearer token with `auth: { type: bearer }`
- Retries only on 5xx with exponential backoff via `retry:`
- Caches the response for 5 minutes with `cache:`

## Next steps

- [HTTP client resource](/resources/http-client) - all auth types, proxy, named connections
- [HTTP client examples](/reference/http-client-examples) - pagination, file downloads
- [Error handling (onError)](/concepts/error-handling) - fallback when retries are exhausted
- [Global config](/configuration/advanced) - named HTTP connections with shared credentials
