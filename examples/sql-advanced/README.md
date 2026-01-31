# Advanced SQL Operations Example

This example demonstrates advanced SQL features in KDeps v2:

## Features Demonstrated

### 🔄 Batch Operations (`paramsBatch`)
- Execute the same query multiple times with different parameter sets
- Atomic transactions for batch updates

### 📊 Result Formatting
- CSV output format for analytics data
- JSON format (default) for structured data

### 🏗️ Named Connections
- Reusable database connection configurations
- Connection pooling settings per connection

### 🛡️ Transaction Support
- Multi-query atomic operations
- Rollback on any failure

## API Endpoints

### GET /api/v1/sql-demo
Returns user analytics in CSV format:
```csv
date,total_users,unique_emails,avg_age
2024-01-15,1250,1180,28.5
2024-01-14,1180,1120,27.8
...
```

### POST /api/v1/sql-demo
Performs batch user updates:

Request body:
```json
{
  "user_updates": [
    ["active", 123],
    ["inactive", 456],
    ["pending", 789]
  ]
}
```

Response:
```json
{
  "success": true,
  "data": {
    "analytics": "date,total_users,unique_emails,avg_age\n...",
    "batch_results": [
      [{"rows_affected": 1}],
      [{"rows_affected": 1}],
      [{"rows_affected": 1}]
    ],
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Database Setup

### PostgreSQL (Analytics)
```sql
CREATE DATABASE analytics;
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    age INTEGER,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert sample data
INSERT INTO users (email, age, status) VALUES
('user1@example.com', 25, 'active'),
('user2@example.com', 30, 'active'),
('user3@example.com', 35, 'inactive');
```

### MySQL (Inventory)
```sql
CREATE DATABASE inventory;
-- Add your inventory tables here
```

## Running the Example

1. Start databases and update connection strings in `workflow.yaml`
2. Run the workflow:
```bash
kdeps run workflow.yaml --dev
```
3. Test the endpoints:
```bash
# Get analytics (CSV)
curl http://localhost:3000/api/v1/sql-demo

# Batch update users
curl -X POST http://localhost:3000/api/v1/sql-demo \
  -H "Content-Type: application/json" \
  -d '{"user_updates": [["active", 1], ["inactive", 2]]}'
```

## Connection Configuration

The example uses named connections for better organization:

```yaml
sqlConnections:
  analytics:
    connection: "postgres://user:pass@localhost:5432/analytics"
    pool:
      maxConnections: 10
      minConnections: 2
  inventory:
    connection: "mysql://user:pass@localhost:3306/inventory"
    pool:
      maxConnections: 5
      minConnections: 1
```

This allows resources to reference connections by name instead of duplicating connection strings.
