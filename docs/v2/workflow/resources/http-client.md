# http_request component

The `http_request` component makes one outbound HTTP request and returns the status code, headers, raw body, and the parsed JSON body under `data`.

```text
component: http_request -> kdeps API tool -> HTTP request -> {statusCode, headers, body, data}
```

<div v-pre>

```yaml
# resources/fetch.yaml
actionId: fetch
component:
  name: http_request                 # from the components asset set
  with:
    method: GET                      # GET, POST, PUT, PATCH, DELETE (default GET)
    url: "https://api.example.com/users/{{ get('id') }}"
```

</div>

The next resource reads the JSON body with `get('fetch').data`. Credentials go in `http_connections` in `~/.kdeps/config.yaml`, not in the workflow.

## Read next

- [http_request reference](/reference/resources/http-client) - methods, authentication, uploads, retry, caching, TLS, redirects, output, errors
- [HTTP client examples](/reference/http-client-examples) - common request shapes
