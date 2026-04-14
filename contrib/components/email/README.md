# email

> Send email via Python smtplib (SMTP with STARTTLS).

The email component sends a plain-text email through any SMTP server that supports STARTTLS. Use it to deliver notifications, reports, or alerts from your workflows.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `to` | string | yes | — | Recipient email address |
| `subject` | string | yes | — | Email subject line |
| `body` | string | yes | — | Email body (plain text) |
| `smtpHost` | string | yes | — | SMTP server hostname (e.g. `smtp.gmail.com`) |
| `smtpPort` | integer | no | `587` | SMTP server port |
| `smtpUser` | string | yes | — | SMTP username / sender email address |
| `smtpPass` | string | yes | — | SMTP password or app password |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| `sent` | string | `"true"` when the email was sent without error |

## Usage

Send a notification email:

```yaml
run:
  component:
    name: email
    with:
      to: "team@example.com"
      subject: "Build succeeded"
      body: "Pipeline completed successfully at {{ output('buildStep').timestamp }}"
      smtpHost: smtp.gmail.com
      smtpUser: "{{ get('env.SMTP_USER') }}"
      smtpPass: "{{ get('env.SMTP_PASS') }}"
  apiResponse:
    success: true
    response:
      emailSent: "{{ output('sendAlert').sent }}"
```

Using a custom port (e.g. port 465 for SSL — note: this component uses STARTTLS, not implicit SSL):

```yaml
run:
  component:
    name: email
    with:
      to: recipient@example.com
      subject: Monthly Report
      body: "{{ output('reportStep').text }}"
      smtpHost: mail.example.com
      smtpPort: 587
      smtpUser: "{{ get('env.MAIL_USER') }}"
      smtpPass: "{{ get('env.MAIL_PASS') }}"
```

## Requirements

- Python stdlib only (`smtplib`, `email.mime`) — no additional packages needed.
- An SMTP server accessible from the workflow runtime.

## Common SMTP Providers

| Provider | Host | Port | Notes |
|----------|------|------|-------|
| Gmail | `smtp.gmail.com` | `587` | Use an [App Password](https://myaccount.google.com/apppasswords) |
| Outlook/Hotmail | `smtp.office365.com` | `587` | |
| SendGrid | `smtp.sendgrid.net` | `587` | Use `apikey` as username |
| Mailgun | `smtp.mailgun.org` | `587` | |
| Self-hosted | your hostname | `587` | |

## Notes

- Store `smtpPass` in an environment variable — never hardcode credentials in workflow YAML.
- This component uses STARTTLS on the configured port. It does not support implicit SSL (port 465) directly.
- HTML email is not supported by this component. Use `run.python:` with `MIMEText(body, "html")` for HTML emails.
- To send to multiple recipients, call the component once per recipient or modify via a custom Python resource.
