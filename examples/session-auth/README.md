# Session Management Example

This example demonstrates session management with persistent SQLite storage in KDeps v2.

## Features

- SQLite-backed persistent session storage
- Session TTL (time-to-live) configuration
- Python-based credential validation
- Session data storage and retrieval
- Automatic cleanup of expired sessions

## Session Configuration

Sessions are configured in `workflow.yaml`:

```yaml
settings:
  session:
    type: "sqlite"              # Storage type: "sqlite" or "memory"
    path: ".kdeps/sessions.db"  # Database path (relative or absolute)
    ttl: "30m"                  # Session expiration time
    cleanupInterval: "5m"       # Cleanup interval for expired sessions
```

### Configuration Options

- **type**: `"sqlite"` (persistent) or `"memory"` (in-memory, lost on restart)
- **path**: Path to SQLite database file (default: `~/.kdeps/sessions.db`)
- **ttl**: Session expiration time in Go duration format (e.g., `"30m"`, `"1h"`, `"24h"`)
- **cleanupInterval**: How often to clean up expired sessions (e.g., `"5m"`, `"1h"`)

## Run Locally

```bash
# From examples/session-auth directory
kdeps run workflow.yaml --dev

# Or from root
kdeps run examples/session-auth/workflow.yaml --dev
```

## Test

### 1. Login (Validate Credentials)

```bash
# Save cookie for session persistence
curl -X POST http://localhost:16395/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "secret"}' \
  -c cookies.txt
```

**Response (success)**:
```json
{
  "success": true,
  "data": {
    "endpoint": "/api/v1/login",
    "login": {"data": {"message": "Login successful", "user_id": "admin"}, "success": true},
    "session": {"logged_in": "true", "login_time": "2026-01-30T08:14:56Z", "user_id": "admin"},
    "user_id": "admin",
    "logged_in": "true",
    "login_time": "2026-01-30T08:14:56Z"
  }
}
```

**Response (invalid credentials)**:
```bash
curl -X POST http://localhost:16395/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "wrong", "password": "wrong"}'
```

```json
{
  "success": false,
  "error": {"code": 500, "message": "Invalid credentials"}
}
```

### 2. Access Session Data (with cookie)

```bash
# Use the saved cookie to access session
curl http://localhost:16395/api/v1/session -b cookies.txt
```

**Response**:
```json
{
  "success": true,
  "data": {
    "endpoint": "/api/v1/session",
    "session": {"logged_in": "true", "login_time": "2026-01-30T08:14:56Z", "user_id": "admin"},
    "user_id": "admin",
    "logged_in": "true",
    "login_time": "2026-01-30T08:14:56Z"
  }
}
```

### 3. Access Session Without Login (401 Unauthorized)

```bash
# Without cookie - returns 401
curl http://localhost:16395/api/v1/session
```

**Response**:
```json
{
  "success": false,
  "error": {"code": 401, "message": "Not logged in. Please login first."}
}
```

## Structure

```
session-auth/
├── workflow.yaml                    # Workflow with session configuration
└── resources/
    ├── login-handler.yaml          # Python validates, sets session on success
    ├── session-handler.yaml        # Returns session data (requires login)
    └── session-response.yaml       # API response (target)
```

## Resources

### loginHandler (login-handler.yaml)

Python validates credentials. If valid, exits with 0 and expr blocks set session data.
If invalid, exits with error code 1 to prevent session from being set.

```yaml
run:
  restrictToRoutes: [/api/v1/login]
  validation:
    required: [username, password]
  python:
    script: |
      import json
      import sys
      username = "{{ get('username') }}"
      password = "{{ get('password') }}"
      if username == "admin" and password == "secret":
          print(json.dumps({"success": True, "user": username}))
      else:
          print(json.dumps({"error": "Invalid credentials"}), file=sys.stderr)
          sys.exit(1)
  # These only run if Python succeeds (exit 0)
  expr:
    - "{{ set('user_id', get('username'), 'session') }}"
    - "{{ set('logged_in', 'true', 'session') }}"
    - "{{ set('login_time', info('current_time'), 'session') }}"
```

