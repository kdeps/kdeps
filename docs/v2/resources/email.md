# Email Resource

The `email:` resource sends outbound email via SMTP and reads or searches inbound messages via IMAP. Use it to deliver notifications, reports, and alerts from any workflow step.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode).

## Actions

Set `action:` to one of four values:

| Action | What it does |
|---|---|
| `send` (default) | Send an email via SMTP |
| `read` | Retrieve recent messages from an IMAP mailbox |
| `search` | Search messages in an IMAP mailbox by criteria |
| `modify` | Change flags or move/delete messages via IMAP |

## Global Named Connections

Both SMTP and IMAP credentials are defined once in `workflow.yaml` under `settings` and referenced by name in each resource. This keeps all secrets in one place and out of resource files.

```yaml
# workflow.yaml
settings:
  smtpConnections:
    default:
      host: "${SMTP_HOST}"      # e.g. smtp.gmail.com
      port: 587
      username: "${SMTP_USER}"
      password: "${SMTP_PASS}"
      tls: true

  imapConnections:
    inbox:
      host: "${IMAP_HOST}"      # e.g. imap.gmail.com
      port: 993
      username: "${IMAP_USER}"
      password: "${IMAP_PASS}"
      tls: true
```

## Sending Email

<div v-pre>

```yaml
# resources/notify.yaml
actionId: notify
requires: [llm]
email:
  action: send
  smtpConnection: default   # references settings.smtpConnections.default
  from: "reports@example.com"
  to:
    - "alice@example.com"
  subject: "Daily Report"
  body: "{{ get('llm') }}"
```

</div>

HTML email — set `html: true` and put HTML in `body:`:

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

## Reading Email

```yaml
# resources/check-inbox.yaml
actionId: checkInbox
email:
  action: read
  imapConnection: inbox   # references settings.imapConnections.inbox
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

## Searching Email

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

## Modifying Messages

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

## Configuration Reference

### `smtpConnections` fields (in `workflow.yaml` settings)

| Field | Type | Description |
|---|---|---|
| `host` | string | SMTP server hostname |
| `port` | int | Port (default: 465 for TLS, 587 for STARTTLS) |
| `username` | string | Auth username |
| `password` | string | Auth password |
| `tls` | bool | `true` = implicit TLS (port 465), `false` = STARTTLS (port 587) |
| `insecureSkipVerify` | bool | Skip TLS certificate verification (dev only) |

### `imapConnections` fields (in `workflow.yaml` settings)

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

Always use environment variables — never hardcode credentials in workflow files:

```yaml
# workflow.yaml
settings:
  smtpConnections:
    default:
      host: "${SMTP_HOST}"
      username: "${SMTP_USER}"
      password: "${SMTP_PASS}"
  imapConnections:
    inbox:
      host: "${IMAP_HOST}"
      username: "${IMAP_USER}"
      password: "${IMAP_PASS}"
```

**Gmail:** Use an [App Password](https://support.google.com/accounts/answer/185833), not your account password. SMTP: `smtp.gmail.com:587` with `tls: false` (STARTTLS). IMAP: `imap.gmail.com:993` with `tls: true`.

## Common Patterns

### Send a report after LLM generation

<div v-pre>

```yaml
# workflow.yaml
settings:
  smtpConnections:
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
  subject: "Weekly Summary — {{ get('week') }}"
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

### onError fallback for SMTP failures

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
