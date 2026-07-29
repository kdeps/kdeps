# Chapter 11: Expressions and Data Flow

Expressions are the glue between resources. They pass data through the pipeline, validate inputs, compute derived values, and control conditional logic. Understanding the expression system is what separates "it works" from "it's maintainable."

## Two Syntaxes

Expressions appear in two distinct syntaxes, each used in different contexts.

**String interpolation** — embed an expression in any string value using `&#123;&#123; &#125;&#125;`:

```yaml
prompt: "Hello &#123;&#123; get('name') &#125;&#125;, your score is &#123;&#123; get('score') * 100 &#125;&#125;%"
url: "https://api.example.com/users/&#123;&#123; get('userId') &#125;&#125;/profile"
command: "process --limit &#123;&#123; get('limit') or 10 &#125;&#125;"
```

**Bare statements** — in `before:`, `after:`, `validations.check`, `validations.skip`, `onError.when`:

```yaml
before:
  - set('query', lower(trim(get('q'))))
  - set('page', int(get('page')) or 1)
  - set('limit', min(int(get('limit')) or 10, 100))

validations:
  check:
    - get('email') != ''
    - get('email') matches '^[^@]+@[^@]+\\.[^@]+$'
```

In `before:` and `after:`, the statements execute sequentially. In `check:` and `skip:`, each expression must evaluate to a boolean. If any `check:` is false, the resource stops.

## The Data Store Functions

### get(key)

Read a value from the request-scoped key-value store:

```yaml
get('q')               # reads 'q' from the request body
get('llm')             # reads the output of resource with actionId: llm
get('record').name     # accesses a field of a stored object
get('items')[0]        # accesses first element of a stored array
get('items')[0].price  # chained field access
```

`get()` searches a priority chain and returns the first match:

1. **Items context** — current iteration item (inside `items:` loops)
2. **Memory** — values set with `set()` in this request
3. **Persistent memory** — values saved with `set(..., 'memory')` from any previous request
4. **Session** — values set with `set(..., 'session')` from previous requests
5. **Resource outputs** — results stored under their `actionId`
6. **URL query params** — `?key=value` from the request URL
7. **Request body** — JSON body fields
8. **Request headers** — HTTP headers
9. **Uploaded files** — file content/metadata
10. **System metadata** — same as `info()`

If a key does not exist in any source, `get()` returns `null`.

An optional second argument forces `get()` to read from a specific source, bypassing auto-detection:

```yaml
get('Authorization', 'header')    # read from request headers only
get('user_id', 'session')         # read from session storage
get('API_KEY', 'env')             # read from environment variables
get('page', 'param')              # read from URL query parameters
```

This is useful when a key name is ambiguous — for example, if both the request body and session contain a key named `user_id`.

### set(key, value)

Write a value to the store:

```yaml
before:
  - set('normalized', lower(trim(get('q'))))
  - set('timestamp', info('timestamp'))
  - set('items', filter(get('rawItems'), {.active == true}))
```

`set()` accepts an optional third argument for the storage scope:

```yaml
set('key', value)                 # request-scoped (default)
set('key', value, 'session')      # session-scoped (persists across requests from same session)
set('key', value, 'memory')       # persistent memory (survives restarts, shared across all sessions)
```

Session storage is covered in Chapter 15. Persistent memory is covered in Chapter 15.

### info(key)

Read system-level metadata about the current request:

```yaml
info('ID')           # unique request ID
info('timestamp')    # request timestamp as RFC3339 string (e.g. 2024-12-25T14:30:00Z)
info('sessionId')    # session ID (if sessions enabled)
info('IP')           # client IP address
info('path')         # request URL path
info('method')       # HTTP method
info('filecount')    # number of uploaded files in this request
info('files')        # array of uploaded file paths
info('filetypes')    # array of uploaded file MIME types
```

Useful for logging, tracing, session management, and inspecting uploaded file metadata.

### env(key)

Read an environment variable:

```yaml
env('OPENAI_API_KEY')
env('DATABASE_URL')
env('DEBUG')
```

