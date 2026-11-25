# SQL component

The `sql` component runs SQL against a named connection - PostgreSQL, MySQL/MariaDB, SQLite, SQL Server or Oracle - and returns the rows or a change summary.

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

- [SQL component reference](/reference/resources/sql) - every key, option and detail
- [Python resource](/workflow/resources/python) - data processing scripts
- [Components](/agencies/components) - how shipped components load
