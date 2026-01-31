# SQL Resource

The SQL resource enables database queries with support for multiple database types, connection pooling, transactions, and batch operations.

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: sqlResource
  name: Database Query

run:
  sql:
    connectionName: main
    query: "SELECT * FROM users WHERE id = $1"
    params:
      - get('user_id')
    timeoutDuration: 30s
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
run:
  sql:
    connectionName: main  # Reference the named connection
    query: "SELECT * FROM users"
```

### Inline Connection

```yaml
run:
  sql:
    connection: "postgres://user:pass@localhost:5432/myapp"
    query: "SELECT * FROM users"
```

## Query Types

### Simple Query

```yaml
run:
  sql:
    connectionName: main
    query: "SELECT name, email FROM users WHERE active = true"
    format: json
    maxRows: 100
    timeoutDuration: 30s
```

### Parameterized Query

<div v-pre>

```yaml
run:
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
run:
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
run:
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
run:
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
run:
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
run:
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
run:
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

## Examples

### User Lookup

```yaml
metadata:
  actionId: userLookup

run:
  preflightCheck:
    validations:
      - get('user_id') != ''
    error:
      code: 400
      message: User ID is required

  sql:
    connectionName: main
    query: |
      SELECT id, name, email, created_at
      FROM users
      WHERE id = $1
    params:
      - get('user_id')
    format: json
    timeoutDuration: 10s
```

### Analytics Query

```yaml
metadata:
  actionId: analytics

run:
  sql:
    connectionName: analytics
    query: |
      SELECT
        DATE(created_at) as date,
        COUNT(*) as total_orders,
        SUM(amount) as revenue,
        AVG(amount) as avg_order
      FROM orders
      WHERE created_at >= $1
        AND created_at < $2
      GROUP BY DATE(created_at)
      ORDER BY date DESC
    params:
      - get('start_date', '2024-01-01')
      - get('end_date', '2024-12-31')
    format: csv
    maxRows: 365
    timeoutDuration: 60s
```

### Multi-Database Workflow

```yaml
# Fetch from source database
metadata:
  actionId: fetchSource

run:
  sql:
    connectionName: source_db
    query: "SELECT * FROM products WHERE updated_at > $1"
    params:
      - get('last_sync')
    format: json

---
# Insert into destination database
metadata:
  actionId: syncProducts
  requires: [fetchSource]

run:
  sql:
    connectionName: dest_db
    transaction: true
    queries:
      - query: |
          INSERT INTO products (id, name, price, updated_at)
          VALUES ($1, $2, $3, $4)
          ON CONFLICT (id) DO UPDATE SET
            name = EXCLUDED.name,
            price = EXCLUDED.price,
            updated_at = EXCLUDED.updated_at
<div v-pre>
        paramsBatch: "{{ get('fetchSource') }}"
</div>


```

### Search with LLM Enhancement

<div v-pre>

```yaml
# Search database
metadata:
  actionId: searchProducts

run:
  sql:
    connectionName: main
    query: |
      SELECT id, name, description, price
      FROM products
      WHERE name ILIKE $1 OR description ILIKE $1
      LIMIT 10
    params:
      - "'%' || get('q') || '%'"
    format: json

---
# Enhance with LLM
metadata:
  actionId: enhancedSearch
  requires: [searchProducts]

run:
  chat:
    model: llama3.2:1b
    prompt: |
      User searched for: {{ get('q') }}

      Found products:
      {{ get('searchProducts') }}

      Provide a helpful summary and recommendations.
    jsonResponse: true
    jsonResponseKeys:
      - summary
      - recommended
      - suggestions
```

</div>

## Accessing Results

```yaml
# In another resource
metadata:
  requires: [sqlResource]

run:
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
# Good - parameterized
query: "SELECT * FROM users WHERE id = $1"
params:
  - get('user_id')

**Bad - SQL injection risk**
<span v-pre>`query: "SELECT * FROM users WHERE id = {{ get('user_id') }}"`</span>
```

## Next Steps

- [Python Resource](python) - Data processing scripts
- [HTTP Client](http-client) - External API calls
- [Workflow Configuration](../configuration/workflow) - Connection settings
