# Add login and sessions to an API

Build an API with two endpoints: `POST /login` checks a
username and password and starts a session. `GET /session` returns the session
data but only for a logged-in caller. Session state persists in SQLite.

Only a logged-in caller passes this check:

<div v-pre>

```yaml
# resources/session.yaml (part)
validations:
  check:
    - "{{ get('logged_in', 'session') == 'true' }}"
  error:
    code: 401
    message: "not logged in"
```

</div>

## Read next

- [Full walkthrough](/reference/examples/session-auth) - every step, file and command
- [Session and memory](/workflow/sessions) - full session configuration
- [Validation and control flow](/workflow/validation) - `check`, `required`, `error`
