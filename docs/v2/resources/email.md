# Email Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install email`
>
> Usage: `run: { component: { name: email, with: { to: "...", subject: "...", body: "...", smtpHost: "...", smtpUser: "...", smtpPass: "..." } } }`

The Email component supports sending email via SMTP, as well as reading, searching, and modifying messages via IMAP.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `to` | string | yes | — | Recipient email address |
| `subject` | string | yes | — | Email subject line |
| `body` | string | yes | — | Email body (plain text or HTML) |
| `smtpHost` | string | yes | — | SMTP server hostname |
| `smtpPort` | integer | no | `587` | SMTP server port |
| `smtpUser` | string | yes | — | SMTP authentication username |
| `smtpPass` | string | yes | — | SMTP authentication password |

## Using the Email Component

```yaml
run:
  component:
    name: email
    with:
      to: "recipient@example.com"
      subject: "Hello from KDeps"
      body: "Your workflow completed successfully."
      smtpHost: "smtp.example.com"
      smtpPort: 587
      smtpUser: "user@example.com"
      smtpPass: "app-password"
```

Access the result via `output('<callerActionId>')`. The result map includes `success`, `action`, `from`, `to`, and `attachments`.

---

## Reference: Full Email Configuration

The following sections document the full configuration surface available in the underlying email implementation (actions, IMAP, attachments, etc.).



---

## Quick Reference

| Action | Protocol | Typical use case |
|---|---|---|
| `send` | SMTP | Send transactional or notification emails |
| `read` | IMAP | Fetch recent messages from a mailbox |
| `search` | IMAP | Find messages that match sender, subject, date, or body criteria |
| `modify` | IMAP | Mark messages as read/unread, flag, move, or delete |

The `action` field defaults to `send`, so existing configurations require no changes.

---

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: sendNotification
  name: Send Notification

run:
  email:
    smtp:
      host: smtp.example.com
      port: 587
    from: noreply@example.com
    to:
      - recipient@example.com
    subject: "Hello from KDeps"
    body: "Your workflow completed successfully."
```

---

## Actions

### Send (SMTP)

The `send` action (default) delivers a plain-text or HTML email through any standard
SMTP server.  It supports STARTTLS, implicit TLS, and unauthenticated relays, and
can attach local files to the outgoing message.

```yaml
run:
  email:
    action: send   # optional — "send" is the default
    smtp:
      host: smtp.gmail.com
      port: 587
      username: "{{env('SMTP_USER')}}"
      password: "{{env('SMTP_PASS')}}"
      tls: false
    from: "{{env('SMTP_FROM')}}"
    to:
      - recipient@example.com
    cc:
      - manager@example.com
    subject: "Your report is ready"
    body: "Please find your report attached."
    attachments:
      - "{{get('report_pdf')}}"
    timeoutDuration: 30s
```

#### SMTP Configuration

```yaml
smtp:
  host: smtp.gmail.com      # Required
  port: 587                 # Optional (default: 587 for STARTTLS, 465 for TLS)
  username: user@gmail.com  # Optional
  password: app-password    # Optional
  tls: false                # true = implicit TLS (port 465); false = STARTTLS opportunistic (port 587)
  # startTLS: deprecated and ignored; STARTTLS is attempted opportunistically when tls: false
  insecureSkipVerify: false # Skip TLS certificate verification (dev/test only)
