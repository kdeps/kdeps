# Data access (get, set, input, request)

`get()` reads a value and `set()` stores one. `input` and `request` read the HTTP request.

```yaml
get('q', 'param')      # URL param or body field only
get('auth', 'header')  # request header only
get('user', 'session') # session only -- persists across requests
get('API_KEY', 'env')  # environment variable
```

## Read next

- [Data access (get, set, input, request) reference](/reference/workflow/data-access) - every key, option and detail
- [Expressions](/workflow/expressions) - expression syntax
- [Expression helpers](/workflow/expression-helpers) - `Json()`, `Safe()`, `default()`, and friends
