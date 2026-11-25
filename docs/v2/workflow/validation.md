# Validation and control flow

The `validations:` block controls if a resource runs and what it accepts. It operates before the action: before an LLM call, an HTTP request, or a script.

```yaml
# resources/example.yaml
validations:
  methods: [POST]          # skip unless method matches
  routes: [/api/v1/data]  # skip unless route matches
  headers: [Authorization] # skip unless header present
  params: [q]              # skip unless param present
  skip:
    - get('mode') == 'dry-run'  # skip silently (no error)
  check:
    - get('q') != ''            # fail with error below if false; a missing q equals ''
  error:
    code: 400
    message: "q is required"
```

## Read next

- [Validation and control flow reference](/reference/workflow/validation) - skip conditions, preflight checks, route and method restrictions, input validation, allowed headers and parameters, execution order, use cases
- [Validation examples](/reference/validation-examples) - best practices and examples
- [Expressions](/workflow/expressions) - expression syntax for conditions
- [Resources overview](/workflow/resources) - resource structure
- [Unified API](/workflow/data-access) - using `get()` in validations
- [Workflow configuration](/workflow/configuration) - route configuration