Returns `null` if the variable is not set. Never hardcode credentials in workflow files; always use `env()`.

### Resource-Specific Accessors

Beyond `get()`, some resource types expose lower-level accessors for inspecting execution details:

```yaml
# exec: resources
exec.exitCode('resourceId')               # integer exit code (0 = success)
exec.stderr('resourceId')                 # stderr output as string

# httpClient: resources
http.responseHeader('resourceId', 'Content-Type')   # response header value

# chat: resources
llm.response('resourceId')               # raw LLM response text
```

Example — check command success and capture errors:

```yaml
requires: [buildStep]
after:
  - set('build_ok', exec.exitCode('buildStep') == 0)
  - set('build_errors', exec.stderr('buildStep'))
```

Example — read a response header from an HTTP call:

```yaml
requires: [apiCall]
after:
  - set('rate_limit_remaining', http.responseHeader('apiCall', 'X-RateLimit-Remaining'))
  - set('content_type', http.responseHeader('apiCall', 'Content-Type'))
```

### output(actionId)

Read the output of a specific resource by its `actionId`:

```yaml
output('llm')             # same as get('llm') for resource outputs
output('fetchData').name  # field access on resource output
```

`output()` is identical to `get()` for resource outputs. The distinction is semantic: use `output()` when you are explicitly referring to a resource's output (as opposed to a request body field or a `set()` value). Some teams prefer `output()` for readability in complex workflows.

### session()

Return the entire session data object — all keys set with `set(..., 'session')` across previous requests in the current session:

```yaml
session()                           # returns the full session map
session().history                   # access a specific session key
len(session().history or [])        # safely count session items
```

This is useful when you need to inspect or iterate the full session state rather than reading individual keys with `get()`. If no session is active, returns an empty map.

## Standard Library

