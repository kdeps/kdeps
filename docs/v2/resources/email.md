# Email resource

The `email:` resource sends outbound email via SMTP and reads or searches inbound messages via IMAP. Use it to deliver notifications, reports, and alerts from any workflow step.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode).

## Actions

Set `action:` to one of six values:

| Action | What it does |
|---|---|
| `send` (default) | Send an email via SMTP |
| `read` | Retrieve recent messages from an IMAP mailbox |
| `search` | Search messages in an IMAP mailbox by criteria |
| `modify` | Change flags or move messages via IMAP |
| `list` | List IMAP mailboxes/folders |
| `delete` | Delete messages from an IMAP mailbox |

## Global named connections

SMTP and IMAP credentials belong in `~/.kdeps/config.yaml`, not in `workflow.yaml`. Resources reference connections by name. This keeps all secrets in one machine-local file and out of version-controlled workflow files.

```yaml
# ~/.kdeps/config.yaml
smtp_connections:
  default:
    host: "${SMTP_HOST}"      # e.g. smtp.gmail.com
    port: 587
    username: "${SMTP_USER}"
    password: "${SMTP_PASS}"
    tls: false                # false = STARTTLS on 587, true = implicit TLS on 465

imap_connections:
  inbox:
    host: "${IMAP_HOST}"      # e.g. imap.gmail.com
    port: 993
    username: "${IMAP_USER}"
    password: "${IMAP_PASS}"
    tls: true
```

### Interactive setup on first run

If a resource references a connection that is missing from `~/.kdeps/config.yaml`, `kdeps run` prompts for its fields at startup (before the server starts) and saves them back to `config.yaml` - so you never have to hand-edit the file to get going:

```text
Connection "inbox" (imap) is referenced but not configured.
Enter its details to save them to ~/.kdeps/config.yaml.
  Host: imap.gmail.com
  Port [993]:
  Username: me@gmail.com
  Password (input hidden): ****
  Use TLS? [Y/n]: y
  ✓ Saved imap connection "inbox" to ~/.kdeps/config.yaml
```

This is interactive-only: when stdin is not a terminal (CI, pipes), kdeps skips the prompt and the usual "connection not found" error surfaces at execution time. The same applies to `sql_connections`, `http_connections`, and `search_connections`, plus cloud LLM API keys (when a `chat` resource uses a cloud model whose key is missing) and the `api_auth_token` (when `apiServer` is configured). Existing content and comments in `config.yaml` are preserved.

Values already provided by an environment variable (e.g. `DEEPSEEK_API_KEY`, `KDEPS_API_AUTH_TOKEN`) are never prompted for and never written to `config.yaml`; kdeps prints a notice that it is using the value from the environment.

### Set connections via environment variables

Every named connection can be supplied entirely from the environment - no `config.yaml` entry needed - using the convention `KDEPS_<KIND>_CONNECTIONS_<NAME>_<FIELD>`. The name is matched case-insensitively and used lowercased, so reference it in lowercase in resources (`connectionName: default`, `smtpConnection: alerts`). Env values win over `config.yaml`, per field:

```bash
# smtp_connections.alerts
export KDEPS_SMTP_CONNECTIONS_ALERTS_HOST=smtp.gmail.com
export KDEPS_SMTP_CONNECTIONS_ALERTS_PORT=587
export KDEPS_SMTP_CONNECTIONS_ALERTS_USERNAME=bot@example.com
export KDEPS_SMTP_CONNECTIONS_ALERTS_PASSWORD=app-password
export KDEPS_SMTP_CONNECTIONS_ALERTS_TLS=true

# imap_connections.inbox  (same fields; plus INSECURE_SKIP_VERIFY)
export KDEPS_IMAP_CONNECTIONS_INBOX_HOST=imap.gmail.com

# sql_connections.main
export KDEPS_SQL_CONNECTIONS_MAIN_CONNECTION="postgres://user:pass@host/db"

# search_connections.web
export KDEPS_SEARCH_CONNECTIONS_WEB_API_KEY=sk-...

# http_connections.api  (PROXY, or AUTH_TYPE / AUTH_USERNAME / AUTH_PASSWORD /
#                         AUTH_TOKEN / AUTH_KEY / AUTH_VALUE)
export KDEPS_HTTP_CONNECTIONS_API_AUTH_TYPE=bearer
export KDEPS_HTTP_CONNECTIONS_API_AUTH_TOKEN=tok-123

# bot_connections (fixed platforms)
export KDEPS_BOT_CONNECTIONS_SLACK_BOT_TOKEN=xoxb-...
export KDEPS_BOT_CONNECTIONS_DISCORD_BOT_TOKEN=...
```

Env-supplied connections are picked up automatically at run time (not prompted, not written to `config.yaml`); kdeps prints a notice that it is using the connection from the environment. This is the CI-friendly way to inject connection secrets.

## Sending email

<div v-pre>

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
  subject: "Daily Report"
  body: "{{ get('llm') }}"
```

</div>

`from` is optional when the agent has a configured [identity](/configuration/advanced#agent-identity) - it defaults to `identity.email`, so a per-agent identity means you don't have to repeat the sender address on every `email:` resource.

HTML email - set `html: true` and put HTML in `body:`:

<div v-pre>

```yaml
email:
  action: send
  smtpConnection: default
  from: "noreply@example.com"
  to: ["{{ get('recipient') }}"]
  subject: "Your Report"
  body: "<h1>Summary</h1><p>{{ get('llm') }}</p>"
  html: true
```

</div>

With attachments:

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
```

### Output (send)

```json
{"success": true, "action": "send", "from": "...", "to": [...], "subject": "..."}
```

