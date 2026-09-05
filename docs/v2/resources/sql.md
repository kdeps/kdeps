# SQL resource

The `sql:` resource runs SQL queries against a named connection and returns the result set as the resource's output. Use it to read, write, or transact against any supported database.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool. Not available in [WASM web apps](/deployment/wasm) - the browser cannot open a database socket.

## Basic usage

<div v-pre>

```yaml
# resources/sql.yaml

actionId: sqlResource
name: Database Query
sql:
  connectionName: main
  query: "SELECT * FROM users WHERE id = $1"
  params:
    - get('user_id')
  timeout: 30s
```

</div>

## Supported databases

| Database | Connection String Format |
|----------|-------------------------|
| PostgreSQL | `postgres://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| SQLite | `sqlite:///path/to/file.db` or `sqlite:///:memory:` |
| SQL Server | `sqlserver://user:pass@host:1433/db` |
| Oracle | `oracle://user:pass@host:1521/service` |

## Connection configuration

Connection strings (DSNs) live in `~/.kdeps/config.yaml` - never in `workflow.yaml`, which is version-controlled. Pool configuration lives in `workflow.yaml`.

`~/.kdeps/config.yaml` - credentials:

```yaml
sql_connections:
  main:
    connection: "postgres://user:pass@localhost:5432/myapp"
  analytics:
    connection: "postgres://user:pass@analytics-db:5432/analytics"
```

If `connectionName` refers to a connection missing from `config.yaml`, `kdeps run` prompts for the DSN at startup and saves it (interactive terminals only; skipped in CI/pipes, where the "connection not found" error surfaces instead). See [Interactive setup on first run](/resources/messaging/email#interactive-setup-on-first-run).

`workflow.yaml` - pool config:

```yaml
settings:
  sqlConnections:
    main:
      pool:
        maxConnections: 10
        minConnections: 2
        maxIdleTime: "30s"
        connectionTimeout: "5s"
    analytics:
      pool:
        maxConnections: 5
        minConnections: 1
```

Use in resources:

```yaml
# resources/example.yaml
sql:
  connectionName: main  # must match key in sql_connections in ~/.kdeps/config.yaml
  query: "SELECT * FROM users"
```

## Query types

### Simple query

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: "SELECT name, email FROM users WHERE active = true"
  format: json
  maxRows: 100
  timeout: 30s
```

### Parameterized query

<div v-pre>

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: |
    SELECT * FROM orders
    WHERE customer_id = $1
      AND created_at >= $2
      AND status = $3
    ORDER BY created_at DESC
    LIMIT $4
  params:
    - get('customer_id')
    - get('start_date')
    - get('status', 'active')  # With default
    - get('limit', '100')
  format: json
```

</div>

### Insert / update / delete

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: |
    INSERT INTO users (name, email, created_at)
    VALUES ($1, $2, NOW())
    RETURNING id
  params:
    - get('name')
    - get('email')
```

## Transactions

Execute multiple queries in a transaction:

```yaml
# resources/example.yaml
sql:
  connectionName: main
  transaction: true
  queries:
    - query: "UPDATE accounts SET balance = balance - $1 WHERE id = $2"
      params:
        - get('amount')
        - get('from_account')

    - query: "UPDATE accounts SET balance = balance + $1 WHERE id = $2"
      params:
        - get('amount')
        - get('to_account')

    - query: |
        INSERT INTO transactions (from_id, to_id, amount, created_at)
        VALUES ($1, $2, $3, NOW())
      params:
        - get('from_account')
        - get('to_account')
        - get('amount')
```

If any query fails, the entire transaction is rolled back.

## Batch operations

Process multiple records efficiently:

<div v-pre>

```yaml
# resources/example.yaml
sql:
  connectionName: main
  transaction: true
  queries:
    - query: |
        INSERT INTO products (name, price, category)
        VALUES ($1, $2, $3)
      paramsBatch: "{{ get('products') }}"
```

</div>

Where `products` is an array of parameter arrays:
```json
[
  ["Product A", 19.99, "electronics"],
  ["Product B", 29.99, "electronics"],
  ["Product C", 9.99, "accessories"]
]
```

## Result formats

### JSON (default)

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: "SELECT id, name, email FROM users"
  format: json
```

Output:
```json
[
  {"id": 1, "name": "Alice", "email": "alice@example.com"},
  {"id": 2, "name": "Bob", "email": "bob@example.com"}
]
```

### CSV

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: "SELECT id, name, email FROM users"
  format: csv
```

Output:
```csv
id,name,email
1,Alice,alice@example.com
2,Bob,bob@example.com
```

### Table

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: "SELECT id, name, email FROM users"
  format: table
```

Output:
```
+----+-------+-------------------+
| id | name  | email             |
+----+-------+-------------------+
| 1  | Alice | alice@example.com |
| 2  | Bob   | bob@example.com   |
+----+-------+-------------------+
```

## Connection pooling

`workflow.yaml`'s `sqlConnections` entry holds pool config only - no
`connection:` field. The DSN itself always comes from `sql_connections` in
`~/.kdeps/config.yaml` (see [Connection Configuration](#connection-configuration)
above); a connection string here would be silently ignored.

```yaml
# workflow.yaml
settings:
  sqlConnections:
    main:
      pool:
        maxConnections: 20      # Maximum pool size
        minConnections: 5       # Minimum idle connections
        maxIdleTime: "30s"     # Close idle connections after
        connectionTimeout: "5s" # Timeout for acquiring connection
```

## Accessing results

```yaml
# In another resource
requires: [sqlResource]
apiResponse:
  response:
    # Full result set
    users: get('sqlResource')

    # First row
    first_user: get('sqlResource')[0]

    # Specific field from first row
    first_name: get('sqlResource')[0].name
```

## Best practices

1. **Use named connections** - Easier to manage and configure pooling
2. **Always use parameterized queries** - Prevent SQL injection
3. **Set appropriate maxRows** - Prevent memory issues
4. **Use transactions for multi-step operations** - Ensure data consistency
5. **Configure connection pooling** - Improve performance under load
6. **Use appropriate timeouts** - Prevent long-running queries from blocking

## Security notes

- Never interpolate user input directly into queries
- Always use parameterized queries (`$1`, `$2`, etc.)
- Store database credentials in environment variables
- Use connection strings from environment in production

```yaml
# Good -- parameterized; user input never touches the query string
query: "SELECT * FROM users WHERE id = $1"
params:
  - get('user_id')
```

**Bad - SQL injection risk**

<div v-pre>

```yaml
# resources/example.yaml
query: "SELECT * FROM users WHERE id = {{ get('user_id') }}"
```

</div>

## See also

- [SQL examples](/reference/sql-examples) - User lookup, analytics, multi-database sync, LLM-enhanced search
- [Python resource](/resources/scripting/python) - data processing scripts
- [HTTP client](/resources/web/http-client) - external API calls
- [Workflow configuration](/configuration/workflow) - connection settings
