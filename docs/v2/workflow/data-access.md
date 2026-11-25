# Data access (get, set, input, request)

`get()` and `set()` are the two functions you use for almost everything in kdeps. `input` and `request` are property-style shorthands for the incoming HTTP request. All of them work the same way in string interpolation <span v-pre>`{{ }}`</span>, in `before:`/`after:` blocks, and in `guardrails.check` conditions. `get()` and `set()` work in every workflow. `input` and `request` read the HTTP request, so they apply to that server.

```yaml
get('q', 'param')      # URL param or body field only
get('auth', 'header')  # request header only
get('user', 'session') # session only -- persists across requests
get('API_KEY', 'env')  # environment variable
```

## Read next

- [Data access (get, set, input, request) reference](/reference/workflow/data-access) - get() - read any value, set() - store a value, file() - read uploaded files, info() - request metadata, resource-specific accessors, the input object, the request object
- [Expressions](/workflow/expressions) - expression syntax
- [Expression helpers](/workflow/expression-helpers) - `Json()`, `Safe()`, `default()`, and friends
- [Expression functions reference](/reference/expression-functions) - complete function list
- [File upload example](/examples/file-upload) - `request.file()` end to end
