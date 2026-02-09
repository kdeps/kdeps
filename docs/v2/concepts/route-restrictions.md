# Route and Method Restrictions

KDeps allows you to restrict resources to specific HTTP methods and routes, enabling powerful routing patterns and method-based behavior.

## HTTP Method Restrictions

### restrictToHttpMethods

Limit a resource to specific HTTP methods:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: createUser
run:
  restrictToHttpMethods:
    - POST

  sql:
    connection: primary
    queries:
      - query: "INSERT INTO users (name, email) VALUES (?, ?)"
        params:
          - "{{ get('name') }}"
          - "{{ get('email') }}"
```

When `restrictToHttpMethods` is set:
- Resource only executes for matching HTTP methods
- Mismatched methods skip the resource silently
- Multiple methods can be specified

### Common Patterns

**CRUD Operations:**

```yaml
# Create - POST only
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: createItem
run:
  restrictToHttpMethods: [POST]
  sql:
    queries:
      - query: "INSERT INTO items (name) VALUES (?)"
        params: ["{{ get('name') }}"]

---
# Read - GET only
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: getItem
run:
  restrictToHttpMethods: [GET]
  sql:
    queries:
      - query: "SELECT * FROM items WHERE id = ?"
        params: ["{{ get('id') }}"]

---
# Update - PUT/PATCH
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: updateItem
run:
  restrictToHttpMethods: [PUT, PATCH]
  sql:
    queries:
      - query: "UPDATE items SET name = ? WHERE id = ?"
        params:
          - "{{ get('name') }}"
          - "{{ get('id') }}"

---
# Delete - DELETE only
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: deleteItem
run:
  restrictToHttpMethods: [DELETE]
  sql:
    queries:
      - query: "DELETE FROM items WHERE id = ?"
        params: ["{{ get('id') }}"]
```

## Route Restrictions

### restrictToRoutes

Limit a resource to specific URL routes:

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: adminDashboard
run:
  restrictToRoutes:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: apiV1Handler
run:
  restrictToRoutes:
    - /api/v1/*
  chat:
    model: llama3.2:1b
    prompt: "V1 API: {{ get('q') }}"

---
# V2 API handler (different model)
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: apiV2Handler
run:
  restrictToRoutes:
    - /api/v2/*
  chat:
    model: llama3.2:3b
    prompt: "V2 API with enhanced model: {{ get('q') }}"
```

</div>

**Admin vs Public routes:**

```yaml
# Public content
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: publicContent
run:
  restrictToRoutes:
    - /public/*
    - /
  sql:
    queries:
      - query: "SELECT * FROM content WHERE is_public = true"

---
# Admin content (requires auth)
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: adminContent
run:
  restrictToRoutes:
    - /admin/*
  preflightCheck:
    validations:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: userCreate
run:
  # Only POST requests to /api/users
  restrictToHttpMethods: [POST]
  restrictToRoutes:
    - /api/users

  sql:
    queries:
      - query: "INSERT INTO users (name, email) VALUES (?, ?)"
        params:
          - "{{ get('name') }}"
          - "{{ get('email') }}"

---
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: userGet
run:
  # Only GET requests to /api/users/*
  restrictToHttpMethods: [GET]
  restrictToRoutes:
    - /api/users/*

  sql:
    queries:
      - query: "SELECT * FROM users WHERE id = ?"
        params: ["{{ get('id') }}"]
```

## RESTful API Example

Complete RESTful API using restrictions:

**workflow.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: rest-api
  version: "1.0.0"
  targetActionId: responseHandler
settings:
  apiServerMode: true
  apiServer:
    portNum: 16395
    routes:
      - path: /api/users
        methods: [GET, POST]
      - path: /api/users/:id
        methods: [GET, PUT, DELETE]
  sqlConnections:
    db:
      connection: "sqlite://./users.db"
```

**resources/list-users.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: listUsers
run:
  restrictToHttpMethods: [GET]
  restrictToRoutes: [/api/users]
  sql:
    connection: db
    queries:
      - query: "SELECT * FROM users LIMIT 100"
```

**resources/create-user.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: createUser
run:
  restrictToHttpMethods: [POST]
  restrictToRoutes: [/api/users]
  validation:
    required: [name, email]
  sql:
    connection: db
    queries:
      - query: "INSERT INTO users (name, email) VALUES (?, ?)"
        params:
          - "{{ get('name') }}"
          - "{{ get('email') }}"
```

**resources/get-user.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: getUser
run:
  restrictToHttpMethods: [GET]
  restrictToRoutes: [/api/users/*]
  sql:
    connection: db
    queries:
      - query: "SELECT * FROM users WHERE id = ?"
        params: ["{{ get('id') }}"]
```

**resources/update-user.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: updateUser
run:
  restrictToHttpMethods: [PUT]
  restrictToRoutes: [/api/users/*]
  sql:
    connection: db
    queries:
      - query: "UPDATE users SET name = ?, email = ? WHERE id = ?"
        params:
          - "{{ get('name') }}"
          - "{{ get('email') }}"
          - "{{ get('id') }}"
```

**resources/delete-user.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: deleteUser
run:
  restrictToHttpMethods: [DELETE]
  restrictToRoutes: [/api/users/*]
  sql:
    connection: db
    queries:
      - query: "DELETE FROM users WHERE id = ?"
        params: ["{{ get('id') }}"]
```

**resources/response.yaml:**
```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: responseHandler
  requires:
    - listUsers
    - createUser
    - getUser
    - updateUser
    - deleteUser
run:
  expr:
    # Determine which resource ran based on method and route
    - set('result',
        default(get('listUsers'),
        default(get('createUser'),
        default(get('getUser'),
        default(get('updateUser'),
        get('deleteUser'))))))
  apiResponse:
    success: true
    response:
      data: get('result')
```

## Best Practices

1. **Be Specific**: Use exact routes when possible rather than wildcards
2. **Combine with Validation**: Add `preflightCheck` or `validation` for secure endpoints
3. **Handle All Methods**: Ensure all expected methods have corresponding resources
4. **Use skipCondition as Fallback**: For complex routing logic, combine with skipCondition

## See Also

- [Validation](validation) - Input validation
- [Request Object](request-object) - Accessing request data
- [Workflow Configuration](../configuration/workflow) - Route configuration
- [Resources Overview](../resources/overview) - Resource basics
