# Chapter 7: Data Resources

AI workflows do not exist in isolation. They read from databases, call external APIs, run shell commands, and execute Python scripts. The resources in this chapter — `sql:`, `httpClient:`, `python:`, and `exec:` — are the connective tissue between your LLM calls and the rest of your infrastructure.

## SQL Resource

The `sql:` resource runs a query against a named database connection and stores the result set as the resource's output.

### Basic Usage

```yaml
# resources/get-user.yaml
actionId: getUser
requires: [validate]
sql:
  connectionName: main
  query: "SELECT id, name, email FROM users WHERE id = $1"
  params:
    - get('userId')
```

The result is stored as a JSON array of row objects. Downstream resources access it with `get('getUser')`.

```yaml
# reads the first row's name field
after:
  - set('userName', get('getUser')[0].name)
```

### Connection Configuration

SQL connection strings (DSNs) live in `~/.kdeps/config.yaml` - never in `workflow.yaml`, which is version-controlled. Pool settings live in `workflow.yaml`.

`~/.kdeps/config.yaml`:

```yaml
sql_connections:
  main:
    connection: "postgres://user:pass@localhost:5432/mydb"
  analytics:
    connection: "postgres://readonly@analytics-db:5432/warehouse"
  cache:
    connection: "sqlite:///data/cache.db"
```

`workflow.yaml` (pool settings only - no credentials):

```yaml
settings:
  sqlConnections:
    main:
      pool:
        maxConnections: 20      # max pool size
        minConnections: 5       # minimum idle connections
        maxIdleTime: "30s"      # close idle connections after this
        connectionTimeout: "5s" # timeout for acquiring a connection
```

The `connectionName` in a resource must match a key in `sql_connections` in `~/.kdeps/config.yaml`. Use environment variable substitution in the DSN to avoid hardcoding credentials:

```yaml
# ~/.kdeps/config.yaml
sql_connections:
  main:
    connection: "${DATABASE_URL}"
```

**Supported databases:** PostgreSQL, MySQL, SQLite, SQL Server, Oracle, and any `database/sql`-compatible driver.

### Result Formats

By default the result is a JSON array. You can request other formats:

```yaml
sql:
  connectionName: main
  query: "SELECT name, email FROM users"
  format: json    # default: array of row objects
  # format: csv   # comma-separated with header row
  # format: table # ASCII table for debugging
  maxRows: 100    # cap results; prevents unbounded memory use
```

### Multiple Queries and Transactions

```yaml
# resources/audit-insert.yaml
actionId: auditInsert
requires: [processResult]
sql:
  connectionName: main
  transaction: true    # explicit; default for queries: list
  queries:
    - query: "INSERT INTO results (input, output, created_at) VALUES ($1, $2, NOW())"
      params:
        - get('input')
        - get('processResult')
    - query: "UPDATE usage_stats SET call_count = call_count + 1 WHERE workflow = $1"
      params:
        - "my-agent"
```

With `transaction: true`, all queries run in a single database transaction. If any query fails, all are rolled back.

### Batch Inserts

Use `paramsBatch:` to insert multiple rows in one resource:

```yaml
sql:
  connectionName: main
  transaction: true
  queries:
    - query: "INSERT INTO products (name, price, category) VALUES ($1, $2, $3)"
      paramsBatch: "&#123;&#123; get('products') &#125;&#125;"
```

`paramsBatch` expects an array of parameter arrays:
```json
[
  ["Widget A", 19.99, "tools"],
  ["Widget B", 29.99, "tools"]
]
```

Each sub-array is bound as one execution of the query. The whole batch runs inside the transaction.

### Parameterized Queries

Always use parameterized queries. Never interpolate user input directly into SQL strings.

T> **Correct:**
T> ```yaml
T> sql:
T>   query: "SELECT * FROM users WHERE email = $1"
T>   params:
T>     - get('email')
T> ```

W> **Never do this:**
W> ```yaml
W> sql:
W>   query: "SELECT * FROM users WHERE email = '&#123;&#123; get('email') &#125;&#125;'"  # SQL injection risk
W> ```

The expression `get('email')` in `params:` is passed as a bound parameter by the driver. The expression in the query string would be string-interpolated before the driver sees it — classic injection vector.

## HTTP Client Resource

The `httpClient:` resource makes an outbound HTTP request and stores the parsed response body as output. JSON responses are parsed automatically; other content types are stored as strings.

### Basic Usage