The expression engine includes the full [expr-lang](https://expr-lang.org/) standard library plus kdeps-specific helpers.

### String Operations

```yaml
# Trim whitespace
trim("  hello  ")              # "hello"
trim(get('q'))                 # trims the value of 'q'

# Case conversion
lower("Hello World")           # "hello world"
upper("hello")                 # "HELLO"

# Split and join
split("a,b,c", ",")            # ["a", "b", "c"]
join(["a", "b", "c"], ", ")    # "a, b, c"

# Replace
replace("hello world", "world", "kdeps")   # "hello kdeps"

# Check
"hello" contains "ell"         # true
"hello" startsWith "hel"       # true
"hello" endsWith "llo"         # true
"hello@world.com" matches '^[^@]+@[^@]+\\.[^@]+$'   # true (regex)

# Length
len("hello")                   # 5
len(get('text'))                # length of stored string

# Slice
get('text')[0:100]             # first 100 characters
```

### Numeric Operations

```yaml
int("42")           # 42
float("3.14")       # 3.14
string(42)          # "42"

min(3, 5)           # 3
max(3, 5)           # 5
abs(-7)             # 7
ceil(3.2)           # 4
floor(3.8)          # 3

# With nullsafe defaults
int(get('page')) or 1         # parse or default to 1
min(int(get('limit')) or 10, 100)  # parse, default 10, cap at 100
```

### List and Map Operations

```yaml
# Length
len(get('items'))                # number of items

# Access
get('items')[0]                  # first item
get('items')[len(get('items'))-1]  # last item

# First and last shorthand
first(get('items'))              # first element
last(get('items'))               # last element

# Slice (extract sub-array; negative indices count from end)
slice(get('items'), 0, 5)        # first five
slice(get('items'), -10, len(get('items')))  # last ten

# Filter (returns items where condition is true)
filter(get('items'), {.active == true})
filter(get('results'), {.score > 0.5})

# Map (transform each item)
map(get('results'), {.title})              # extract title field from each
map(get('items'), {.price * 1.21})        # apply VAT to each price

# Aggregation
sum(get('prices'))               # sum of all values
reduce(get('prices'), {# + .}, 0)  # fold to single value (expr-lang native)

# Membership
get('role') in ['admin', 'editor']        # true if role is admin or editor
get('status') not in ['deleted', 'banned']
```

### JSON Operations

```yaml
# Encode to JSON string
json({"key": "value", "count": 42})
json(get('record'))

# Decode from JSON string
fromJSON(get('rawJson'))
```

### Type Functions

```yaml
# Check for null
get('value') == null          # check for null
get('value') != null          # check not null

# Type assertions (expr-lang native)
get('value') is string        # type check
get('items') is array         # type check

# type() — returns type as a string
type(get('value'))            # "string", "int", "float", "bool", "array", "map", or "nil"
type(42)                      # "int"
type("hello")                 # "string"
type(null)                    # "nil"
```

### Date and Time

```yaml
# info('timestamp') — RFC3339 string; use in responses, logging, audit fields
info('timestamp')             # "2024-12-25T14:30:00Z"

# now() — returns a time.Time value; useful for comparisons
now()
```

### Conditional Operators

**Ternary `? :`** — inline if/else:

```yaml
set('status', get('score') >= 70 ? 'pass' : 'fail')
set('discount', get('isPremium') ? 0.2 : 0.1)
set('label', len(get('text')) > 1000 ? 'long' : 'short')
```

**Elvis `?:`** — return the left-hand value if truthy, otherwise the right-hand value. Shorthand for the common `x != nil ? x : y` pattern:

```yaml
set('name', get('name') ?: 'Unknown')      # use stored name, or 'Unknown' if nil/empty
set('host', get('host') ?: 'localhost')    # prefer runtime value, fall back to default
```

The difference between the three conditional operators:

| Operator | Triggers when left is... | Use for |
|---|---|---|
| `? :` (ternary) | Any condition | Full if/else with a computed condition |
| `?:` (Elvis) | Nil or falsy | "Use this value, or that default" |
| `??` (null coalescing) | Nil or empty string only | "Use this if present, even if `false` or `0`" |

### Operator Precedence

When mixing operators without parentheses, this is the evaluation order (highest to lowest):

| Level | Operators |
|---|---|
| 1 (highest) | `(…)` — parentheses |
| 2 | `!`, unary `-` |
| 3 | `*`, `/`, `%` |
| 4 | `+`, `-` |
| 5 | `<`, `<=`, `>`, `>=` |
| 6 | `==`, `!=` |
| 7 | `&&`, `and` |
| 8 | `\|\|`, `or` |
| 9 | `? :` (ternary) |
| 10 (lowest) | `??` (null coalescing) |

When in doubt, add parentheses — `(get('a') > 0) && (get('b') != nil)` is always clearer than relying on precedence.

**Null coalescing `??`** — return right-hand value when left-hand is nil or empty string:

```yaml
set('name', get('name') ?? 'Anonymous')
set('limit', get('limit') ?? 10)
set('region', get('region') ?? env('DEFAULT_REGION'))
```

`??` is stricter than `?:` — it only triggers on nil/empty string, not on `false` or `0`. Use `??` when `0` or `false` are valid values you want to preserve.

## Helper Functions

kdeps provides utility functions that complement the standard library for common safety and debugging patterns.

### safe(obj, path)

Safely access nested properties without panicking when an intermediate value is nil:

```yaml
safe(get('user'), "profile.address.city")   # returns city, or nil if any level is nil
safe(get('response'), "data.items.0.name")  # safe array index access
```

Without `safe()`, accessing `get('user').profile.address.city` when `profile` is nil causes a runtime error. Use `safe()` any time you are accessing data from external APIs or LLM-produced JSON where the shape is not guaranteed.

### default(value, fallback)

Return a fallback value if the primary value is nil or empty:

```yaml
default(get('limit'), 10)                   # 10 if limit is missing
default(get('language'), 'en')              # 'en' if language is missing
default(safe(get('config'), 'timeout'), 30) # combine with safe()
```

Equivalent to `get('limit') ?? 10` but reads more explicitly as "default to 10."

### debug(obj)

Return a pretty-printed JSON string for inspecting complex objects during development:

```yaml
after:
  - set('_debug_response', debug(get('httpResponse')))
  - set('_debug_extract', debug(get('extract')))
```

The `_debug_` prefix is a convention — kdeps does not treat it specially, but it signals to readers that this value is diagnostic only and should be removed before production.

### urlencode(string)

URL-encode a string for use in query parameters or path segments:

```yaml
url: "https://api.example.com/search?q=&#123;&#123; urlencode(get('query')) &#125;&#125;"
url: "https://example.com/docs/&#123;&#123; urlencode(get('title')) &#125;&#125;"
```

## before: and after: Patterns

### Input Normalization (before:)

```yaml
before:
  - set('query', lower(trim(get('q'))))
  - set('page', max(int(get('page')) or 1, 1))
  - set('limit', min(int(get('limit')) or 20, 100))
  - set('tags', split(get('tags') or '', ','))
```

Run `before:` to clean and normalize inputs before the resource action sees them. The resource's `prompt:`, `query:`, or `command:` then reads the normalized values.

### Output Enrichment (after:)

```yaml
after:
  - set('word_count', len(split(get('llm'), ' ')))
  - set('first_sentence', split(get('llm'), '.')[0])
  - set('is_long', len(get('llm')) > 1000)
  - set('processed_at', info('timestamp'))
```

Run `after:` to compute derived values from the resource's output. These become available to downstream resources via `get()`.

### Conditional Logic

kdeps does not have if/else syntax at the resource level. Use `validations.skip` to conditionally skip a resource:

```yaml
# resources/premium-processing.yaml
actionId: premiumProcess
validations:
  skip:
    - get('tier') != 'premium'    # skip if not premium tier
chat:
  # expensive processing
```

For branching logic where different resources handle different cases, use route/method filtering plus `skip:` conditions. Design your DAG so that the response resource requires both branches, and whichever branch actually ran produces the output:

```yaml
# resources/respond.yaml
actionId: respond
requires: [premiumProcess, standardProcess]   # both are required
# But one will have been skipped, so get() reads from the one that ran
apiResponse:
  success: true
  response:
    result: get('premiumProcess') or get('standardProcess')
```

## Practical: Data Pipeline with Expressions

A complete example showing expressions throughout:

```yaml
# resources/ingest.yaml
actionId: ingest

before:
  - set('text', trim(get('content')))
  - set('lang', get('language') or 'en')
  - set('max_words', min(int(get('max_words')) or 500, 2000))

validations:
  check:
    - len(get('text')) > 0
    - len(get('text')) <= 50000
    - get('lang') in ['en', 'nl', 'de', 'fr']
  error:
    code: 422
    message: "content required (max 50000 chars), language must be en/nl/de/fr"

chat:
  model: llama3.2:1b
  systemPrompt: "You are a summarizer. Respond in the same language as the input."
  prompt: |
    Summarize in &#123;&#123; get('max_words') &#125;&#125; words or fewer:
    
    &#123;&#123; get('text') &#125;&#125;

after:
  - set('summary_words', len(split(get('ingest'), ' ')))
  - set('compression_ratio', float(len(get('text'))) / float(len(get('ingest'))))
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [ingest]
apiResponse:
  success: true
  response:
    summary: get('ingest')
    stats:
      original_chars: len(get('text'))
      summary_words: get('summary_words')
      compression_ratio: get('compression_ratio')
      language: get('lang')
```

The response includes both the summary and diagnostic metadata about the compression, all derived from expressions in `before:` and `after:` without any extra resource calls.

## The input Object Shorthand

`input.field` is shorthand for `get('field')` when reading request body fields. Both are equivalent:

```yaml
# These are identical
prompt: "Hello &#123;&#123; get('name') &#125;&#125;, you asked: &#123;&#123; get('topic') &#125;&#125;"
prompt: "Hello &#123;&#123; input.name &#125;&#125;, you asked: &#123;&#123; input.topic &#125;&#125;"
```

`input` supports nested access and array indexing:

```yaml
input.user.address.city      # get('user').address.city
input.items[0]               # get('items')[0]
input.items[0].price         # get('items')[0].price
```

Use whichever reads more naturally. `get()` is more explicit about its source; `input.` is more concise for simple request body reads.

## Inline Resources in before: and after:

Beyond expression statements, `before:` and `after:` blocks can contain full resource actions — `httpClient:`, `exec:`, `sql:`, `python:`, or `chat:`. These run as part of the containing resource's execution, without needing separate resource files.

```yaml
# resources/process.yaml
actionId: process

before:
  - httpClient:
      method: GET
      url: "https://api.example.com/config/&#123;&#123; get('configId') &#125;&#125;"
  - exec:
      command: "echo 'starting processing'"

chat:
  model: llama3.2:1b
  prompt: "Process with config: &#123;&#123; get('httpClient') &#125;&#125;"

after:
  - sql:
      connectionName: main
      query: "INSERT INTO audit_log (action, result) VALUES ('process', $1)"
      params:
        - get('process')
  - python:
      script: |
        import json
        print(json.dumps({"logged": True}))
```

The execution order within a resource is:

```
expressions in before:
inline resources in before:   (httpClient, exec, ...)
main action                   (chat, sql, python, ...)
inline resources in after:    (sql, python, ...)
expressions in after:
```

**When to use inline resources vs. separate resource files:**

Use separate files for anything you want to `requires:` from other resources, test independently, or reuse. Use inline resources for small, tightly-coupled side effects — a logging write, a config fetch, a cleanup command — where creating a separate file would be over-engineering.

Inline resources cannot be referenced by `requires:` from other resources. Their output is accessible via the action name as a key:

```yaml
before:
  - httpClient:
      method: GET
      url: "https://api.example.com/config"

# reads inline httpClient output
prompt: "Using config: &#123;&#123; get('httpClient') &#125;&#125;"
```

**Resource with no main action:**

A resource can contain only `before:` and `after:` inline resources with no primary action. This is useful for orchestration tasks that coordinate multiple side effects:

```yaml
# resources/log-and-notify.yaml
actionId: logAndNotify
requires: [process]

before:
  - exec:
      command: "echo 'starting post-process tasks'"

after:
  - sql:
      connectionName: main
      query: "INSERT INTO audit_log (action, result, ts) VALUES ('process', $1, NOW())"
      params:
        - get('process')
  - httpClient:
      method: POST
      url: "https://hooks.slack.com/services/&#123;&#123; get('slackHook', 'env') &#125;&#125;"
      body:
        text: "Process complete: &#123;&#123; get('process').summary &#125;&#125;"
```

No `chat:`, `sql:`, `python:`, or `exec:` main action is required. The resource exists purely to group inline side-effects that run after `process` completes.

## Jinja2 YAML Preprocessing

Before kdeps parses any YAML file, it preprocesses it through a Jinja2-compatible template engine. This lets you use environment variables to conditionally include configuration sections.

```yaml
# workflow.yaml
settings:
  apiServer:
    hostIp: "&#123;&#123; env.HOST_IP | default('127.0.0.1') &#125;&#125;"
{% if env.PORT %}
    portNum: &#123;&#123; env.PORT | int &#125;&#125;
{% else %}
    portNum: 16395
{% endif %}

{% if env.ENABLE_TLS == 'true' %}
    tls:
      certFile: "/certs/server.crt"
      keyFile: "/certs/server.key"
{% endif %}
```

Available context in Jinja2 preprocessing:

| Variable | Type | Description |
|---|---|---|
| `env` | map | All environment variables (`env.MY_VAR`) |
| `name` | string | Project name (scaffolding templates only) |
| `description` | string | Project description (scaffolding templates only) |
| `version` | string | Version string (scaffolding templates only) |
| `port` | int | API server port (scaffolding templates only) |
| `resources` | array | Enabled resource types (scaffolding templates only) |

The `name`, `description`, `version`, `port`, and `resources` variables are only available inside `.j2` scaffolding templates used by `kdeps new`. In regular workflow and resource YAML files, only `env` is available.

**Auto-protection of runtime calls:** kdeps automatically wraps all `get()`, `set()`, `info()`, `item()`, `loop()`, `session()`, `json()`, `safe()`, `debug()`, `default()`, `output()`, `file()`, and `input()` calls in Jinja2 `{% raw %}` blocks before preprocessing. You do not need to add `{% raw %}` yourself — expressions inside `&#123;&#123; &#125;&#125;` that use kdeps functions are preserved as-is after preprocessing.

```yaml
# This works correctly — no manual {% raw %} needed
httpClient:
  url: "https://&#123;&#123; env.API_HOST &#125;&#125;/users/&#123;&#123; get('userId') &#125;&#125;"
  #        ↑ Jinja2 evaluated      ↑ kdeps runtime, auto-protected
```

**Additional template syntax:**

Jinja2 comments — stripped before parsing, never appear in the final YAML:

```yaml
{# This section configures the rate limiter #}
settings:
  apiServer:
    rateLimit:
      requestsPerMinute: 60
```

Whitespace control — the `-` suffix trims surrounding whitespace and blank lines, useful when conditional blocks would otherwise produce empty lines in the YAML:

```yaml
settings:
{%- if env.TLS_ENABLED == 'true' %}
  tls:
    certFile: "/certs/server.crt"
    keyFile:  "/certs/server.key"
{%- endif %}
  apiServer:
    portNum: 16395
```

**When to use Jinja2 preprocessing vs. expressions:**

Use Jinja2 (`&#123;&#123; env.X &#125;&#125;`, `{% if env.X %}`) for configuration values that vary between environments but are fixed at startup: host names, ports, feature flags, TLS configuration.

Use kdeps expressions (`get()`, `set()`) for per-request data that varies between API calls.

## Common Mistakes

**Forgetting that `get()` returns null for missing keys.** Use `??` or `or` to provide defaults: `get('limit') ?? 10`. Accessing fields on null causes a runtime error.

**Using `or` when you mean `??`.** `get('count') or 0` returns `0` when count is `0` (falsy), even if the value was intentionally zero. Use `get('count') ?? 0` to only substitute when nil/missing.

**Putting `&#123;&#123; &#125;&#125;` in bare statement blocks.** In `before:`, `after:`, and `check:`, expressions are bare — no braces. Braces are only for string interpolation in string-typed fields.

**Mutating response objects.** The value stored under an `actionId` is the resource's raw output. Use `set()` to create derived values rather than trying to modify the output in place.

**Not normalizing before validation.** If a user sends `" admin "` (with spaces), `get('role') == 'admin'` fails. Always `trim()` string inputs in `before:` before validating.

**Accessing deep paths on untrusted data without `safe()`.** LLM outputs and external API responses may omit fields. Use `safe(get('data'), 'nested.path')` or `??` to guard against nil-path panics.

X> ## Exercise
X>
X> Write the `before:` and `after:` expressions for a resource that processes a raw user search query before sending it to an LLM.
X>
X> Given an incoming request body `{ "q": "  What IS the CAPITAL of france??  " }`, write expressions that produce:
X>
X> 1. `before:` — normalize the query: trim whitespace, convert to lowercase, strip trailing punctuation. Store the result as `cleanQuery`.
X> 2. `before:` — extract the word count of the cleaned query. Store as `wordCount`.
X> 3. `before:` — set a boolean `isShortQuery` that is `true` if `wordCount` is 3 or fewer.
X> 4. `after:` — append the request ID and a timestamp to a session key `queryLog` as a JSON object.
X>
X> Write each expression as a bare statement (not inside a string):
X> ```yaml
X> before:
X>   - set('cleanQuery', ...)
X>   - set('wordCount', ...)
X>   - set('isShortQuery', ...)
X> after:
X>   - set('queryLog', ..., 'session')
X> ```
X>
X> Verify your expressions produce `cleanQuery = "what is the capital of france"`, `wordCount = 7`, `isShortQuery = false` for the given input.
X>
X> **Stretch goal:** Write a ternary expression that selects a different prompt template based on `isShortQuery`: a one-sentence answer for short queries, a paragraph for longer ones.
