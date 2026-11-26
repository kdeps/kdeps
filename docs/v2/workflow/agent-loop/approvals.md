# Approval tokens

When the [permission mode](/reference/agent-loop/tools#permission-modes) denies a tool call, the agent can request a one-time exception with an approval token. Only you can grant a token.

You grant a token through the kdeps API:

```bash
curl -sS -X POST -H "Authorization: Bearer $KDEPS_API_TOKEN" \
  -H "Content-Type: application/json" --data '{"token_id":"apt-1"}' \
  "http://$KDEPS_API_LISTEN/apis/tools.kdeps.io/v1/tools/approval_grant/call"
```

## Read next

- [Approval tokens reference](/reference/agent-loop/approvals) - every key, option and detail
- [Agent loop](/workflow/agent-loop/) - overview
- [Built-in tools](/workflow/agent-loop/tools) - permission modes and the full tool catalog