```yaml
# resources/fetch-weather.yaml
actionId: fetchWeather
httpClient:
  method: GET
  url: "https://api.openweathermap.org/data/2.5/weather"
  queryParams:
    q: "&#123;&#123; get('city') &#125;&#125;"
    appid: "&#123;&#123; env('OPENWEATHER_API_KEY') &#125;&#125;"
    units: metric
```

### Full Configuration Reference

```yaml
httpClient:
  method: GET                    # GET, POST, PUT, PATCH, DELETE
  url: "https://api.example.com/&#123;&#123; get('id') &#125;&#125;"
  headers:
    Authorization: "Bearer &#123;&#123; get('token') &#125;&#125;"
    Content-Type: application/json
  data:                          # request body — serialised as JSON
    key: value
  timeout: 30s                   # hard stop; returns error, does not retry

  # Retry on transient failures
  retry:
    maxAttempts: 3               # total attempts including the first
    backoff: 1s                  # initial wait; doubles on each retry
    maxBackoff: 30s              # ceiling on the retry wait
    retryOn: [500, 502, 503, 504, 429]

  # Response caching — presence of the cache: block enables it
  cache:
    ttl: 5m                      # cache lifetime
    key: "custom-cache-key"      # optional; defaults to the URL

  followRedirects: true          # set false to stop at 3xx responses

  connectionName: my-api         # optional: named connection from http_connections in ~/.kdeps/config.yaml

  tls:
    insecureSkipVerify: false    # never true in production
    certFile: "/path/to/cert.pem"
    keyFile: "/path/to/key.pem"
    caFile: "/path/to/ca.pem"
```

`data:` accepts a map that is JSON-encoded before sending. Use it for POST/PUT/PATCH bodies.

### Reading the Response

After execution, the output is the parsed response body. For JSON APIs:

```yaml
# reads a field from the JSON response
after:
  - set('temperature', get('fetchWeather').main.temp)
  - set('city_name', get('fetchWeather').name)
```

For non-JSON responses, the output is the raw response body as a string.

### Retries

Use `retry:` to automatically retry on transient failures:

```yaml
httpClient:
  url: "https://unreliable-api.example.com/data"
  retry:
    maxAttempts: 3       # total attempts including the first
    backoff: 1s          # initial wait; doubles on each retry (1s, 2s, 4s...)
    maxBackoff: 30s      # ceiling on the exponential wait
    retryOn: [500, 502, 503, 504, 429]   # status codes that trigger a retry
```

The default is no retries. Set `retry.maxAttempts: 3` for any external service that occasionally flakes.

### Authentication

HTTP authentication credentials live in `~/.kdeps/config.yaml` under `http_connections`, not on the resource. Reference the connection by name with `connectionName:`.

`~/.kdeps/config.yaml`:

```yaml
http_connections:
  stripe:
    auth:
      type: bearer
      token: "${STRIPE_SECRET_KEY}"
  internal-api:
    auth:
      type: basic
      username: "${SVC_USER}"
      password: "${SVC_PASS}"
  my-service:
    auth:
      type: api_key
      key: "X-API-Key"
      value: "${MY_API_KEY}"
    proxy: "http://proxy.internal:8080"
```

Reference in the resource:

```yaml
httpClient:
  method: POST
  url: "https://api.stripe.com/v1/charges"
  connectionName: stripe
```

For one-off requests without a named connection, set the `Authorization` header directly:

```yaml
httpClient:
  headers:
    Authorization: "Bearer &#123;&#123; env('API_TOKEN') &#125;&#125;"
```

### Response Caching

Cache responses to avoid redundant API calls:

```yaml
httpClient:
  method: GET
  url: "https://api.example.com/config"
  cache:
    ttl: 5m              # cache for 5 minutes
    key: "global-config" # optional; defaults to the URL
```

Cache key defaults to the request URL if `key:` is omitted.

### TLS and Proxy

For internal services with custom certificates, use `tls:` on the resource:

```yaml
httpClient:
  url: "https://internal.example.com"
  tls:
    certFile: "/certs/client.pem"
    keyFile: "/certs/client-key.pem"
    caFile: "/certs/ca.pem"
    insecureSkipVerify: false    # never true in production
```

Proxy settings go in `http_connections` in `~/.kdeps/config.yaml`:

```yaml
# ~/.kdeps/config.yaml
http_connections:
  proxied:
    proxy: "http://proxy.internal:8080"
```

Then reference with `connectionName: proxied` on the resource.

### Accessing Response Details

By default `get('resourceId')` returns the parsed response body. Resource-specific accessors give access to lower-level response data:

