# Validation and control flow

The `validations:` block controls if a resource runs and what it accepts. It operates before the action: before an LLM call, an HTTP request, or a script.

```yaml
# resources/example.yaml
validations:
  methods: [POST]          # skip unless the method matches
  check:
    - get('q') != ''       # fail with the error below if false
  error:
    code: 400
    message: "q is required"
```

## Read next

- [Validation and control flow reference](/reference/workflow/validation) - every key, option and detail
- [Expressions](/workflow/expressions) - expression syntax for conditions
- [Resources overview](/workflow/resources) - resource structure
