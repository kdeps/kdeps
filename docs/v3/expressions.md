# Expressions

Glue between resources: pass data, validate, branch.

## Two forms

**Interpolation** in strings:

<div v-pre>

```yaml
prompt: "Hello {{ get('name') }}"
url: "https://api.example.com/{{ get('id') }}"
```

</div>

**Bare statements** in `before:`, `after:`, `check:`, `skip:`, `onError.when`:

```yaml
before:
  - set('query', lower(trim(get('q'))))
  - set('page', int(get('page')) or 1)
validations:
  check:
    - get('email') != ''
```

## get / set

```yaml
get('q')                 # request field
get('llm')               # resource output (actionId)
get('record').name       # field access
get('items')[0]

set('key', value)                 # request scope
set('key', value, 'session')      # cross-request session
set('key', value, 'memory')       # persistent memory
set('key', value, 'item')         # items: loop scope
```

Optional source pin: `get('Authorization', 'header')`, `get('API_KEY', 'env')`, `get('page', 'param')`.

LookupLookup order (simplified):** item context → request memory → persistent memory → session → resource outputs → query → body → headers → files → system `info()`.

## info / env

```yaml
info('ID')           # request id
info('timestamp')
env('HOME')
```

## Useful library (subset)

| Area | Examples |
|------|----------|
| String | `trim`, `lower`, `upper`, `split`, `join`, `replace`, `len`, `matches` |
| Number | `int`, `float`, `min`, `max`, `abs` |
| List / map | `filter`, `map`, `len`, indexing |
| Logic | `and`, `or`, `not`, `?:`, comparisons |

Heavy logic → `python:` or `exec:`. Full surface evolves with the binary — validate expressions with `kdeps validate`.

## With iteration and errors

- `items:` → `get('current')`, item scope — [Iteration](/iteration)  
- Failures → [Errors](/errors)  
- Sessions → [Config](/config)

[Resources](/resources) · [LLM](/llm).