```yaml
requires: [apiCall]
after:
  - set('body', http.responseBody('apiCall'))
  - set('content_type', http.responseHeader('apiCall', 'Content-Type'))
  - set('rate_left', http.responseHeader('apiCall', 'X-RateLimit-Remaining'))
  - set('ok', get('apiCall').statusCode >= 200)
```

- `http.responseBody('id')` — raw response body as string
- `http.responseHeader('id', 'Name')` — value of a specific response header

## Python Resource

The `python:` resource runs an inline Python script and stores its stdout (parsed as JSON) as output.

### Basic Usage

```yaml
# resources/process.yaml
actionId: process
python:
  script: |
    import json
    data = &#123;&#123; get('inputData') &#125;&#125;
    result = {"count": len(data), "items": [x.upper() for x in data]}
    print(json.dumps(result))
```

The script must print exactly one JSON value to stdout. That value becomes the resource's output. Anything printed to stderr is captured in logs but does not affect output.

### Full Configuration Reference

```yaml
python:
  script: |                       # inline script — must print JSON to stdout
    import json
    print(json.dumps({"result": 42}))

  scriptFile: "./scripts/process.py"  # alternative: path to a .py file

  args:                           # command-line arguments passed to the script
    - "--mode"
    - "analyze"

  venvName: "my-env"              # isolated virtualenv; resources sharing the same
                                  # name share packages; different names stay isolated

  timeout: 60s
```

`script` and `scriptFile` are mutually exclusive. Use `scriptFile` for scripts too large for inline YAML.

### Python Package Management

