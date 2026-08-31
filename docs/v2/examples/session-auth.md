# Add login and sessions to an API

*Applies to workflow mode.*

## Overview

In this tutorial you build an API with two endpoints: `POST /login` checks a
username and password and starts a session; `GET /session` returns the session
data but only for a logged-in caller. Session state persists in SQLite.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic Python
- How cookies and sessions work at a high level

By the end you will be able to:

- Configure SQLite-backed sessions with a TTL
- Write session values with `set(key, value, 'session')`
- Read them with `get(key, 'session')`
- Gate a resource on a session value with `validations.check`

## Background

A session is per-caller key/value storage that survives across requests. kdeps
issues a session id automatically and tracks it with a cookie. With
`session.type: sqlite`, session data is written to a file, so it survives a
server restart.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

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

## Summary

You built an API that:

- Persists sessions to SQLite with a 30-minute TTL
- Writes session values in an `after:` block with `set(..., 'session')`
- Reads them with `get(..., 'session')` and `session()`
- Blocks unauthenticated calls with a `validations.check` on a session value

## Next steps

- [Session and memory](/configuration/session) - full session configuration
- [Validation and control flow](/concepts/validation-and-control) - `check`, `required`, `error`
- [Python resource](/resources/scripting/python) - packages, virtual environments
- [Unified API](/concepts/unified-api) - `get`, `set`, storage scopes
