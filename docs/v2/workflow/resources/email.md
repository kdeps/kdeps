# Email component

The `email` component sends email over SMTP and reads, searches, modifies, lists or deletes IMAP mailboxes. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool.

## Requirements

None beyond kdeps itself. The SMTP and IMAP clients are compiled in, and the tool reads the named connection from `~/.kdeps/config.yaml` in-process - passwords never cross the request.

## Actions

Set the `action` input:

| Action | What it does |
|---|---|
| `send` (default) | Send an email over SMTP |
| `read` | Return the most recent messages of a mailbox |
| `search` | Return messages matching `search:` criteria |
| `modify` | Set or clear flags, or move messages |
| `list` | List the selectable mailboxes |
| `delete` | Delete a mailbox (never `INBOX` or `[Gmail]/...`) |

## Global named connections

SMTP and IMAP credentials belong in `~/.kdeps/config.yaml`, not in `workflow.yaml`. The component's `smtpConnection` / `imapConnection` inputs name them. This keeps all secrets in one machine-local file and out of version-controlled workflow files.

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

## Interactive setup on first run

If a workflow names a connection that is missing from `~/.kdeps/config.yaml`, `kdeps run` prompts for its fields at startup (before the server starts) and saves them back to `config.yaml` - so you never have to hand-edit the file to get going:

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

## Set connections via environment variables

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
component:
  name: email                    # from the components asset set; a components/email/ of your own wins
  with:
    smtpConnection: default      # smtp_connections.default in ~/.kdeps/config.yaml
    from: reports@example.com    # optional: defaults to identity.email
    to: [alice@example.com]      # a list, or one comma-separated string
    bcc: [audit@example.com]     # delivered, never written to the headers
    subject: Daily Report
    body: "{{ get('llm').message.content }}"
    html: false                  # true: body is text/html
    attachments: [reports/q3.pdf]  # relative paths resolve against the workflow directory
```

</div>

`from` defaults to the agent's [identity](/reference/advanced-config#agent-identity) `identity.email`. A `from`, `subject` or recipient containing a line break fails the call (header injection). With `tls: true` the connection uses implicit TLS (port 465); otherwise it upgrades with STARTTLS when the server offers it (port 587).

```json
{"success": true, "action": "send", "from": "reports@example.com", "to": ["alice@example.com"], "cc": [], "subject": "Daily Report", "attachments": 1}
```

## Reading and searching

<div v-pre>

```yaml
# resources/find-orders.yaml
actionId: findOrders
component:
  name: email
  with:
    action: search                # read: the newest messages, no criteria
    imapConnection: inbox         # imap_connections.inbox in ~/.kdeps/config.yaml
    mailbox: INBOX                # default INBOX
    limit: 20                     # newest 20 matches (default 10)
    markRead: false               # true: mark returned messages \Seen
    search:
      from: orders@shop.example
      subject: New order
      unseen: true
      since: "2024-01-01"         # YYYY-MM-DD; also before:
```

</div>

Search fields: `from`, `subject`, `body`, `since`, `before`, `unseen`, `flagged`.

```json
{
  "success": true,
  "action": "search",
  "mailbox": "INBOX",
  "count": 1,
  "messages": [
    {
      "uid": 42,
      "messageId": "<abc@shop.example>",
      "from": "Shop <orders@shop.example>",
      "to": "ops@example.com",
      "subject": "New order #1234",
      "date": "2024-03-15T09:00:00Z",
      "body": "Order details...",
      "seen": false,
      "listUnsubscribe": "<https://shop.example/u>, <mailto:u@shop.example>",
      "unsubscribeUrl": "https://shop.example/u",
      "unsubscribeMailto": "mailto:u@shop.example",
      "unsubscribeOneClick": true
    }
  ]
}
```

Messages are oldest first. `body` is the message's text part (`text/plain` when present, otherwise `text/html`), trimmed - not the raw MIME source. The unsubscribe fields are present only when the message has a `List-Unsubscribe` header. Read them with `output('findOrders').messages[0].subject`.

## Modifying messages

<div v-pre>

```yaml
# resources/archive.yaml
actionId: archive
requires: [findOrders]
component:
  name: email
  with:
    action: modify
    imapConnection: inbox
    uids: ["{{ output('findOrders').messages[0].uid }}"]   # empty: every message search: matches
    modify:
      markSeen: true              # true sets, false clears; also markFlagged, markDeleted
      moveTo: Processed           # matched case-insensitively to an existing mailbox, else created
      expunge: false              # true: remove \Deleted messages (ignored with moveTo)