```

| Field | Type | Description |
|---|---|---|
| `host` | string | **Required.** SMTP server hostname. |
| `port` | int | SMTP port. Defaults to `465` when `tls: true`, `587` otherwise. |
| `username` | string | SMTP authentication username. Omit for unauthenticated servers. |
| `password` | string | SMTP authentication password. Supports <span v-pre>`{{env(...)}}`</span>. |
| `tls` | bool | `true` = implicit TLS (SMTPS, port 465). `false` = opportunistic STARTTLS (port 587). |
| `startTLS` | bool | Ignored. STARTTLS is always attempted opportunistically when `tls: false`. |
| `insecureSkipVerify` | bool | Skip TLS certificate verification. **Do not use in production.** |

#### Connection Modes

**STARTTLS (default, port 587)**

The client connects over plain TCP then upgrades to TLS using the `STARTTLS`
command if the server advertises it.  This is the recommended mode for most
providers (Gmail, SendGrid, Mailgun, etc.).

```yaml
smtp:
  host: smtp.gmail.com
  port: 587
  username: "{{env('SMTP_USER')}}"
  password: "{{env('SMTP_PASS')}}"
  tls: false   # default
```

**Implicit TLS (port 465)**

The connection is established over TLS from the very first byte.  Required by
some older or strict mail servers (SMTPS).

```yaml
smtp:
  host: mail.example.com
  port: 465
  tls: true
```

#### HTML Emails

Set `html: true` to send the body as `text/html`:

```yaml
email:
  smtp:
    host: smtp.example.com
  from: no-reply@example.com
  to:
    - customer@example.com
  subject: "Your report is ready"
  html: true
  body: |
    <!DOCTYPE html>
    <html>
    <body>
      <h1>Match Report</h1>
      <p>Candidate: <strong>Jane Smith</strong></p>
      <p>Score: <strong>87%</strong></p>
    </body>
    </html>
```

#### File Attachments

List local file paths under `attachments`. Each file is read at send time
and encoded as a base64 `application/octet-stream` MIME part:

```yaml
email:
  smtp:
    host: smtp.example.com
  from: reports@example.com
  to:
    - manager@example.com
  subject: "Match Report — Jane Smith"
  body: "Please see the attached report."
  attachments:
    - "{{get('report_pdf')}}"
    - /tmp/motivation-letter.pdf