### sessionHandler (session-handler.yaml)

Returns session data. Requires user to be logged in:

```yaml
run:
  restrictToRoutes: [/api/v1/session]
  preflightCheck:
    validations:
      - "{{ get('logged_in', 'session') == 'true' }}"
    error:
      code: 401
      message: "Not logged in. Please login first."
  python:
    script: |
      import json
      session_data = {{ session() }}
      print(json.dumps({"session": session_data}))
```

### sessionResponse (session-response.yaml)

Target resource that combines outputs:

```yaml
run:
  apiResponse:
    success: true
    response:
      endpoint: "{{ get('path') }}"
      login: "{{ output('loginHandler') }}"
      session: "{{ output('sessionHandler') }}"
```

## Key Concepts

### Expression Positioning (exprBefore, expr, exprAfter)

Control when expression blocks execute relative to the primary execution type:

```yaml
run:
  # Runs BEFORE the primary execution type
  exprBefore:
    - "{{ set('start_time', info('current_time')) }}"

  python:
    script: |
      # Your Python code here

  # Runs AFTER the primary execution type (default)
  expr:
    - "{{ set('user_id', get('username'), 'session') }}"

  # Also runs AFTER (alias for expr)
  exprAfter:
    - "{{ set('completed', 'true') }}"
```

### Combining apiResponse with Primary Types

You can combine `apiResponse` with primary execution types (python, chat, sql, etc.):

```yaml
run:
  python:
    script: |
      import json
      print(json.dumps({"result": "success"}))

  # apiResponse runs after the primary type and uses its output
  apiResponse:
    success: true
    response:
      message: "Operation completed"
      python_output: "{{ output('currentResource') }}"
```

### Python Credential Validation

Use Python scripts to validate credentials against any backend:

```yaml
python:
  script: |
    import json
    # Validate against database, LDAP, etc.
    if validate_user(username, password):
        print(json.dumps({"success": True}))
    else:
        print(json.dumps({"success": False, "error": "Invalid"}))
```

### Session Function

Access all session data using the `session()` function:

```yaml
python:
  script: |
    session_data = {{ session() }}
```

### Checking Current Session

There are multiple ways to check the current user's session:

```yaml
# 1. Get all session data as a map
session_data: "{{ session() }}"
# Returns: {"user_id": "admin", "logged_in": "true", "login_time": "..."}

# 2. Get specific session values
user_id: "{{ get('user_id', 'session') }}"
logged_in: "{{ get('logged_in', 'session') }}"

# 3. Use in preflightCheck to require login
preflightCheck:
  validations:
    - "{{ get('logged_in', 'session') == 'true' }}"
  error:
    code: 401
    message: "Not logged in"

# 4. Use in Python scripts
python:
  script: |
    import json
    session = {{ session() }}
    user_id = session.get('user_id')
    if user_id:
        print(json.dumps({"user": user_id, "authenticated": True}))

# 5. Skip resource if not logged in
skipCondition:
  - "{{ get('logged_in', 'session') != 'true' }}"
```

### Route Restrictions

Resources only execute when their route matches:

```yaml
run:
  restrictToRoutes: [/api/v1/login]  # Only runs for this route
```

### Output Function

Access output from other resources:

```yaml
response:
  login: "{{ output('loginHandler') }}"
```

## Demo Credentials

- **Username**: `admin`
- **Password**: `secret`

## Session Storage Details

### SQLite Schema

Sessions are stored in SQLite with the following schema:

```sql
CREATE TABLE sessions (
    session_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    accessed_at INTEGER NOT NULL,
    expires_at INTEGER,
    PRIMARY KEY (session_id, key)
);
```

### Persistent vs In-Memory

- **SQLite** (`type: "sqlite"`): Persists across restarts, stored on disk
- **Memory** (`type: "memory"`): Lost on restart, faster for development

## Use Cases

1. **User Authentication**: Validate credentials with Python
2. **Session Management**: Store and retrieve session data
3. **Multi-Step Forms**: Store form data across requests
4. **User Preferences**: Store user settings temporarily
5. **API Rate Limiting**: Track API usage per session
