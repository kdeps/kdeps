# Build a SQL-backed API

*Applies to workflow mode.*

## Overview

In this tutorial you build an API with two endpoints over a SQLite database:
`GET /report` runs an analytics query and returns CSV; `POST /update` applies a
batch of updates in a transaction.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic SQL

By the end you will be able to:

- Declare a database connection in `workflow.yaml`
- Run a parameterized query with `sql:` and `params:`
- Return results as CSV or JSON with `format:`
- Apply a batch of writes in one transaction with `paramsBatch:`

## Background

The `sql:` resource runs queries against PostgreSQL, MySQL, SQLite, SQL Server,
or Oracle. Parameters use `$1`, `$2`, ... placeholders - never string
interpolation - so input cannot break out of the query. This tutorial uses
SQLite so it runs with no database server.

## Before you start

- kdeps installed (`kdeps --version`).
- The `sqlite3` CLI (for creating the demo database).
- A working directory for the project.

## Step 1: create the project and database

```bash
mkdir sql-api
cd sql-api
mkdir resources

sqlite3 demo.db <<'SQL'
CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT, age INTEGER,
                    status TEXT, created_at TEXT);
INSERT INTO users (email, age, status, created_at) VALUES
  ('a@example.com', 31, 'active',   '2024-03-01'),
  ('b@example.com', 27, 'active',   '2024-03-01'),
  ('c@example.com', 44, 'inactive', '2024-03-02');
SQL
```

## Step 2: declare the connection and routes

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: sql-api
  version: "1.0.0"
  targetActionId: results

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /report
        methods: [GET]
      - path: /update
        methods: [POST]
  sqlConnections:
    demo:
      connection: "sqlite:///./demo.db"   # relative to the workflow directory
```

## Step 3: the analytics query

Create `resources/report.yaml`:

<div v-pre>

```yaml
# resources/report.yaml
actionId: report
name: User report
validations:
  methods: [GET]
  routes: [/report]
sql:
  connectionName: demo
  query: |
    SELECT status,
           COUNT(*)        AS total,
           ROUND(AVG(age)) AS avg_age
    FROM users
    WHERE created_at >= $1
    GROUP BY status
  params:
    - get('since', '2024-01-01')     # ?since=... or default
  format: csv                        # csv, json, or table
  maxRows: 100
  timeout: 30s
```

</div>

## Step 4: the batch update

Create `resources/update.yaml`:

<div v-pre>

```yaml
# resources/update.yaml
actionId: update
name: Batch status update
validations:
  methods: [POST]
  routes: [/update]
sql:
  connectionName: demo
  transaction: true                  # all or nothing
  queries:
    - query: "UPDATE users SET status = $1 WHERE id = $2"
      paramsBatch: "{{ get('changes') }}"   # a list of [status, id] pairs
```

</div>

`paramsBatch` runs the query once per element in the list, inside one
transaction.

## Step 5: combine the results

Create `resources/results.yaml`:

<div v-pre>

```yaml
# resources/results.yaml
actionId: results
name: Combined results
requires: [report, update]
apiResponse:
  success: true
  response:
    report: "{{ get('report') }}"
    update: "{{ get('update') }}"
    at: "{{ info('current_time') }}"
```

</div>

Only the resource matching the request's route runs; the other is skipped.

## Step 6: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

Run the report:

```bash
curl "http://localhost:16395/report?since=2024-03-01" \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Apply updates:

```bash
curl -X POST http://localhost:16395/update \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"changes": [["inactive", 1], ["active", 3]]}'
```

## Summary

You built a SQL-backed API that:

- Declares a SQLite connection in `sqlConnections`
- Runs a parameterized analytics query returning CSV
- Applies a batch of updates in one transaction with `paramsBatch:`
- Routes `GET` and `POST` to different resources

## Next steps

- [SQL resource](/resources/sql) - all connection types, formats, transactions
- [SQL examples](/reference/sql-examples) - joins, pagination, upserts
- [Validation and control flow](/concepts/validation-and-control) - route and method scoping
- [Global config](/configuration/advanced) - keeping connection strings out of `workflow.yaml`
