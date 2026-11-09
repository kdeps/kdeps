# Validation and control flow - examples

Best practices and examples for the [`guardrails:` block](/workflow/validation).

Workflow mode only.

## Best practices

### Use [`skip`](/reference/glossary#skip) for optional logic

```yaml
# resources/example.yaml
guardrails:
  skip:
    - get('enableCache') != true
```

### Validate early with [`check`](/reference/glossary#check)

<div v-pre>

```yaml
# Good: Catch errors before expensive operations
guardrails:
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
guardrails:
  routes: [/api/v1/admin]
  methods: [POST]
```

</div>

### Combine all controls

<div v-pre>

```yaml
# resources/example.yaml
guardrails:
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
  context:
    prompt: "Admin: {{ get('action') }}"
```

</div>

## Examples

### Conditional processing

```yaml
# resources/smart-processor.yaml
actionId: smartProcessor
name: Smart Processor
guardrails:
  skip:
    - get('process') != true
  check:
    - get('data') != null
    - len(get('data')) > 0
  error:
    code: 400
    message: Data is required
component:
  name: python
  with:
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
guardrails:
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
  context:
    prompt: "Secure: {{ get('q') }}"
```

</div>

## See also

- [Validation and control flow](/workflow/validation) - Full `guardrails:` block reference
- [Expressions](/workflow/expressions) - Expression syntax for conditions
- [Unified API](/workflow/data-access) - Using `get()` in validations
