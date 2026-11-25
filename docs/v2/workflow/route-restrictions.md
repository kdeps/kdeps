# Route and method restrictions

`validations:` gates which HTTP methods and routes trigger a resource. A request that does not match skips the resource silently.

```yaml
# resources/create-user.yaml
validations:
  methods: [POST]
  routes: [/api/v1/users]
```

## Read next

- [Route and method restrictions reference](/reference/workflow/route-restrictions) - every key, option and detail
- [Validation and control flow](/workflow/validation) - Full `validations:` block reference
- [Workflow configuration](/workflow/configuration) - Route configuration
