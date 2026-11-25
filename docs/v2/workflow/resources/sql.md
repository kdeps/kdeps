# SQL component

The `sql` component runs SQL against a named connection - PostgreSQL, MySQL/MariaDB, SQLite, SQL Server or Oracle - and returns the rows or a change summary. One statement, or several in a transaction. It is a YAML component from the `components` asset set. Call it with a `component:` resource, no install step.

<div v-pre>

```yaml
# resources/user.yaml
actionId: user
name: Load User
component:
  name: sql
  with:
    connectionName: main                       # sql_connections.main in ~/.kdeps/config.yaml
    query: "SELECT id, name, email FROM users WHERE id = $1"
    params:
      - "{{ get('user_id') }}"                 # a lone {{ }} keeps its type: a number stays a number
```

</div>

## Read next

- [SQL component reference](/reference/resources/sql) - basic usage, connections, placeholders, results, transactions and batches, inputs, security, how it is built
- [SQL examples](/reference/sql-examples) - user lookup, analytics, multi-database sync, LLM-enhanced search
- [Python resource](/workflow/resources/python) - data processing scripts
- [Components](/agencies/components) - how shipped components load
