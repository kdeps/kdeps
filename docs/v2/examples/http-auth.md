# Call an authenticated external API

Build an endpoint that calls a third-party API with a
bearer token, retries on transient failures, and caches the response. You will
also see API-key auth and TLS options.

The token is in `~/.kdeps/config.yaml`, not in the workflow:

```yaml
# ~/.kdeps/config.yaml
http_connections:
  httpbin:
    auth:
      type: bearer          # basic | bearer | api_key | oauth2
      token: my-real-token
```

## Read next

- [Full walkthrough](/reference/examples/http-auth) - every step, file and command
- [HTTP client component](/workflow/resources/http-client) - all auth types, proxy, named connections
- [Error handling (onError)](/workflow/error-handling) - fallback after the last retry
