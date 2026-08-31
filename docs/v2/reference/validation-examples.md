# Validation and control flow - examples

Best practices and examples for the [`validations:` block](/concepts/validation-and-control).

*Applies to workflow mode.*

## Best practices

### Use [`skip`](/reference/glossary#skip) for optional logic

```yaml
# resources/example.yaml
validations:
  skip:
    - get('enableCache') != true
```

### Validate early with [`check`](/reference/glossary#check)

<div v-pre>

```yaml
# Good: Catch errors before expensive operations
validations:
  check:
    - get('userId') != null
    - get('apiKey') != ''
  error:
    code: 400
    message: "userId and apiKey are required"
```

</div>

### Restrict routes for security

<div v-pre>

```yaml
# resources/example.yaml
validations:
  routes: [/api/v1/admin]
  methods: [POST]
```

</div>

### Combine all controls

<div v-pre>

```yaml
# resources/example.yaml
validations:
  methods: [POST]
  routes: [/api/v1/admin]
  headers: [Authorization]
  check:
    - get('adminToken') != null
  error:
    code: 401
    message: Admin token required
  skip:
    - get('dryRun') == true
chat:
  prompt: "Admin: {{ get('action') }}"
```

</div>

## Examples

### Conditional processing

```yaml
# resources/smart-processor.yaml
actionId: smartProcessor
name: Smart Processor
validations:
  skip:
    - get('process') != true
  check:
    - get('data') != null
    - len(get('data')) > 0
  error:
    code: 400
    message: Data is required
python:
  script: |
    data = get('data')
    return process(data)
```

### Secure endpoint

<div v-pre>

```yaml
# resources/secure-endpoint.yaml
actionId: secureEndpoint
name: Secure Endpoint
validations:
  methods: [POST]
  routes: [/api/v1/secure]
  headers: [Authorization, Content-Type]
  check:
    - get('Authorization') != null
    - get('Authorization') startsWith 'Bearer '
  error:
    code: 401
    message: Valid authorization token required
chat:
  prompt: "Secure: {{ get('q') }}"
```

</div>

## See also

- [Validation and control flow](/concepts/validation-and-control) - Full `validations:` block reference
- [Expressions](/concepts/expressions) - Expression syntax for conditions
- [Unified API](/concepts/unified-api) - Using `get()` in validations
