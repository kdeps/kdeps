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

## Read next

- [http_request component reference](/reference/resources/http-client) - every key, option and detail
