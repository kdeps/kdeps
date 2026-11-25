# Route and method restrictions

`validations:` gates which HTTP methods and routes trigger a resource. A request that does not match skips the resource silently.

<div v-pre>

```yaml
# resources/create-user.yaml
actionId: createUser
validations:
  methods: [POST]

component:
  name: sql
  with:
    connectionName: primary
    queries:
      - query: "INSERT INTO users (name, email) VALUES (?, ?)"
        params:
          - "{{ get('name') }}"
          - "{{ get('email') }}"
```

</div>

## Read next

- [Route and method restrictions reference](/reference/workflow/route-restrictions) - HTTP method restrictions, route restrictions, combining restrictions, RESTful API example, best practices
- [Validation and control flow](/workflow/validation) - Full `validations:` block reference
- [Data access](/reference/workflow/data-access#the-request-object) - the `request` object
- [Workflow configuration](/workflow/configuration) - Route configuration
- [Resources overview](/workflow/resources) - Resource basics
