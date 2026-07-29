# Expressions

Lightweight access to request data and earlier resource output.

## get and set

<div v-pre>

```yaml
prompt: "{{ get('q') }}"
# ...
answer: get('llm').message.content
```

</div>

- `get('q')` — request field / param
- `get('llm')` — full output of resource `actionId: llm`
- `set('key', value)` — store a value for later steps (`before` / `after`)

Use mustache-style double braces around expressions inside string templates. Outside strings, bare expressions are fine.

## Validation checks

```yaml
validations:
  check:
    - get('q') != ''
    - get('limit') <= 100
  skip:
    - get('dry_run') == true
  error:
    code: 400
    message: Invalid input
```

Failed `check` returns the error. True `skip` skips the resource.

## Tips

- Keep expressions short. Heavy logic belongs in `python` or `exec`.
- Prefer stable `actionId` names — they are your API between steps.
- For lists, `items:` runs the resource per element; combine with `get` carefully.

If you need the full operator and function list, start from examples in the repo (`examples/`) and the archived v2 reference under `/v2/` until this page grows a short appendix.
