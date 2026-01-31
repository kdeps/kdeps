# Multi-Database Workflow

This tutorial demonstrates how to work with multiple databases in a single KDeps workflow, using named connections, connection pooling, and cross-database operations.

## Prerequisites

- KDeps installed (see [Installation](../getting-started/installation.md))
- PostgreSQL and/or MySQL databases set up
- Basic understanding of SQL

## Overview

KDeps supports multiple database connections in a single workflow. You can:
- Define named connections for different databases
- Use connection pooling for performance
- Query multiple databases in the same workflow
- Perform cross-database operations

## Step 1: Configure Multiple Connections

Create `workflow.yaml` with multiple database connections:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: multi-database
  description: Workflow with multiple database connections
  version: "1.0.0"
  targetActionId: results

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 3000
    routes:
      - path: /api/v1/data
        methods: [GET, POST]

  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"

  sqlConnections:
    # PostgreSQL connection for analytics
    analytics:
      connection: "postgres://user:pass@localhost:5432/analytics"
      pool:
        maxConnections: 10
        minConnections: 2
    
    # MySQL connection for inventory
    inventory:
      connection: "mysql://user:pass@localhost:3306/inventory"
      pool:
        maxConnections: 5
        minConnections: 1
    
    # SQLite for local cache
    cache:
      connection: "sqlite://./cache.db"
```

## Step 2: Using Named Connections

Reference connections by name in SQL resources:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: analyticsQuery
  name: Analytics Query

run:
  sql:
    connectionName: analytics  # Use named connection
    query: |
      SELECT 
        date,
        COUNT(*) as total_users,
        COUNT(DISTINCT email) as unique_emails,
        AVG(age) as avg_age
      FROM users
      WHERE created_at >= NOW() - INTERVAL '7 days'
      GROUP BY date
      ORDER BY date DESC
    format: csv
```

## Step 3: Querying Multiple Databases

Query different databases in sequence:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: userData
  name: User Data
  requires:
    - analyticsQuery
    - inventoryQuery

run:
  apiResponse:
    success: true
    response:
      analytics: get('analyticsQuery')
      inventory: get('inventoryQuery')
```

With separate resources:

```yaml
# resources/analytics.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: analyticsQuery
  name: Analytics Query

run:
  sql:
    connectionName: analytics
    query: <span v-pre>"SELECT * FROM user_stats WHERE date = {{ get('date') }}"</span>


---
# resources/inventory.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: inventoryQuery
  name: Inventory Query

run:
  sql:
    connectionName: inventory
    query: "SELECT * FROM products WHERE status = 'active'"
```

## Step 4: Cross-Database Operations

Combine data from multiple databases:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: combinedData
  name: Combined Data
  requires:
    - analyticsQuery
    - inventoryQuery

run:
  python:
    script: |
      import json
      
      # Get data from both databases
      analytics = get('analyticsQuery')
      inventory = get('inventoryQuery')
      
      # Combine and process
      result = {
          'user_count': len(analytics),
          'product_count': len(inventory),
          'analytics': analytics,
          'inventory': inventory
      }
      
      return result
```

## Step 5: Batch Operations Across Databases

Perform batch updates on multiple databases:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: batchUpdate
  name: Batch Update

run:
  sql:
    connectionName: analytics
    query: |
      UPDATE users 
      SET status = $1, updated_at = NOW()
      WHERE id = $2
    paramsBatch:
      - ["active", 123]
      - ["inactive", 456]
      - ["pending", 789]
    transaction: true
```

## Step 6: Transaction Management

Use transactions for atomic operations:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: transactionalUpdate
  name: Transactional Update

run:
  sql:
    connectionName: analytics
    transaction: true
    queries:
      - query: |
          INSERT INTO orders (user_id, total) 
          VALUES ($1, $2)
        params: [get('user_id'), get('total')]
      - query: |
          UPDATE users 
          SET last_order_at = NOW()
          WHERE id = $1
        params: [get('user_id')]
```

## Complete Example

