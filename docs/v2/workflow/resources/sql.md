# SQL component

The `sql` component runs SQL against a named connection - PostgreSQL, MySQL/MariaDB, SQLite, SQL Server or Oracle - and returns the rows or a change summary. One statement, or several in a transaction. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool.

## Requirements

The SQL drivers are compiled into kdeps: no Python, no uv, no first-call driver install. SQLite needs nothing else; the other databases need only a reachable server. Each call opens its own connection - there is no pool between calls.

## Basic usage

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

```json
{"id": 7, "name": "Ann", "email": "ann@example.com"}
```

## Connections

Connection strings live in `~/.kdeps/config.yaml`, never in `workflow.yaml`. The component names the connection; the kdeps API reads its string from `config.yaml` in-process, so the password never crosses the request and is not on any command line.

```yaml
# ~/.kdeps/config.yaml
sql_connections:
  main:
    connection: "postgres://user:pass@localhost:5432/myapp"
  local:
    connection: "sqlite:///var/data/app.db"
```

| Database | Connection string |
|---|---|
| PostgreSQL | `postgres://user:pass@host:5432/db` |
| MySQL / MariaDB | `mysql://user:pass@host:3306/db` |
| SQLite | `sqlite:///path/to/file.db`, `sqlite://:memory:` |
| SQL Server | `sqlserver://user:pass@host:1433?database=db` |
| Oracle | `oracle://user:pass@host:1521/service` |

A connection missing from `config.yaml` is prompted for by `kdeps run` (interactive terminals only), or supplied with `KDEPS_SQL_CONNECTIONS_<NAME>_CONNECTION`. See [Interactive setup on first run](/workflow/resources/email#interactive-setup-on-first-run).

## Placeholders

Write placeholders the way you know them - `$1`, `?`, `@p1` or `:1` - and pass the values in `params`, in order. The component rewrites them into the driver's own style, skipping text inside quotes and comments and Postgres `::` casts. `$N`, `@pN` and `:N` refer to the Nth param; each bare `?` takes the next one.

## Results

| Statement | Result |
|---|---|
| `SELECT` returning one row | That row: `{column: value}` |
| `SELECT` returning 0 or several rows | `{rowsAffected: n, data: [rows], columns: [sorted names]}` |
| `SELECT` with `format: json` | The rows as a JSON string |
| `SELECT` with `format: csv` | CSV text, header first (columns sorted) |
| `INSERT` / `UPDATE` / `DELETE` | `{rowsAffected, lastInsertID, success: true}` |
| `transaction: true` | A list: each statement's rows, or its `{rowsAffected, lastInsertID}` |

Dates come back as ISO 8601 strings and decimals as strings. A connection that cannot be opened returns `{error: "failed to get database connection: ..."}`; a failing statement fails the call with `sql: query execution failed: ...` and rolls back.

## Transactions and batches

<div v-pre>

```yaml
# resources/transfer.yaml
actionId: transfer
name: Transfer Funds
component:
  name: sql
  with:
    connectionName: main
    transaction: true                  # all statements commit together, or none do
    queries:
      - query: "UPDATE accounts SET balance = balance - $1 WHERE id = $2"
        params: ["{{ get('amount') }}", "{{ get('from') }}"]
      - query: "UPDATE accounts SET balance = balance + $1 WHERE id = $2"
        params: ["{{ get('amount') }}", "{{ get('to') }}"]
      - query: "INSERT INTO audit (note) VALUES ($1)"
        paramsBatch: "{{ get('notes') }}"   # a list of param lists: the statement runs once per entry
```

</div>

## Inputs

| Input | Default | Description |
|---|---|---|
| `connectionName` | - | Name under `sql_connections` (required) |
| `query` | | The statement |
| `params` | `[]` | Placeholder values, in order |
| `transaction` | `false` | Run `queries` in one transaction |
| `queries` | `[]` | `[{query, params}]` or `[{query, paramsBatch}]` |
| `format` | `''` | `''`/`table` (rows), `json`, `csv` |
| `timeout` | `KDEPS_SQL_TIMEOUT`, else `30s` | Statement timeout |
| `maxRows` | `KDEPS_SQL_MAX_ROWS`, else `1000` | Rows a `SELECT` returns at most |

`config.yaml` `resource_defaults.sql.timeout` and `max_rows` set the two environment variables.

## Security

Pass user input in `params`, never inside the query text:

<div v-pre>

```yaml
query: "SELECT * FROM users WHERE id = $1"     # good: the value never touches the SQL
params: ["{{ get('user_id') }}"]
```

```yaml
query: "SELECT * FROM users WHERE id = {{ get('user_id') }}"   # bad: SQL injection
```

</div>

## How it is built

```text
component: sql -> POST /apis/tools.kdeps.io/v1/tools/sql_ops/call -> rows or change summary
```

One `httpClient` POST carrying the eight inputs to the `sql_ops` tool, then an `interface.output` that decodes the tool's JSON body. The tool reads the named connection from `config.yaml` itself and runs the statements with `database/sql` - the PostgreSQL, MySQL, SQLite, SQL Server and Oracle drivers are compiled in. Placeholder rewriting, result shapes, transactions and the error texts live in the tool. Copy `components/sql/` into your project to change an input default; the SQL itself runs in the kdeps API.

## See also

- [SQL examples](/reference/sql-examples) - user lookup, analytics, multi-database sync, LLM-enhanced search
- [Python resource](/workflow/resources/python) - data processing scripts
- [Components](/agencies/components) - how shipped components load