Declare Python dependencies in `workflow.yaml`. kdeps uses [uv](https://github.com/astral-sh/uv) for fast package installation (significantly faster than pip):

```yaml
# workflow.yaml
settings:
  agentSettings:
    pythonVersion: "3.12"         # optional; defaults to system Python

    # Option 1: explicit package list (most common)
    pythonPackages:
      - pandas>=2.0
      - requests
      - beautifulsoup4

    # Option 2: requirements.txt file
    requirementsFile: "requirements.txt"

    # Option 3: pyproject.toml + lockfile (for uv projects)
    pyprojectFile: "pyproject.toml"
    lockFile: "uv.lock"
```

Packages are installed before the first Python resource runs and shared across all `python:` resources in the workflow.

### Virtual Environment Isolation

Use `venvName:` to isolate incompatible package sets across resources:

```yaml
# resources/data-science.yaml
actionId: analyze
python:
  venvName: "datascience-env"    # has pandas, numpy, scikit-learn
  script: |
    import pandas as pd
    # ...

# resources/web-scraper.yaml
actionId: scrape
python:
  venvName: "scraper-env"        # has requests, beautifulsoup4
  script: |
    from bs4 import BeautifulSoup
    # ...
```

Resources with the same `venvName` share the same virtualenv. Resources with different names (or no `venvName`) use the default shared environment.

### Practical Example: Data Transformation

```yaml
# resources/transform.yaml
actionId: transform
requires: [fetchData]
python:
  script: |
    import json
    import statistics
    
    rows = &#123;&#123; get('fetchData') &#125;&#125;
    values = [float(r['value']) for r in rows if r.get('value') is not None]
    
    result = {
        "count": len(values),
        "mean": statistics.mean(values) if values else None,
        "median": statistics.median(values) if values else None,
        "stdev": statistics.stdev(values) if len(values) > 1 else None,
        "min": min(values) if values else None,
        "max": max(values) if values else None,
    }
    print(json.dumps(result))
```

### Passing Data to the Script

The `&#123;&#123; get('key') &#125;&#125;` interpolation works inside Python scripts. The expression is evaluated and its JSON representation is substituted into the script source before execution:

```yaml
python:
  script: |
    import json
    
    # get('records') is interpolated as a Python literal (JSON)
    records = &#123;&#123; get('records') &#125;&#125;
    user_id = "&#123;&#123; get('userId') &#125;&#125;"
    
    filtered = [r for r in records if r['user_id'] == user_id]
    print(json.dumps(filtered))
```

Note: string values from `&#123;&#123; get('field') &#125;&#125;` are quoted. Object/array values from `&#123;&#123; get('records') &#125;&#125;` are inlined as Python-compatible JSON literals (since Python's `True`/`False`/`None` differ from JSON's `true`/`false`/`null`, be careful with boolean fields — use `json.loads` if needed).

### External Script Files

For scripts too long for inline YAML, reference a file with `scriptFile:`:

```yaml
python:
  scriptFile: "./scripts/analyze.py"
  args:
    - "--mode"
    - "analyze"
    - "--input"
    - "&#123;&#123; get('rawData') &#125;&#125;"
  timeout: 60s
```

The script receives arguments via `sys.argv` and must print JSON to stdout:

```python
# scripts/analyze.py
import sys, json, argparse

parser = argparse.ArgumentParser()
parser.add_argument("--mode", required=True)
parser.add_argument("--input", required=True)
args = parser.parse_args()

data = json.loads(args.input)
result = {"mode": args.mode, "count": len(data)}
print(json.dumps(result))
```

### Accessing Output Details

Access exit code and stderr from downstream resources:

```yaml
requires: [transform]
after:
  - set('ok', python.exitCode('transform') == 0)
  - set('errors', python.stderr('transform'))
```

- `python.exitCode('resourceId')` — integer exit code (0 = success)
- `python.stderr('resourceId')` — stderr output as string (useful for debugging)

## Exec Resource

The `exec:` resource runs a shell command and stores its stdout as output. Use it for system operations, file processing, or wrapping CLI tools that do not have a native resource type.

### Configuration Reference

```yaml
exec:
  command: "your-command"    # shell command; supports multiline
  args:                      # optional: command-line arguments (appended after command)
    - "--flag"
    - "value"
  workingDir: "/tmp"         # optional: set working directory before execution
  env:                       # optional: per-resource environment variables
    KEY: "value"
  timeout: 30s               # max execution time; default 30s
```

### Basic Usage

```yaml
# resources/run-script.yaml
actionId: runScript
exec:
  command: "echo 'Hello, World!'"
  timeout: 30s
```

### Processing Files

```yaml
# resources/count-lines.yaml
actionId: lineCount
exec:
  command: "wc -l /data/input.txt | awk '{print $1}'"

# resources/generate-report.yaml
actionId: generateReport
requires: [processData]
exec:
  command: "python3 /scripts/generate_pdf.py"
  args:
    - "--data"
    - "&#123;&#123; get('processData') &#125;&#125;"
    - "--output"
    - "/output/report.pdf"
  workingDir: "/scripts"
  timeout: 120s
```

### Multi-Line Scripts

```yaml
exec:
  command: |
    set -e
    mkdir -p /output
    cp /data/input.csv /output/
    csvtool col 1,3,5 /output/input.csv > /output/filtered.csv
    wc -l /output/filtered.csv
```

### Environment Variables

```yaml
exec:
  command: "aws s3 cp s3://&#123;&#123; get('bucket') &#125;&#125;/&#123;&#123; get('key') &#125;&#125; /tmp/download"
  env:
    AWS_ACCESS_KEY_ID: "&#123;&#123; env('AWS_ACCESS_KEY_ID') &#125;&#125;"
    AWS_SECRET_ACCESS_KEY: "&#123;&#123; env('AWS_SECRET_ACCESS_KEY') &#125;&#125;"
    AWS_DEFAULT_REGION: us-east-1
```

### Accessing Output Details

By default, `get('resourceId')` returns the exec resource's stdout. To access stderr or exit code in downstream resources:

```yaml
# resources/process.yaml
actionId: process
exec:
  command: "some-tool"

# resources/check.yaml
requires: [process]
after:
  - set('ok', exec.exitCode('process') == 0)
  - set('errors', exec.stderr('process'))
```

- `exec.exitCode('resourceId')` — integer exit code (0 = success)
- `exec.stderr('resourceId')` — stderr output as string

This lets you branch on command success/failure without needing wrapper scripts that capture exit codes themselves.

### Security Considerations

The `exec:` resource runs in the workflow's execution environment with the permissions of the process running kdeps. Follow these practices:

- Validate inputs with `validations:` before they reach exec commands
- Never interpolate user input directly into shell commands without sanitization
- Use `exec:` for trusted operations, not for executing user-supplied commands

```yaml
# Safer: command is fixed, only the argument varies and is validated upstream
exec:
  command: "process-document --id &#123;&#123; get('documentId') &#125;&#125;"

# Risky: user input could inject shell metacharacters
exec:
  command: "&#123;&#123; get('userSuppliedCommand') &#125;&#125;"  # Never do this
```

## Combining Data Resources

The real power of data resources comes from chaining them in a DAG. A typical pattern:

```yaml
# Fetch from external API
fetchData → httpClient

# Store in database
storeResult → sql, requires: [fetchData]

# Enrich with LLM
enrich → chat, requires: [fetchData]

# Post to webhook
notify → httpClient, requires: [enrich, storeResult]

# Respond
respond → apiResponse, requires: [notify]
```

Each resource does one thing. The DAG makes the dependencies explicit. The result is a workflow that is easy to understand, test independently, and modify without breaking unrelated steps.

The next chapter covers knowledge resources — scraping, search, and embeddings — which connect your workflows to unstructured information.

## Email Resource

The `email:` resource sends outbound email via SMTP and reads or searches inbound messages via IMAP. It is the standard way to deliver notifications, reports, and alerts from a kdeps workflow.

Four actions are available via `action:`:

| Action | What it does |
|---|---|
| `send` (default) | Sends an email via SMTP |
| `read` | Retrieves recent messages from an IMAP mailbox |
| `search` | Searches messages in an IMAP mailbox by criteria |
| `modify` | Changes flags or moves/deletes messages via IMAP |

### Global Named Connections

SMTP and IMAP credentials belong in `~/.kdeps/config.yaml` -- not in `workflow.yaml`. Resources reference connections by name. This keeps all secrets in one machine-local file and out of version-controlled workflow files.

```yaml
# ~/.kdeps/config.yaml
smtp_connections:
  default:
    host: "${SMTP_HOST}"          # e.g. smtp.gmail.com
    port: 587
    username: "${SMTP_USER}"
    password: "${SMTP_PASS}"
    tls: false                    # false = STARTTLS on 587, true = implicit TLS on 465
imap_connections:
  inbox:
    host: "${IMAP_HOST}"          # e.g. imap.gmail.com
    port: 993
    username: "${IMAP_USER}"
    password: "${IMAP_PASS}"
    tls: true
```

### Sending Email

```yaml
# resources/notify.yaml
actionId: notify
requires: [llm]
email:
  action: send
  smtpConnection: default   # references smtp_connections.default in ~/.kdeps/config.yaml
  from: "reports@example.com"
  to:
    - "alice@example.com"
    - "bob@example.com"
  subject: "Daily Report — &#123;&#123; get('date') &#125;&#125;"
  body: "&#123;&#123; get('llm') &#125;&#125;"
```

For HTML email, set `html: true` and put your HTML in `body:`:

```yaml
email:
  action: send
  smtpConnection: default
  from: "noreply@example.com"
  to: ["&#123;&#123; get('recipient') &#125;&#125;"]
  subject: "Your Report"
  body: "<h1>Summary</h1><p>&#123;&#123; get('llm') &#125;&#125;</p>"
  html: true
```

To send attachments, list local file paths in `attachments:`:

```yaml
email:
  action: send
  smtpConnection: default
  from: "reports@example.com"
  to: ["cfo@example.com"]
  subject: "Q3 Report"
  body: "See attached."
  attachments:
    - "/data/reports/q3.pdf"
    - "/data/reports/q3-summary.csv"
```

### Reading Email

```yaml
# resources/check-inbox.yaml
actionId: checkInbox
email:
  action: read
  imapConnection: inbox   # references imap_connections.inbox in ~/.kdeps/config.yaml
  mailbox: "INBOX"
  limit: 10               # retrieve at most 10 messages
  markRead: true          # mark retrieved messages as read
```

The output is an array of message objects. Access it with `get('checkInbox')`:

```yaml
before:
  - set('first_subject', get('checkInbox')[0].subject)
  - set('first_body', get('checkInbox')[0].body)
  - set('message_count', len(get('checkInbox')))
```

Each message object has: `uid`, `subject`, `from`, `to`, `date`, `body`, `html`.

### Searching Email

```yaml
# resources/find-orders.yaml
actionId: findOrders
email:
  action: search
  imapConnection: inbox   # named connection from imap_connections in ~/.kdeps/config.yaml
  mailbox: "INBOX"
  limit: 50
  search:
    from: "orders@shopify.com"
    subject: "New order"
    unseen: true
    since: "2024-01-01"   # ISO date string
```

Available search fields: `from`, `to`, `subject`, `body`, `since`, `before`, `unseen`, `flagged`.

### Modifying Messages

```yaml
# resources/archive-processed.yaml
actionId: archiveProcessed
requires: [processOrder]
email:
  action: modify
  imapConnection: inbox   # named connection from imap_connections in ~/.kdeps/config.yaml
  mailbox: "INBOX"
  uids:
    - "&#123;&#123; get('findOrders')[0].uid &#125;&#125;"
  modify:
    markSeen: true
    moveTo: "Processed"
```

`modify:` fields: `markSeen` (*bool), `markFlagged` (*bool), `markDeleted` (*bool), `moveTo` (mailbox name), `expunge` (bool - permanently deletes messages flagged for deletion).

### Output Shape

The `send` action returns:

```json
{"success": true, "recipients": 2}
```

The `read` and `search` actions return an array:

```json
[
  {
    "uid": "42",
    "subject": "New order #1234",
    "from": "orders@shopify.com",
    "to": ["ops@example.com"],
    "date": "2024-03-15T09:00:00Z",
    "body": "Order details...",
    "html": ""
  }
]
```

The `modify` action returns:

```json
{"success": true, "modified": 1}
```

### Configuration Reference

**`smtp_connections` fields (in `~/.kdeps/config.yaml`):**

| Field | Type | Description |
|---|---|---|
| `host` | string | SMTP server hostname |
| `port` | int | Port (default: 465 for TLS, 587 for STARTTLS) |
| `username` | string | Auth username |
| `password` | string | Auth password |
| `tls` | bool | `true` = implicit TLS on 465, `false` = STARTTLS on 587 |
| `insecureSkipVerify` | bool | Skip TLS certificate verification (dev only) |

**`imap_connections` fields (in `~/.kdeps/config.yaml`):**

| Field | Type | Description |
|---|---|---|
| `host` | string | IMAP server hostname |
| `port` | int | Port (default: 993 for TLS, 143 for plain) |
| `username` | string | Auth username |
| `password` | string | Auth password |
| `tls` | bool | Enable TLS |
| `insecureSkipVerify` | bool | Skip TLS certificate verification (dev only) |

**Top-level `email:` fields:**

| Field | Type | Default | Description |
|---|---|---|---|
| `action` | string | `send` | `send`, `read`, `search`, or `modify` |
| `smtpConnection` | string | | Named SMTP connection (required for send) |
| `imapConnection` | string | | Named IMAP connection (required for read/search/modify) |
| `from` | string | | Sender address (send only) |
| `to` | []string | | Recipients (send only) |
| `cc` | []string | | CC recipients (send only) |
| `bcc` | []string | | BCC recipients (send only) |
| `subject` | string | | Subject line (send only) |
| `body` | string | | Plain-text or HTML body (send only) |
| `html` | bool | false | Treat `body` as HTML (send only) |
| `attachments` | []string | | File paths to attach (send only) |
| `mailbox` | string | `INBOX` | Mailbox to read/search/modify |
| `limit` | int | 10 | Max messages to return (read/search) |
| `markRead` | bool | false | Mark retrieved messages as read |
| `uids` | []string | | Message UIDs to modify (modify only) |
| `timeout` | string | `30s` | Operation timeout |

### Secrets

Never hardcode credentials. All SMTP and IMAP passwords live in `~/.kdeps/config.yaml`, always referencing environment variables:

```yaml
# ~/.kdeps/config.yaml
smtp_connections:
  default:
    host: "${SMTP_HOST}"
    username: "${SMTP_USER}"
    password: "${SMTP_PASS}"
imap_connections:
  inbox:
    host: "${IMAP_HOST}"
    username: "${IMAP_USER}"
    password: "${IMAP_PASS}"
```

For Gmail: use an App Password (not your account password). SMTP: `smtp.gmail.com:587` with `tls: false` (STARTTLS). IMAP: `imap.gmail.com:993` with `tls: true`.

X> ## Exercise
X>
X> Build a workflow that answers the question: "What are the top 5 most expensive products in our catalog?" by combining a SQL query with an LLM summary.
X>
X> 1. Configure a SQLite connection named `catalog` in `~/.kdeps/config.yaml` pointing to a local SQLite file (`sqlite:///data/catalog.db`). Seed it with a table `products(id, name, price)` containing at least 10 rows. Add a matching `sqlConnections.catalog:` pool block in `workflow.yaml`.
X> 2. Write a `sql:` resource that queries the top 5 products by price.
X> 3. Write a `python:` resource that transforms the result set into a formatted string (name + price per line).
X> 4. Write a `chat:` resource that uses the formatted string in its prompt to generate a natural-language summary.
X> 5. Return the summary and the raw top-5 list in the `apiResponse:`.
X>
X> Verify that changing the data in the SQLite file changes the LLM's answer without modifying any YAML.
X>
X> **Stretch goal:** Add an `email:` resource that emails the LLM summary to a configured recipient using environment variable credentials. Verify the email arrives with the correct subject and body.