## Reading email

```yaml
# resources/check-inbox.yaml
actionId: checkInbox
email:
  action: read
  imapConnection: inbox   # references imap_connections.inbox in ~/.kdeps/config.yaml
  mailbox: "INBOX"
  limit: 10
  markRead: true
```

### Output (read)

An array of message objects:

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

Access fields with `get('checkInbox')[0].subject`, `get('checkInbox')[0].body`, etc.

## Searching email

<div v-pre>

```yaml
# resources/find-orders.yaml
actionId: findOrders
email:
  action: search
  imapConnection: inbox
  mailbox: "INBOX"
  limit: 50
  search:
    from: "orders@shopify.com"
    subject: "New order"
    unseen: true
    since: "2024-01-01"
```

</div>

Search fields: `from`, `to`, `subject`, `body`, `since` (ISO date), `before` (ISO date), `unseen` (bool), `flagged` (bool).

## Modifying messages

<div v-pre>

```yaml
# resources/archive.yaml
actionId: archive
email:
  action: modify
  imapConnection: inbox
  mailbox: "INBOX"
  uids:
    - "{{ get('findOrders')[0].uid }}"
  modify:
    markSeen: true
    moveTo: "Processed"
```

</div>

### Output (modify)

```json
{"success": true, "modified": 1}
```

## Configuration reference

### `smtp_connections` fields (in `~/.kdeps/config.yaml`)

| Field | Type | Description |
|---|---|---|
| `host` | string | SMTP server hostname |
| `port` | int | Port (default: 465 for TLS, 587 for STARTTLS) |
| `username` | string | Auth username |
| `password` | string | Auth password |
| `tls` | bool | `true` = implicit TLS (port 465), `false` = STARTTLS (port 587) |
| `insecureSkipVerify` | bool | Skip TLS certificate verification (dev only) |

### `imap_connections` fields (in `~/.kdeps/config.yaml`)

| Field | Type | Description |
|---|---|---|
| `host` | string | IMAP server hostname |
| `port` | int | Port (default: 993 for TLS, 143 for plain) |
| `username` | string | Auth username |
| `password` | string | Auth password |
| `tls` | bool | Enable TLS |
| `insecureSkipVerify` | bool | Skip TLS certificate verification (dev only) |

### Top-level `email:` fields

| Field | Type | Default | Description |
|---|---|---|---|
| `action` | string | `send` | `send`, `read`, `search`, `modify`, `list`, or `delete` |
| `smtpConnection` | string | | Named SMTP connection (required for send) |
| `imapConnection` | string | | Named IMAP connection (required for read/search/modify/list/delete) |
| `from` | string | | Sender address (send only) |
| `to` | []string | | Recipients (send only) |
| `cc` | []string | | CC recipients (send only) |
| `bcc` | []string | | BCC recipients (send only) |
| `subject` | string | | Subject line (send only) |
| `body` | string | | Plain-text or HTML body (send only) |
| `html` | bool | false | Treat `body` as HTML (send only) |
| `attachments` | []string | | Local file paths to attach (send only) |
| `mailbox` | string | `INBOX` | Mailbox for read/search/modify |
| `limit` | int | 10 | Max messages to return (read/search) |
| `markRead` | bool | false | Mark retrieved messages as read |
| `uids` | []string | | Message UIDs to target (modify) |
| `search` | object | | Search criteria (search action) |
| `modify` | object | | Modification flags (modify action) |
| `timeout` | string | `30s` | Operation timeout |

### `modify:` fields

| Field | Type | Description |
|---|---|---|
| `markSeen` | *bool | Set or clear \\Seen flag |
| `markFlagged` | *bool | Set or clear \\Flagged flag |
| `markDeleted` | *bool | Set or clear \\Deleted flag |
| `moveTo` | string | Move messages to this mailbox |
| `expunge` | bool | Permanently delete messages marked for deletion |

## Secrets

Always use environment variables - never hardcode credentials:

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

**Gmail:** Use an [App Password](https://support.google.com/accounts/answer/185833), not your account password. SMTP: `smtp.gmail.com:587` with `tls: false` (STARTTLS). IMAP: `imap.gmail.com:993` with `tls: true`.

## Common patterns

### Send a report after LLM generation

<div v-pre>

```yaml
# ~/.kdeps/config.yaml
smtp_connections:
  reports:
    host: "${SMTP_HOST}"
    port: 587
    username: "${SMTP_USER}"
    password: "${SMTP_PASS}"
    tls: false

# resources/send-report.yaml
actionId: sendReport
requires: [generateReport]
email:
  action: send
  smtpConnection: reports
  from: "${REPORT_FROM}"
  to: ["${REPORT_TO}"]
  subject: "Weekly Summary - {{ get('week') }}"
  body: "{{ get('generateReport') }}"
```

</div>

### Poll inbox and process new messages

```yaml
# resources/poll.yaml
actionId: poll
email:
  action: search
  imapConnection: inbox
  search:
    unseen: true
  limit: 20
```

### `onError` fallback for SMTP failures

```yaml
email:
  action: send
  smtpConnection: default
  from: "alerts@example.com"
  to: ["ops@example.com"]
  subject: "Alert"
  body: "Something happened."
onError:
  action: continue
  fallback: {"success": false, "message": "email delivery failed"}
```

## See also

- [Global config](/configuration/advanced) - where SMTP and IMAP credentials live
- [Error handling (onError)](/concepts/error-handling) - retry and fallback behavior
- [Expressions](/concepts/expressions) - templating the subject and body
- [Resources overview](/resources/overview) - resource structure and dependencies
