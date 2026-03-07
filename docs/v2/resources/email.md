# Email Resource

The email resource sends an email (plain-text or HTML) with optional file attachments
via any standard SMTP server.  It supports three connection modes — plain SMTP,
STARTTLS, and implicit TLS — and can be used as a primary resource or as an
[inline resource](../concepts/inline-resources) inside `before` / `after` blocks.

All string fields support [KDeps expressions](../concepts/expressions) such as
`{{get(...)}}` and `{{env(...)}}`.

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

## Configuration Options

| Option | Type | Description |
|---|---|---|
| `smtp` | object | **Required.** SMTP server settings. See [SMTP Configuration](#smtp-configuration). |
| `from` | string | **Required.** Sender email address. Supports expressions. |
| `to` | []string | **Required.** One or more recipient addresses. Supports expressions per item. |
| `cc` | []string | Carbon-copy recipients. Supports expressions per item. |
| `bcc` | []string | Blind carbon-copy recipients (added to SMTP envelope only, not message headers). Supports expressions per item. |
| `subject` | string | **Required.** Email subject line. Supports expressions. |
| `body` | string | **Required.** Email body. Plain text by default; set `html: true` for HTML. Supports expressions. |
| `html` | bool | When `true`, the body is sent as `text/html`. Default: `false`. |
| `attachments` | []string | File paths to attach. Each file is base64-encoded as `application/octet-stream`. Supports expressions per item. |
| `timeoutDuration` | string | Dial + send timeout as a Go duration (e.g. `30s`, `2m`). Default: `30s`. |
| `timeout` | string | Alias for `timeoutDuration`. |

### SMTP Configuration

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
| `password` | string | SMTP authentication password. Supports `{{env(...)}}`. |
| `tls` | bool | `true` = implicit TLS (SMTPS, port 465). `false` = opportunistic STARTTLS (port 587). |
| `startTLS` | bool | **Deprecated.** Ignored by the executor. STARTTLS is always attempted opportunistically when `tls: false`. Retained for backward compatibility. |
| `insecureSkipVerify` | bool | Skip TLS certificate verification. **Do not use in production.** |

---

## Connection Modes

### STARTTLS (default, port 587)

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

### Implicit TLS (port 465)

The connection is established over TLS from the very first byte.  Required by
some older or strict mail servers (SMTPS).

```yaml
smtp:
  host: mail.example.com
  port: 465
  tls: true
```

---

## HTML Emails

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

---

## File Attachments

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

---

## Expression Support

All string fields and per-item list entries are evaluated using the KDeps
expression engine.  This lets you inject runtime values from previous steps,
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

---

## Result Map

When the email is sent successfully, `Execute` returns:

```json
{
  "success": true,
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

---

## Provider Quick-Reference

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