Here's a complete workflow that demonstrates multi-database operations:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: multi-database-demo
  version: "1.0.0"
  targetActionId: results

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 3000
    routes:
      - path: /api/v1/data
        methods: [GET, POST]

  sqlConnections:
    analytics:
      connection: "postgres://user:pass@localhost:5432/analytics"
      pool:
        maxConnections: 10
    inventory:
      connection: "mysql://user:pass@localhost:3306/inventory"
      pool:
        maxConnections: 5

---
# resources/analytics.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: analyticsQuery
  name: Analytics Query

run:
  sql:
    connectionName: analytics
    query: |
      SELECT date, COUNT(*) as users
      FROM users
      GROUP BY date
      ORDER BY date DESC
    format: csv

---
# resources/inventory.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: inventoryQuery
  name: Inventory Query

run:
  sql:
    connectionName: inventory
    query: |
      SELECT name, quantity, price
      FROM products
      WHERE status = 'active'
    format: json

---
# resources/results.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: results
  name: Results
  requires:
    - analyticsQuery
    - inventoryQuery

run:
  apiResponse:
    success: true
    response:
      analytics: get('analyticsQuery')
      inventory: get('inventoryQuery')
      timestamp: info('now')
```

## Connection Pooling

Configure connection pools for better performance:

```yaml
sqlConnections:
  analytics:
    connection: "postgres://user:pass@localhost:5432/analytics"
    pool:
      maxConnections: 20      # Maximum connections
      minConnections: 5       # Minimum connections
      maxIdleTime: "30m"      # Close idle connections
      maxLifetime: "1h"       # Recycle connections
```

### Pool Settings

- **maxConnections**: Maximum number of connections in the pool
- **minConnections**: Minimum number of connections to maintain
- **maxIdleTime**: Close connections idle for this duration
- **maxLifetime**: Recycle connections after this duration

## Supported Databases

KDeps supports multiple database types:

| Database | Connection String Format |
|----------|-------------------------|
| PostgreSQL | `postgres://user:pass@host:port/db` |
| MySQL | `mysql://user:pass@host:port/db` |
| SQLite | `sqlite://./path/to/db.db` |
| SQL Server | `sqlserver://user:pass@host:port?database=db` |

## Best Practices

1. **Use Named Connections**: Name connections for clarity and reusability
2. **Configure Pools**: Set appropriate pool sizes based on workload
3. **Use Transactions**: Enable transactions for atomic operations
4. **Handle Errors**: Add error handling for database operations
5. **Optimize Queries**: Use indexes and efficient queries

## Error Handling

Handle database errors gracefully:

```yaml
run:
  sql:
    connectionName: analytics
    query: "SELECT * FROM users WHERE id = $1"
    params: [get('user_id')]
  onError:
    apiResponse:
      success: false
      response:
        error: "Database query failed"
        message: get('error')
```

## Performance Tips

1. **Connection Pooling**: Use connection pools to reuse connections
2. **Batch Operations**: Use `paramsBatch` for multiple similar queries
3. **Transactions**: Group related queries in transactions
4. **Indexes**: Ensure proper database indexes
5. **Query Optimization**: Use efficient SQL queries

## Troubleshooting

### Connection Errors

- Verify connection strings are correct
- Check database is running and accessible
- Ensure credentials are valid
- Check network connectivity

### Pool Exhaustion

- Increase `maxConnections` if needed
- Check for connection leaks
- Monitor connection usage

### Transaction Errors

- Ensure `transaction: true` is set
- Check for deadlocks
- Verify transaction isolation level

## Next Steps

- **Advanced SQL**: Learn about [SQL resource](../resources/sql) features
- **Batch Processing**: Use [items iteration](../concepts/items) for large datasets
- **Error Handling**: Implement comprehensive error handling
- **Performance**: Optimize queries and connection pools

## Related Documentation

- [SQL Resource](../resources/sql) - Complete SQL configuration reference
- [Workflow Configuration](../configuration/workflow) - Connection settings
- [Python Resource](../resources/python) - Data processing with Python
- [Unified API](../concepts/unified-api) - Accessing data across resources
