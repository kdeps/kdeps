# Approval tokens

When the [permission mode](/agent/tools#permission-modes) denies a tool call, the agent can request a one-time exception with an approval token. A token grants one tool+action override without relaxing the permission mode. Only you grant it: the agent can request, list and revoke tokens, never grant one.

## How it works in practice

1. You run with `KDEPS_PERMISSION_MODE=read-only`
2. The agent attempts a write operation (e.g. `bash_exec rm -rf /tmp/cache`)
3. `PermissionEnforcer` blocks the call
4. The agent calls `approval_request(tool=bash_exec, action="rm -rf /tmp/cache")` - creates a `pending` token with scope `{ToolName:"bash_exec", Action:"rm -rf /tmp/cache"}`
5. The agent tells you the token id (or `/approve` lists the pending ones):
   ```
   Pending approval:
     apt-1: tool=bash_exec action="rm -rf /tmp/cache" status=pending
   ```
6. You run `/approve apt-1`
7. The agent retries the tool call - `BeforeToolCall` finds the granted token via `FindMatchingGranted`, consumes it (one-time use), and lets the call proceed

## CLI tools

| Tool | Description |
|------|-------------|
| `approval_request` | Create a pending token for a specific tool+action scope |
| `approval_grant` | Grant a pending token. Not offered to the model: you grant with `/approve <token_id>` |
| `approval_list` | List all tokens with status |
| `approval_revoke` | Revoke a granted or pending token |

## Lifecycle

- **Pending** - created by the agent when a tool call is denied, waiting for your approval
- **Granted** - you approved the exception with `/approve`
- **Consumed** - the token was used for one tool call and is now spent
- **Expired** - TTL elapsed without being consumed (default 5 minutes)
- **Revoked** - manually revoked via `approval_revoke`

Scope matching supports wildcards: an empty `Action` matches any action. `FindMatchingGranted(toolName, action, now)` is called automatically in the `BeforeToolCall` hook - you never call it directly.

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- [Built-in tools](/agent/tools) - permission modes and the full tool catalog
