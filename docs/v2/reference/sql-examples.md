# SQL Resource Examples

Example queries for the [`sql:` resource](/resources/sql). All examples use parameterized queries to prevent SQL injection.

## User Lookup

```yaml
# resources/user-lookup.yaml
actionId: userLookup
validations:
  check:
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
  timeout: 10s
```

## Analytics Query

```yaml
# resources/analytics.yaml
actionId: analytics
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
  timeout: 60s
```

## Multi-Database Sync

```yaml
# Fetch from source database
actionId: fetchSource
sql:
  connectionName: source_db
  query: "SELECT * FROM products WHERE updated_at > $1"
  params:
    - get('last_sync')
  format: json

---
# Insert into destination database
actionId: syncProducts
requires: [fetchSource]
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

## Search with LLM Enhancement

<div v-pre>

```yaml
# Search database
actionId: searchProducts
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
actionId: enhancedSearch
requires: [searchProducts]
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

## See Also

- [SQL Resource](/resources/sql) - Full sql: reference with transactions, batch ops, connection pooling
- [Python Resource](/resources/python) - Post-process SQL results with pandas
- [Tools Reference](/reference/tools-reference) - Use SQL as an LLM tool
