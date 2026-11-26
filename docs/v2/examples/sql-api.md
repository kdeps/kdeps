# Build a SQL-backed API

Build an API with two endpoints over a SQLite database:
`GET /report` runs an analytics query and returns CSV. `POST /update` applies a
batch of updates in a transaction.

The connection is in `~/.kdeps/config.yaml`:

```yaml
# ~/.kdeps/config.yaml
sql_connections:
  demo:
    connection: "sqlite://./demo.db"   # relative to the directory you run kdeps from
```

## Read next

- [Full walkthrough](/reference/examples/sql-api) - every step, file and command
- [SQL resource](/workflow/resources/sql) - all connection types, formats, transactions
- [Validation and control flow](/workflow/validation) - route and method scoping