```

#### Send Result Map

When the email is sent successfully, `Execute` returns:

```json
{
  "success": true,
  "action": "send",
  "from": "sender@example.com",
  "to": ["r1@example.com", "r2@example.com"],
  "cc": ["cc@example.com"],
  "subject": "Your subject",
  "attachments": 1
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the email was accepted by the server. |
| `action` | string | Always `"send"`. |
| `from` | string | Evaluated sender address. |
| `to` | []string | Evaluated To recipients. |
| `cc` | []string | Evaluated CC recipients (empty slice if none). |
| `subject` | string | Evaluated subject line. |
| `attachments` | int | Number of files attached. |

Access these fields from downstream resources:

```yaml
# In a later resource:
run:
  apiResponse:
    success: true
    response:
      email_sent: "{{get('sendEmail.success')}}"
```

---

### Read (IMAP)

The `read` action connects to an IMAP server and fetches the most recent messages
from a mailbox.  Use `limit` to control how many messages are returned and
`markRead` to automatically mark fetched messages as seen.

```yaml
run:
  email:
    action: read
    imap:
      host: imap.gmail.com
      username: "{{env('IMAP_USER')}}"
      password: "{{env('IMAP_PASS')}}"
      tls: true
    mailbox: INBOX
    limit: 20
    markRead: false
    timeoutDuration: 30s
```

#### IMAP Configuration

| Field | Type | Description |
|---|---|---|
| `host` | string | **Required.** IMAP server hostname. |
| `port` | int | IMAP port. Defaults to `993` when `tls: true`, `143` otherwise. |
| `username` | string | IMAP authentication username. |
| `password` | string | IMAP authentication password. Supports <span v-pre>`{{env(...)}}`</span>. |
| `tls` | bool | `true` = implicit TLS (default). `false` = plain connection. |
| `insecureSkipVerify` | bool | Skip TLS certificate verification. **Do not use in production.** |

#### Read Options

| Field | Type | Description |
|---|---|---|
| `mailbox` | string | Mailbox/folder to read from. Default: `"INBOX"`. |
| `limit` | int | Maximum number of messages to return (most recent first). Default: `10`. |
| `markRead` | bool | When `true`, marks each fetched message as seen. Default: `false`. |

#### Read Result Map

```json
{
  "success": true,
  "action": "read",
  "mailbox": "INBOX",
  "count": 3,
  "messages": [
    {
      "uid": 42,
      "messageId": "<abc123@mail.gmail.com>",
      "from": "sender@example.com",
      "to": ["you@example.com"],
      "subject": "Hello",
      "date": "2024-03-14T09:00:00Z",
      "body": "Message body text...",
      "seen": false
    }
  ]
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the operation completed without error. |
| `action` | string | Always `"read"`. |
| `mailbox` | string | The mailbox that was read. |
| `count` | int | Number of messages returned. |
| `messages` | []object | Array of message objects (see fields below). |

**EmailMessage fields:**

| Field | Type | Description |
|---|---|---|
| `uid` | uint32 | IMAP UID of the message. |
| `messageId` | string | `Message-ID` header value. |
| `from` | string | Sender address. |
| `to` | []string | Recipient addresses. |
| `subject` | string | Subject line. |
| `date` | string | Message date in RFC3339 format. |
| `body` | string | Plain-text body (or HTML body if no plain-text part exists). |
| `seen` | bool | `true` if the message has the `\Seen` flag set. |

---

### Search (IMAP)

The `search` action finds messages in a mailbox that match one or more criteria.
Criteria are ANDed together — only messages satisfying all specified criteria are
returned.  Use `limit` to cap the result set.

```yaml
run:
  email:
    action: search
    imap:
      host: imap.gmail.com
      username: "{{env('IMAP_USER')}}"
      password: "{{env('IMAP_PASS')}}"
    mailbox: INBOX
    limit: 50
    search:
      from: "invoices@supplier.com"
      subject: "Invoice"
      since: "2024-01-01"
      unseen: true
```

#### Search Criteria

| Field | Type | Description |
|---|---|---|
| `from` | string | Match messages where the From header contains this value. |
| `subject` | string | Match messages where the Subject header contains this value. |
| `since` | string | Match messages on or after this date. Accepts `YYYY-MM-DD` or RFC3339. |
| `before` | string | Match messages before this date. Accepts `YYYY-MM-DD` or RFC3339. |
| `unseen` | bool | When `true`, only return messages without the `\Seen` flag. |
| `body` | string | Match messages whose body contains this text. |

All criteria are optional; omitting all criteria returns the most recent messages up
to `limit`.

#### Search Result Map

```json
{
  "success": true,
  "action": "search",
  "mailbox": "INBOX",
  "count": 2,
  "messages": [...]
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the search completed without error. |
| `action` | string | Always `"search"`. |
| `mailbox` | string | The mailbox that was searched. |
| `count` | int | Number of messages returned. |
| `messages` | []object | Array of message objects (same fields as [Read](#read-imap)). |

---

### Modify (IMAP)

The `modify` action changes the flags of messages, moves them to another mailbox,
or marks them for deletion.  Messages to modify are identified either by explicit
`uids` or by a `search` criteria block (the same criteria as the `search` action).
Both can be combined — the effective set is the intersection.

**Example 1 — Mark messages as read by UID:**

```yaml
run:
  email:
    action: modify
    imap:
      host: imap.gmail.com
      username: "{{env('IMAP_USER')}}"
      password: "{{env('IMAP_PASS')}}"
    mailbox: INBOX
    uids:
      - "{{get('email_uid')}}"
    modify:
      markSeen: true
```

**Example 2 — Move matching emails to an archive folder:**

```yaml
run:
  email:
    action: modify
    imap:
      host: imap.gmail.com
      username: "{{env('IMAP_USER')}}"
      password: "{{env('IMAP_PASS')}}"
    mailbox: INBOX
    search:
      from: "newsletters@example.com"
    modify:
      moveTo: "[Gmail]/Archive"
```

**Example 3 — Delete and expunge:**

```yaml
run:
  email:
    action: modify
    imap:
      host: imap.gmail.com
      username: "{{env('IMAP_USER')}}"
      password: "{{env('IMAP_PASS')}}"
    mailbox: INBOX
    search:
      before: "2023-01-01"
    modify:
      markDeleted: true
      expunge: true
```

#### Modify Configuration

| Field | Type | Description |
|---|---|---|
| `uids` | []string | Explicit IMAP UIDs to modify. Supports expressions per item. |
| `modify.markSeen` | *bool | `true` = mark messages as read; `false` = mark as unread. Omit to leave unchanged. |
| `modify.markFlagged` | *bool | `true` = flag/star messages; `false` = remove flag. Omit to leave unchanged. |
| `modify.markDeleted` | *bool | `true` = mark messages with `\Deleted`; `false` = remove the deleted flag. |
| `modify.moveTo` | string | Destination mailbox/folder name. The message is copied then deleted from the source. |
| `modify.expunge` | bool | When `true`, issues an `EXPUNGE` command after flag changes to permanently remove deleted messages. |

#### Modify Result Map

```json
{
  "success": true,
  "action": "modify",
  "mailbox": "INBOX",
  "count": 5,
  "uids": [101, 102, 103, 104, 105]
}
```

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when all modifications completed without error. |
| `action` | string | Always `"modify"`. |
| `mailbox` | string | The source mailbox. |
| `count` | int | Number of messages modified. |
| `uids` | []uint32 | UIDs of all messages that were modified. |

---

## Expression Support

All string fields and per-item list entries across every action are evaluated using
the KDeps expression engine.  This lets you inject runtime values from previous steps,
environment variables, or request parameters:

```yaml
email:
  smtp:
    host: "{{env('SMTP_HOST')}}"
    port: 587
    username: "{{env('SMTP_USER')}}"
    password: "{{env('SMTP_PASS')}}"
  from: "{{env('SMTP_FROM')}}"
  to:
    - "{{get('distribution_list')}}"
  subject: "[CV Match] {{get('candidate_name')}} — {{get('job_title')}} ({{get('score_pct')}}%)"
  body: "{{get('email_html')}}"
  html: true
  attachments:
    - "{{get('report_pdf')}}"
```

IMAP fields support expressions the same way:

```yaml
email:
  action: search
  imap:
    host: "{{env('IMAP_HOST')}}"
    username: "{{env('IMAP_USER')}}"
    password: "{{env('IMAP_PASS')}}"
  search:
    from: "{{get('sender_filter')}}"
    since: "{{get('start_date')}}"
```

---

## Email as an Inline Resource

Send an email **before** or **after** the main resource action using the
`before` / `after` inline blocks.  This is useful for notifications without
making email the primary step:

```yaml
run:
  after:
    - email:
        smtp:
          host: "{{env('SMTP_HOST')}}"
        from: noreply@example.com
        to:
          - ops@example.com
        subject: "PDF generated"
        body: "The match report PDF is ready at: {{get('report_pdf')}}"

  pdf:
    content: "{{get('report_html')}}"
    backend: wkhtmltopdf
```

---

## Provider Quick-Reference

### Send (SMTP)

| Provider | Host | Port | TLS |
|---|---|---|---|
| Gmail | `smtp.gmail.com` | 587 | false (STARTTLS) |
| Gmail (SMTPS) | `smtp.gmail.com` | 465 | true |
| SendGrid | `smtp.sendgrid.net` | 587 | false |
| Mailgun | `smtp.mailgun.org` | 587 | false |
| Amazon SES | `email-smtp.<region>.amazonaws.com` | 587 | false |
| Postmark | `smtp.postmarkapp.com` | 587 | false |
| Mailchimp Transactional | `smtp.mandrillapp.com` | 587 | false |

> **Gmail note:** Use an [App Password](https://support.google.com/accounts/answer/185833)
> rather than your account password. Enable 2-Step Verification first.

### Read / Search / Modify (IMAP)

| Provider | IMAP Host | Port | TLS |
|---|---|---|---|
| Gmail | `imap.gmail.com` | 993 | true |
| Outlook / Microsoft 365 | `outlook.office365.com` | 993 | true |
| Yahoo | `imap.mail.yahoo.com` | 993 | true |
| iCloud | `imap.mail.me.com` | 993 | true |
| Custom / Self-hosted | `your-server.com` | 993 | true |

> **Gmail note:** IMAP must be enabled in Gmail Settings → See all settings → Forwarding
> and POP/IMAP. Use an App Password for authentication.

---

## Full Example: Invoice Processing Pipeline

This pipeline searches an inbox for unread invoices, processes the data with an LLM
step, and then marks the emails as read and archives them.

```yaml
# Step 1: Search for unread invoice emails
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: read-invoices
    name: Read Invoice Emails

  run:
    email:
      action: search
      imap:
        host: imap.gmail.com
        username: "{{env('IMAP_USER')}}"
        password: "{{env('IMAP_PASS')}}"
      mailbox: INBOX
      search:
        from: "billing@supplier.com"
        subject: "Invoice"
        unseen: true
      limit: 10

# Step 2: LLM processes invoice data
# (uses get('read-invoices.messages') to access the fetched messages)
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: process-invoices
    name: Extract Invoice Data

  run:
    llm:
      prompt: |
        Extract the invoice number, amount, and due date from each of the
        following emails and return the results as JSON.

        Emails: {{get('read-invoices.messages')}}

# Step 3: Mark processed emails as read and move them to the archive folder
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: archive-invoices
    name: Archive Invoice Emails

  run:
    email:
      action: modify
      imap:
        host: imap.gmail.com
        username: "{{env('IMAP_USER')}}"
        password: "{{env('IMAP_PASS')}}"
      mailbox: INBOX
      search:
        from: "billing@supplier.com"
        subject: "Invoice"
        unseen: true
      modify:
        markSeen: true
        moveTo: "Invoices/Processed"
```

---

## Full Example: CV Matcher Distribution Email

This is the `send-email` step from the `cv-matcher` example.  It sends an HTML
summary with the match-report PDF attached to a distribution list:

```yaml
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: send-email
    name: Email Distribution

  validations:
    skip: "{{get('compute-match.is_match') == false}}"

  run:
    email:
      smtp:
        host: "{{env('SMTP_HOST')}}"
        port: 587
        username: "{{env('SMTP_USERNAME')}}"
        password: "{{env('SMTP_PASSWORD')}}"
      from: "{{env('SMTP_FROM')}}"
      to:
        - "{{get('distribution_list')}}"
      subject: >-
        [CV Match] {{get('extract-cv.name')}} —
        {{get('extract-jd.title')}}
        ({{get('compute-match.score_pct')}}%)
      html: true
      body: |
        <!DOCTYPE html>
        <html>
        <body style="font-family:sans-serif">
          <h2>CV / JD Match Report</h2>
          <table>
            <tr><td><strong>Candidate</strong></td><td>{{get('extract-cv.name')}}</td></tr>
            <tr><td><strong>Position</strong></td><td>{{get('extract-jd.title')}}</td></tr>
            <tr><td><strong>Match Score</strong></td><td>{{get('compute-match.score_pct')}}%</td></tr>
          </table>
          <h3>Download</h3>
          <ul>
            <li><a href="{{get('upload-s3.url')}}">Match Report (S3)</a></li>
            <li><a href="{{get('upload-gdrive.url')}}">Motivation Letter (Drive)</a></li>
          </ul>
        </body>
        </html>
      attachments:
        - "{{get('generate-report-pdf.outputFile')}}"
      timeoutDuration: 30s
```
