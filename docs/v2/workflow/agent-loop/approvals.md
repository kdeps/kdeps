# Approval tokens

When the [permission mode](/workflow/agent-loop/tools#permission-modes) denies a tool call, the agent can request a one-time exception with an approval token. A token grants one tool+action override. The permission mode does not change. Only you grant it: the agent can request, list and revoke tokens, never grant one.

## How it works in practice

1. You run with `KDEPS_PERMISSION_MODE=read-only`
2. The agent attempts a write operation (e.g. `bash_exec rm -rf /tmp/cache`)
3. `PermissionEnforcer` blocks the call
4. The agent calls `approval_request(tool=bash_exec, action="rm -rf /tmp/cache")` - creates a `pending` token with scope `{ToolName:"bash_exec", Action:"rm -rf /tmp/cache"}`
5. The agent tells you the token id, for example `apt-1`.
6. You grant it through the [kdeps API](/reference/api):
   ```bash
   curl -sS -X POST -H "Authorization: Bearer $KDEPS_API_TOKEN" \
     -H "Content-Type: application/json" --data '{"token_id":"apt-1"}' \
     "http://$KDEPS_API_LISTEN/apis/tools.kdeps.io/v1/tools/approval_grant/call"
   ```
7. The agent tries the tool call again. `BeforeToolCall` finds the granted token with `FindMatchingGranted`, uses it (one time only), and lets the call continue

## CLI tools

| Tool | Description |
|------|-------------|
| `approval_request` | Create a pending token for a specific tool+action scope |
| `approval_grant` | Grant a pending token. Not offered to the model: you call it on the kdeps API |
| `approval_list` | List all tokens with status |
| `approval_revoke` | Revoke a granted or pending token |

## Lifecycle

- **Pending** - the agent creates it when kdeps denies a tool call. It waits for your approval
- **Granted** - you approved the exception with `approval_grant`
- **Consumed** - one tool call used the token. You cannot use it again
- **Expired** - the TTL ended before a tool call used the token (default 5 minutes)
- **Revoked** - you revoked it with `approval_revoke`

Scope matching supports wildcards: an empty `Action` matches any action. The `BeforeToolCall` hook calls `FindMatchingGranted(toolName, action, now)` automatically. You do not call it directly.

## See also

- [Agent loop](/workflow/agent-loop/) - overview
- [Built-in tools](/workflow/agent-loop/tools) - permission modes and the full tool catalog
