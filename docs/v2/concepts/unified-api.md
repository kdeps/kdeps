# Data access (get, set, input, request)

`get()` and `set()` are the two functions you use for almost everything in kdeps. `input` and `request` are property-style shorthands for the incoming HTTP request. All of them work the same way in string interpolation <span v-pre>`{{ }}`</span>, in `before:`/`after:` blocks, and in `validations.check` conditions. `get()`/`set()` are available in both modes; `input`/`request` are workflow mode only (they read the HTTP request).

## get() - read any value

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
  8. system metadata
  9. uploaded files
```

To skip the chain and read from a specific source:

```yaml
get('q', 'param')      # URL param or body field only
get('auth', 'header')  # request header only
get('user', 'session') # session only -- persists across requests
get('API_KEY', 'env')  # environment variable
```

Reading a resource output works the same way - `get('llm')` returns whatever the `llm` resource produced:

```yaml
# resources/example.yaml
requires: [llm]
apiResponse:
  response:
    answer: get('llm')          # full output
    text: get('llm').answer     # field access when LLM returns JSON
```

## set() - store a value

`set()` writes into memory (current request) by default. Pass `'session'` to persist across requests.

```yaml
# resources/example.yaml
after:
  - set('normalized', lower(trim(get('q'))))   # available to downstream resources
  - set('user_id', get('id'), 'session')        # survives to the next request
```

`set()` is like assigning to a variable. Downstream resources read it with `get()`.

## file() - read uploaded files

```yaml
# resources/example.yaml
content: file('doc.pdf')    # file uploaded with the request
images: file('*.jpg')       # glob pattern -- returns first match
```

## info() - request metadata

```yaml
# resources/example.yaml
id: info('ID')           # unique ID for this request
ip: info('IP')           # caller IP address
path: info('path')       # URL path
ts: info('timestamp')    # current timestamp
```

## Resource-specific accessors

`get('resourceId')` returns the main output of a resource. Use these accessors when you need lower-level details:

```yaml
# resources/example.yaml
after:
  # Python and exec resources
  - set('ok',  exec.exitCode('build') == 0)
  - set('err', exec.stderr('build'))

  # HTTP resources
  - set('status', get('api').statusCode)
  - set('ct',     http.responseHeader('api', 'Content-Type'))

  # LLM resources
  - set('raw', llm.response('chat'))
```

## The input object

`input.field` is a shorthand for a request **body** field - identical to `get('field')`, but with property syntax:

<div v-pre>

```yaml
# resources/example.yaml
after:
  - set('query', input.q)
  - set('city', input.user.address.city)   # nested access
chat:
  prompt: "Hello {{ input.name }}, you asked about {{ input.topic }}"
```

</div>

| `input`             | equivalent `get()`  |
|---------------------|---------------------|
| `input.field`       | `get('field')`      |
| `input.user.name`   | `get('user').name`  |
| `input.items[0]`    | `get('items')[0]`   |

`input` reads body data only - no query params, no headers, no source hints. It returns `nil` for a missing field, so wrap it in `default()` when the field is optional. `input.name` is exactly `request.body.name`. Reach for `get()` when you need any other source or a fallback.

## The request object

`request` gives direct access to the raw HTTP request when running in workflow mode.

| Property           | Type   | Description                    |
|--------------------|--------|-------------------------------|
| `request.method`   | string | HTTP method (GET, POST, ...)  |
| `request.path`     | string | Request URL path              |
| `request.IP`       | string | Client IP address             |
| `request.ID`       | string | Unique request identifier     |
| `request.headers`  | object | All request headers           |
| `request.query`    | object | Query parameters              |
| `request.body`     | object | Request body (same as `input`)|

File and lookup methods:

```yaml
# resources/example.yaml
after:
  - set('doc',     request.file('document'))       # file content (text) or nil
  - set('path',    request.filepath('image'))      # temp path on disk
  - set('type',    request.filetype('upload'))     # MIME type
  - set('images',  request.filesByType('image/*')) # paths matching a MIME glob
  - set('count',   request.filecount())            # number of uploaded files
  - set('all',     request.files())                # all uploaded file paths
  - set('types',   request.filetypes())            # MIME type of every upload
  - set('auth',    request.header('Authorization'))
  - set('page',    request.params('page'))         # query parameter
```

Each `request` method has a `get()`/`info()` equivalent - use whichever reads clearer:

| `request`                  | Unified API              |
|----------------------------|--------------------------|
| `request.params('key')`    | `get('key', 'param')`    |
| `request.header('Name')`   | `get('Name', 'header')`  |
| `request.file('name')`     | `get('name', 'file')`    |
| `request.filepath('name')` | `get('name', 'filepath')`|
| `request.filecount()`      | `info('filecount')`      |
| `request.method`           | `info('method')`         |

## See also

- [Expressions](/concepts/expressions) - expression syntax
- [Expression helpers](/concepts/expression-helpers) - `Json()`, `Safe()`, `default()`, and friends
- [Expression functions reference](/reference/expression-functions-reference) - complete function list
- [File upload example](/examples/file-upload) - `request.file()` end to end
