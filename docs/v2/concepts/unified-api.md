# Unified API

`get()` and `set()` are the two functions you use for almost everything in kdeps. They work the same way in string interpolation `{{ }}`, in `before:`/`after:` blocks, and in `validations.check` conditions.

## get() -- read any value

`get('key')` searches a priority chain and returns the first match. You rarely need to specify a source explicitly.

```
get('q') search order:
  1. items context    (loop/iteration current item)
  2. memory           (values set with set() this request)
  3. session          (values set with set(..., 'session'))
  4. resource outputs (values produced by prior resources)
  5. URL query params (?q=hello)
  6. request body     ({"q": "hello"})
  7. request headers
  8. uploaded files
  9. system metadata
```

To skip the chain and read from a specific source:

```yaml
get('q', 'param')      # URL param or body field only
get('auth', 'header')  # request header only
get('user', 'session') # session only -- persists across requests
get('API_KEY', 'env')  # environment variable
```

Reading a resource output works the same way -- `get('llm')` returns whatever the `llm` resource produced:

```yaml
requires: [llm]
apiResponse:
  response:
    answer: get('llm')          # full output
    text: get('llm').answer     # field access when LLM returns JSON
```

## set() -- store a value

`set()` writes into memory (current request) by default. Pass `'session'` to persist across requests.

```yaml
after:
  - set('normalized', lower(trim(get('q'))))   # available to downstream resources
  - set('user_id', get('id'), 'session')        # survives to the next request
```

`set()` is like assigning to a variable. Downstream resources read it with `get()`.

## file() -- read uploaded files

```yaml
content: file('doc.pdf')    # file uploaded with the request
images: file('*.jpg')       # glob pattern -- returns first match
```

## info() -- request metadata

```yaml
id: info('requestId')    # unique ID for this request
ip: info('clientIp')     # caller IP address
path: info('path')       # URL path
ts: info('timestamp')    # current timestamp
```

## Resource-specific accessors

`get('resourceId')` returns the main output of a resource. Use these accessors when you need lower-level details:

```yaml
after:
  # Python and exec resources
  - set('ok',  exec.exitCode('build') == 0)
  - set('err', exec.stderr('build'))

  # HTTP resources
  - set('status', http.responseBody('api').statusCode)
  - set('ct',     http.responseHeader('api', 'Content-Type'))

  # LLM resources
  - set('raw', llm.response('chat'))
```

## See Also

- [Request Object](/concepts/request-object) - HTTP request data and file methods
- [Expression Functions Reference](/reference/expression-functions-reference) - Complete function list
- [Expressions](/advanced/expressions) - Expression syntax