```

</div>

```json
{"success": true, "action": "modify", "mailbox": "INBOX", "count": 1, "uids": [42]}
```

A missing source mailbox or no matching messages returns `count: 0`. On servers without `MOVE`, `moveTo` copies, marks the originals deleted and expunges.

## Listing and deleting mailboxes

```yaml
# resources/folders.yaml
actionId: folders
component:
  name: email
  with:
    action: list                  # {success, action, count, folders: ["INBOX", "Archive", ...]}
    imapConnection: inbox
```

`action: delete` with `mailbox: Old` deletes that mailbox; a mailbox that is already gone counts as deleted.

## Inputs

| Input | Default | Description |
|---|---|---|
| `action` | `send` | `send`, `read`, `search`, `modify`, `list`, `delete` |
| `smtpConnection` | | Name under `smtp_connections` (send) |
| `imapConnection` | | Name under `imap_connections` (the IMAP actions) |
| `from` | `identity.email` | Sender address |
| `to`, `cc`, `bcc` | `[]` | Recipients: a list or a comma-separated string |
| `subject` | | Subject (required for send) |
| `body` | | Message body |
| `html` | `false` | Send `body` as `text/html` |
| `attachments` | `[]` | File paths to attach |
| `mailbox` | `INBOX` | Mailbox for read, search, modify; required for delete |
| `limit` | `10` | Messages to return at most |
| `markRead` | `false` | Mark returned messages as seen |
| `uids` | `[]` | Message UIDs for modify |
| `search` | `{}` | `from`, `subject`, `body`, `since`, `before`, `unseen`, `flagged` |
| `modify` | `{}` | `markSeen`, `markFlagged`, `markDeleted`, `moveTo`, `expunge` |
| `timeout` | `30s` | Connection timeout |

### Connection fields

| Field | Description |
|---|---|
| `host` | Server hostname |
| `port` | Default 465 (SMTP, `tls: true`), 587 (SMTP), 993 (IMAP, `tls: true`), 143 (IMAP) |
| `username`, `password` | Login; omitted means no login |
| `tls` | `true`: implicit TLS. SMTP without it uses STARTTLS when offered |
| `insecureSkipVerify` | Skip certificate verification (development only) |

**Gmail:** use an [App Password](https://support.google.com/accounts/answer/185833). SMTP `smtp.gmail.com:587` with `tls: false` (STARTTLS); IMAP `imap.gmail.com:993` with `tls: true`.

## Errors

| Error | Cause |
|---|---|
| `email: smtpConnection is required for send ...` | No `smtpConnection` (likewise `imapConnection` for the IMAP actions) |
| `email: smtpConnection 'x' not found in ~/.kdeps/config.yaml smtp_connections` | The name is not configured |
| `email: header Subject contains CR or LF (header injection)` | A line break in a header value |
| `email: send: ...` / `email: imap connect ...` | The server refused or could not be reached |
| `email: delete: refusing to delete protected mailbox 'INBOX'` | `delete` on `INBOX` or `[Gmail]/...` |

Add `onError:` to the calling resource to continue with a fallback instead of failing the workflow.

## How it is built

```text
component: email -> POST /apis/tools.kdeps.io/v1/tools/email_ops/call -> send summary or mailbox result
```

One `httpClient` POST carrying the eighteen inputs to the `email_ops` tool, then an `interface.output` that decodes the tool's JSON body. The tool reads the named SMTP/IMAP connection from `config.yaml` itself and runs the action with the Go standard library (`net/smtp`, `mime`) and `go-imap/v2` - no python, no uv. Message building, IMAP command sequencing, result shapes and the error texts live in the tool. Copy `components/email/` into your project to change an input default; the mail operations themselves live in the kdeps API.

## See also

- [Global config](/reference/advanced-config) - where SMTP and IMAP credentials live
- [Error handling (onError)](/workflow/error-handling) - retry and fallback behavior
- [Components](/agencies/components) - how shipped components load
