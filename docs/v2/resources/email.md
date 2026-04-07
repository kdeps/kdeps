# Email Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install email`
>
> Usage: `run: { component: { name: email, with: { to: "...", subject: "...", body: "...", smtpHost: "...", smtpUser: "...", smtpPass: "..." } } }`

The Email component sends email via SMTP using Python `smtplib`.

> **Note**: The component supports **sending only**. For IMAP read/search/modify operations, use a Python resource with the `imaplib` library directly.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `to` | string | yes | — | Recipient email address |
| `subject` | string | yes | — | Email subject line |
| `body` | string | yes | — | Email body (plain text) |
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

Access the result via `output('<callerActionId>')`.

---

## Result Map

| Field | Type | Description |
|---|---|---|
| `success` | bool | `true` when the email was accepted by the server. |

---

## Provider Quick-Reference

| Provider | Host | Port |
|---|---|---|
| Gmail (STARTTLS) | `smtp.gmail.com` | 587 |
| SendGrid | `smtp.sendgrid.net` | 587 |
| Mailgun | `smtp.mailgun.org` | 587 |
| Amazon SES | `email-smtp.<region>.amazonaws.com` | 587 |
| Postmark | `smtp.postmarkapp.com` | 587 |

> **Gmail**: Use an [App Password](https://support.google.com/accounts/answer/185833) rather than your account password.

---

## Expression Support

All fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: email
    with:
      to: "{{ get('recipient') }}"
      subject: "[Report] {{ get('candidate_name') }} - {{ get('score') }}%"
      body: "{{ get('email_body') }}"
      smtpHost: "{{ env('SMTP_HOST') }}"
      smtpPort: 587
      smtpUser: "{{ env('SMTP_USER') }}"
      smtpPass: "{{ env('SMTP_PASS') }}"
```

</div>

---

## Full Example: CV Matcher Distribution Email

<div v-pre>

```yaml
- apiVersion: kdeps.io/v1
  kind: Resource

  metadata:
    actionId: send-email
    name: Email Distribution

  validations:
    skip: "{{ get('compute-match.is_match') == false }}"

  run:
    component:
      name: email
      with:
        to: "{{ get('distribution_list') }}"
        subject: >-
          [CV Match] {{ get('extract-cv.name') }} -
          {{ get('extract-jd.title') }}
          ({{ get('compute-match.score_pct') }}%)
        body: |
          CV/JD Match Report

          Candidate: {{ get('extract-cv.name') }}
          Position: {{ get('extract-jd.title') }}
          Match Score: {{ get('compute-match.score_pct') }}%
        smtpHost: "{{ env('SMTP_HOST') }}"
        smtpPort: 587
        smtpUser: "{{ env('SMTP_USERNAME') }}"
        smtpPass: "{{ env('SMTP_PASSWORD') }}"
```

</div>
