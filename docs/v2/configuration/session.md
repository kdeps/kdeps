# Session Configuration

Session storage enables persistent data storage across multiple requests for the same user.

## Overview

KDeps supports two session storage backends:

| Type | Persistence | Use Case |
|------|-------------|----------|
| `sqlite` | Persistent (file-based) | Production, multi-container |
| `memory` | In-memory only | Development, single instance |

## Configuration

```yaml
settings:
  session:
    type: sqlite                    # "sqlite" or "memory"
    path: ".kdeps/sessions.db"      # SQLite file path
    ttl: "30m"                      # Session expiration
    cleanupInterval: "5m"           # Cleanup frequency
```

## Session Types

### SQLite (Recommended for Production)

```yaml
settings:
  session:
    type: sqlite
    path: "/data/sessions.db"   # Absolute or relative path
    ttl: "24h"                  # 24 hour sessions
    cleanupInterval: "1h"       # Cleanup hourly
```

Benefits:
- Survives container restarts
- Can be shared across instances (with shared volume)
- Reliable for production use

### Memory

```yaml
settings:
  session:
    type: memory
    ttl: "30m"
    cleanupInterval: "5m"
```

Benefits:
- Fast (no disk I/O)
- Simple (no external dependencies)
- Good for development

Limitations:
- Lost on restart
- Not shared across instances

## TTL (Time To Live)

Session expiration durations:

```yaml
ttl: "30s"    # 30 seconds
ttl: "5m"     # 5 minutes
ttl: "1h"     # 1 hour
ttl: "24h"    # 24 hours
ttl: "7d"     # 7 days (168h)
```

## Using Sessions

### Store Data

```yaml
run:
  expr:
    # Store user ID
    - set('user_id', '123', 'session')

    # Store preferences
    - set('theme', 'dark', 'session')

    # Increment counter
    - set('visits', get('visits', 'session') + 1, 'session')
```

### Retrieve Data

<div v-pre>

```yaml
run:
  chat:
    prompt: |
      User {{ get('user_id', 'session') }} prefers {{ get('theme', 'session') }}.
      This is visit #{{ get('visits', 'session') }}.
```

</div>

## Examples

### Login Session

```yaml
# Login resource
metadata:
  actionId: login

run:
  preflightCheck:
    validations:
      - get('username') != ''
      - get('password') != ''
    error:
      code: 400
      message: Username and password required

  expr:
    # Validate credentials (simplified)
    - set('authenticated', get('username') == 'admin', 'session')
    - set('user', get('username'), 'session')
    - set('login_time', info('timestamp'), 'session')

  apiResponse:
    success: get('authenticated', 'session')
    response:
      message: <span v-pre>"{{ get('authenticated', 'session') ? 'Login successful' : 'Invalid credentials' }}"</span>
```

### Protected Route

<div v-pre>

```yaml
metadata:
  actionId: protectedResource

run:
  preflightCheck:
    validations:
      - get('authenticated', 'session') == true
    error:
      code: 401
      message: Authentication required

  chat:
    prompt: "Hello {{ get('user', 'session') }}, how can I help?"
```

</div>

### Shopping Cart

```yaml
# Add to cart
metadata:
  actionId: addToCart

run:
  expr:
    - set('cart', get('cart', 'session') + [get('item')], 'session')

  apiResponse:
    success: true
    response:
      cart: get('cart', 'session')
      count: len(get('cart', 'session'))

---
# View cart
metadata:
  actionId: viewCart

run:
  apiResponse:
    success: true
    response:
      items: get('cart', 'session')
      total: sum(get('cart', 'session').price)
```

### User Preferences

```yaml
# Save preferences
metadata:
  actionId: savePrefs

run:
  expr:
    - set('language', get('language', 'en'), 'session')
    - set('timezone', get('timezone', 'UTC'), 'session')
    - set('theme', get('theme', 'light'), 'session')

  apiResponse:
    success: true
    response:
      message: Preferences saved

---
# Apply preferences
metadata:
  actionId: getContent

run:
  chat:
    prompt: |
<div v-pre>
      Respond in {{ get('language', 'session') }}.
      Current time in {{ get('timezone', 'session') }}: {{ info('timestamp') }}
</div>


```

## Session ID

Each session has a unique ID accessible via:

```yaml
session_id: info('sessionId')
```

The session ID is typically stored in a cookie or passed as a header.

## Cleanup

Expired sessions are automatically cleaned up based on `cleanupInterval`:

```yaml
session:
  ttl: "30m"              # Sessions expire after 30 minutes
  cleanupInterval: "5m"   # Check for expired sessions every 5 minutes
```

## Docker Considerations

### Persistent Volume

For SQLite sessions in Docker:

```yaml
# docker-compose.yml
services:
  myagent:
    volumes:
      - session_data:/data

volumes:
  session_data:
```

```yaml
# workflow.yaml
settings:
  session:
    type: sqlite
    path: "/data/sessions.db"
```

### Shared Sessions

For multiple containers sharing sessions:

1. Use a shared volume for SQLite
2. Or use an external session store (Redis, PostgreSQL)

## Best Practices

1. **Use SQLite for production** - Survives restarts
2. **Set appropriate TTL** - Balance security and convenience
3. **Store minimal data** - Session storage is not a database
4. **Handle missing sessions** - Always provide defaults
5. **Secure sensitive data** - Don't store passwords in sessions

## Security Notes

- Sessions are identified by a unique ID
- Session IDs should be transmitted securely (HTTPS)
- Implement logout by clearing session data
- Set reasonable TTL to limit exposure

## Next Steps

- [Workflow Configuration](workflow.md) - Full settings reference
- [Unified API](../concepts/unified-api.md) - get() and set() usage
- [Docker Deployment](../deployment/docker.md) - Production setup
