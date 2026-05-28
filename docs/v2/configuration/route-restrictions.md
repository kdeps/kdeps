# Route and Method Restrictions

`validations:` gates which HTTP methods and routes can trigger a resource -- resources are skipped silently when the incoming request does not match.

## HTTP Method Restrictions

Limit a resource to specific HTTP methods using `methods:` inside the `validations:` block:

```yaml
# resources/create-user.yaml
actionId: createUser
validations:
  methods: [POST]

sql:
  connectionName: primary
  queries:
    - query: "INSERT INTO users (name, email) VALUES (?, ?)"
      params:
        - "{{ get('name') }}"
        - "{{ get('email') }}"
```

When `methods:` is set inside `validations:`:
- Resource only executes for matching HTTP methods
- Mismatched methods skip the resource silently
- Multiple methods can be specified

### Common Patterns

**CRUD Operations:**

```yaml
# Create - POST only
actionId: createItem
validations:
  methods: [POST]
sql:
  queries:
    - query: "INSERT INTO items (name) VALUES (?)"
      params: ["{{ get('name') }}"]

---
# Read - GET only
actionId: getItem
validations:
  methods: [GET]
sql:
  queries:
    - query: "SELECT * FROM items WHERE id = ?"
      params: ["{{ get('id') }}"]

---
# Update - PUT/PATCH
actionId: updateItem
validations:
  methods: [PUT, PATCH]
sql:
  queries:
    - query: "UPDATE items SET name = ? WHERE id = ?"
      params:
        - "{{ get('name') }}"
        - "{{ get('id') }}"

---
# Delete - DELETE only
actionId: deleteItem
validations:
  methods: [DELETE]
sql:
  queries:
    - query: "DELETE FROM items WHERE id = ?"
      params: ["{{ get('id') }}"]
```

## Route Restrictions

Limit a resource to specific URL routes using `routes:` inside the `validations:` block:

```yaml
# resources/admin-dashboard.yaml
actionId: adminDashboard
validations:
  routes:
    - /admin
    - /admin/*

sql:
  queries:
    - query: "SELECT * FROM admin_stats"
```

### Route Patterns

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `/users` | `/users` | `/users/123` |
| `/users/*` | `/users/123`, `/users/abc` | `/users` |
| `/api/*` | `/api/v1`, `/api/users/123` | `/api` |
| `/api/v1/*` | `/api/v1/users` | `/api/v2/users` |

### Examples

**Version-specific API:**

<div v-pre>

```yaml
# V1 API handler
actionId: apiV1Handler
validations:
  routes: [/api/v1/*]
chat:
  prompt: "V1 API: {{ get('q') }}"

---
# V2 API handler (different model)
actionId: apiV2Handler
validations:
  routes: [/api/v2/*]
chat:
  prompt: "V2 API with enhanced model: {{ get('q') }}"
```

</div>

**Admin vs Public routes:**

```yaml
# Public content
actionId: publicContent
validations:
  routes:
    - /public/*
    - /
sql:
  queries:
    - query: "SELECT * FROM content WHERE is_public = true"

---
# Admin content (requires auth)
actionId: adminContent
validations:
  routes: [/admin/*]
  check:
    - get('Authorization') != ''
  error:
    code: 401
    message: "Authentication required"
sql:
  queries:
    - query: "SELECT * FROM content"
```

## Combining Restrictions

Combine method and route restrictions for precise control:

```yaml
# resources/user-create.yaml
actionId: userCreate
validations:
  methods: [POST]
  routes: [/api/users]

sql:
  queries:
    - query: "INSERT INTO users (name, email) VALUES (?, ?)"
      params:
        - "{{ get('name') }}"
        - "{{ get('email') }}"

---
actionId: userGet
validations:
  methods: [GET]
  routes: [/api/users/*]

sql:
  queries:
    - query: "SELECT * FROM users WHERE id = ?"
      params: ["{{ get('id') }}"]
```

## RESTful API Example

Complete RESTful API using restrictions:

**workflow.yaml:**
```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
name: rest-api
version: "1.0.0"
targetActionId: responseHandler
settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/users
        methods: [GET, POST]
      - path: /api/users/:id
        methods: [GET, PUT, DELETE]
  sqlConnections:
    db: {}  # pool config here; DSN goes in ~/.kdeps/config.yaml sql_connections.db.connection
```

**resources/list-users.yaml:**
```yaml
# resources/list-users.yaml
actionId: listUsers
validations:
  methods: [GET]
  routes: [/api/users]
sql:
  connectionName: db
  queries:
    - query: "SELECT * FROM users LIMIT 100"
```

**resources/create-user.yaml:**
```yaml
# resources/create-user.yaml
actionId: createUser
validations:
  methods: [POST]
  routes: [/api/users]
  required: [name, email]
sql:
  connectionName: db
  queries:
    - query: "INSERT INTO users (name, email) VALUES (?, ?)"
      params:
        - "{{ get('name') }}"
        - "{{ get('email') }}"
```

**resources/get-user.yaml:**
```yaml
# resources/get-user.yaml
actionId: getUser
validations:
  methods: [GET]
  routes: [/api/users/*]
sql:
  connectionName: db
  queries:
    - query: "SELECT * FROM users WHERE id = ?"
      params: ["{{ get('id') }}"]
```

**resources/update-user.yaml:**
```yaml
# resources/update-user.yaml
actionId: updateUser
validations:
  methods: [PUT]
  routes: [/api/users/*]
sql:
  connectionName: db
  queries:
    - query: "UPDATE users SET name = ?, email = ? WHERE id = ?"
      params:
        - "{{ get('name') }}"
        - "{{ get('email') }}"
        - "{{ get('id') }}"
```

**resources/delete-user.yaml:**
```yaml
# resources/delete-user.yaml
actionId: deleteUser
validations:
  methods: [DELETE]
  routes: [/api/users/*]
sql:
  connectionName: db
  queries:
    - query: "DELETE FROM users WHERE id = ?"
      params: ["{{ get('id') }}"]
```

## Best Practices

1. **Be Specific**: Use exact routes when possible rather than wildcards
2. **Combine with [`check`](/reference/glossary#check)**: Add preflight validation for secure endpoints
3. **Handle All Methods**: Ensure all expected methods have corresponding resources
4. **Use [`skip`](/reference/glossary#skip) as Fallback**: For complex routing logic, combine with `skip` conditions

## See Also

- [Validation](/concepts/validation) — Full `validations:` block reference
- [Request Object](/concepts/request-object) — Accessing request data
- [Workflow Configuration](../configuration/workflow) — Route configuration
- [Resources Overview](../resources/overview) — Resource basics
