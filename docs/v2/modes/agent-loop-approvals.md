# Approval tokens

When a tool call is denied by the [permission mode](/modes/agent-loop-tools#permission-modes), the agent can request a one-time exception via an approval token. Tokens let you grant scoped overrides for specific tool+action combinations without relaxing the overall permission mode.

## How it works in practice

1. You run with `KDEPS_PERMISSION_MODE=read-only`
2. The agent attempts a write operation (e.g. `bash_exec rm -rf /tmp/cache`)
3. `PermissionEnforcer` blocks the call
4. The agent calls `approval_request(tool=bash_exec, action="rm -rf /tmp/cache")` - creates a `pending` token with scope `{ToolName:"bash_exec", Action:"rm -rf /tmp/cache"}`
5. The agent calls `approval_list` to show you the pending token:
   ```
   Pending approval:
     apt-1: tool=bash_exec action="rm -rf /tmp/cache" status=pending
   ```
6. You run `/run approval_grant token_id=apt-1`
7. The agent retries the tool call - `BeforeToolCall` finds the granted token via `FindMatchingGranted`, consumes it (one-time use), and lets the call proceed

## CLI tools

| Tool | Description |
|------|-------------|
| `approval_request` | Create a pending token for a specific tool+action scope |
| `approval_grant` | Grant a pending token (user approves) |
| `approval_list` | List all tokens with status |
| `approval_revoke` | Revoke a granted or pending token |

## Lifecycle

- **Pending** - created by the agent when a tool call is denied, waiting for your approval
- **Granted** - you approved the exception via `approval_grant`
- **Consumed** - the token was used for one tool call and is now spent
- **Expired** - TTL elapsed without being consumed (default 5 minutes)
- **Revoked** - manually revoked via `approval_revoke`

Scope matching supports wildcards: an empty `Action` matches any action. `FindMatchingGranted(toolName, action, now)` is called automatically in the `BeforeToolCall` hook - you never call it directly.

## See also

- [Agent Loop Mode](/modes/agent-loop-mode) - overview and starting the REPL
- [Built-in Tools](/modes/agent-loop-tools) - permission modes and the full tool catalog
