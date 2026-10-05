# Add login and sessions to an API

Build an API with two endpoints: `POST /login` checks a
username and password and starts a session; `GET /session` returns the session
data but only for a logged-in caller. Session state persists in SQLite.

A session is per-caller key/value storage that survives across requests. kdeps
issues a session id automatically and tracks it with a cookie. With
`session.type: sqlite`, session data is written to a file, so it survives a
server restart.

## Step 1: create the project

```bash
mkdir session-api
cd session-api
mkdir resources
```

## Step 2: configure sessions

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: session-api
  version: "1.0.0"
  targetActionId: sessionResponse

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/login
        methods: [POST]
      - path: /api/v1/session
        methods: [GET, POST]
  agentSettings:
    pythonVersion: "3.12"
  session:
    type: sqlite                 # persist to a file
    path: .kdeps/sessions.db     # relative to the workflow directory
    ttl: 30m                     # session expires 30 minutes after last use
    cleanupInterval: 5m          # sweep expired sessions every 5 minutes
```

## Step 3: handle login

Create `resources/login.yaml`:

<div v-pre>

```yaml
# resources/login.yaml
actionId: loginHandler
name: Login handler
validations:
  routes: [/api/v1/login]
  required:
    - username                  # 400 if either field is missing
    - password
python:
  script: |
    import json, sys
    username = "{{ get('username') }}"
    password = "{{ get('password') }}"
    # Demo credentials: admin / secret
    if username == "admin" and password == "secret":
        print(json.dumps({"success": True, "user": username}))
    else:
        print(json.dumps({"success": False, "error": "invalid credentials"}), file=sys.stderr)
        sys.exit(1)              # non-zero exit fails the resource, no session set
  timeout: 10s
after:
  - "set('user_id', get('username'), 'session')"
  - "set('logged_in', 'true', 'session')"
apiResponse:
  success: true
  response:
    message: "login successful"
    user_id: "{{ get('user_id', 'session') }}"
    session_id: "{{ info('session_id') }}"
```

</div>

The `after:` block runs only if the Python script exits 0. It writes three
values into the session.

## Step 4: guard the session endpoint

Create `resources/session.yaml`:

<div v-pre>

```yaml
# resources/session.yaml
actionId: sessionHandler
name: Session handler
validations:
  routes: [/api/v1/session]
  check:
    - "{{ get('logged_in', 'session') == 'true' }}"   # must be logged in
  error:
    code: 401
    message: "not logged in"
apiResponse:
  success: true
  response:
    user_id: "{{ get('user_id', 'session') }}"
    logged_in: "{{ get('logged_in', 'session') }}"
    all_session: "{{ session() }}"
```

</div>

`session()` returns the whole session as an object.

## Step 5: wire the target resource

Create `resources/response.yaml`:

<div v-pre>

```yaml
# resources/response.yaml
actionId: sessionResponse
name: Response
requires: [loginHandler, sessionHandler]
apiResponse:
  success: true
  response:
    endpoint: "{{ get('path') }}"
    user_id: "{{ get('user_id', 'session') }}"
```

</div>

Only one of `loginHandler` / `sessionHandler` runs per request; the other is
skipped by its `routes` validation.

## Step 6: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

Log in, saving the session cookie:

```bash
curl -c cookies.txt -X POST http://localhost:16395/api/v1/login \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "secret"}'
```

Call the guarded endpoint with the cookie:

```bash
curl -b cookies.txt http://localhost:16395/api/v1/session \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN"
```

Without the cookie the same call returns `401 not logged in`.

## Next steps

- [Session and memory](/workflow/sessions) - full session configuration
- [Validation and control flow](/workflow/validation) - `check`, `required`, `error`
- [Python resource](/workflow/resources/python) - packages, virtual environments
- [Unified API](/workflow/data-access) - `get`, `set`, storage scopes
