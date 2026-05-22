# SQL Resource

The `sql:` resource runs SQL queries against a named connection and returns the result set as the resource's output. Use it to read, write, or transact against any supported database.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). In workflow mode it executes as a DAG step. In agent mode it is auto-registered as a callable tool.

## Basic Usage

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

## Supported Databases

| Database | Connection String Format |
|----------|-------------------------|
| PostgreSQL | `postgres://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| SQLite | `sqlite:///path/to/file.db` or `sqlite:///:memory:` |
| SQL Server | `sqlserver://user:pass@host:1433/db` |
| Oracle | `oracle://user:pass@host:1521/service` |

## Connection Configuration

### Named Connections (Recommended)

Define connections in your workflow:

```yaml
# workflow.yaml
settings:
  sqlConnections:
    main:
      connection: "postgres://user:pass@localhost:5432/myapp"
      pool:
        maxConnections: 10
        minConnections: 2
        maxIdleTime: "30s"
        connectionTimeout: "5s"

    analytics:
      connection: "postgres://user:pass@analytics-db:5432/analytics"
      pool:
        maxConnections: 5
        minConnections: 1
```

Use in resources:

```yaml
# resources/example.yaml
sql:
  connectionName: main  # Reference the named connection
  query: "SELECT * FROM users"
```

### Inline Connection

```yaml
# resources/example.yaml
sql:
  connection: "postgres://user:pass@localhost:5432/myapp"
  query: "SELECT * FROM users"
```

## Query Types

### Simple Query

```yaml
# resources/example.yaml
sql:
  connectionName: main
  query: "SELECT name, email FROM users WHERE active = true"
  format: json
  maxRows: 100
  timeout: 30s
```

### Parameterized Query

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

### Insert / Update / Delete

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

## Batch Operations

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

## Result Formats

### JSON (Default)

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

## Connection Pooling

Configure connection pools in workflow settings:

```yaml
# workflow.yaml
settings:
  sqlConnections:
    main:
      connection: "postgres://user:pass@localhost:5432/myapp"
      pool:
        maxConnections: 20      # Maximum pool size
        minConnections: 5       # Minimum idle connections
        maxIdleTime: "30s"     # Close idle connections after
        connectionTimeout: "5s" # Timeout for acquiring connection
```

## Accessing Results

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

## Best Practices

1. **Use named connections** - Easier to manage and configure pooling
2. **Always use parameterized queries** - Prevent SQL injection
3. **Set appropriate maxRows** - Prevent memory issues
4. **Use transactions for multi-step operations** - Ensure data consistency
5. **Configure connection pooling** - Improve performance under load
6. **Use appropriate timeouts** - Prevent long-running queries from blocking

## Security Notes

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

## See Also

- [SQL Examples](/reference/sql-examples) - User lookup, analytics, multi-database sync, LLM-enhanced search
- [Python Resource](python) -- data processing scripts
- [HTTP Client](http-client) -- external API calls
- [Workflow Configuration](../configuration/workflow) -- connection settings
